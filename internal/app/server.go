package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v4"
	"github.com/hibiken/asynq"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/worker"
	"github.com/karavanix/karavantrack-api-server/internal/events"
	"github.com/karavanix/karavantrack-api-server/internal/infrastructure/persistence/cache"
	"github.com/karavanix/karavantrack-api-server/internal/infrastructure/persistence/repository"
	"github.com/karavanix/karavantrack-api-server/internal/service/broker"
	"github.com/karavanix/karavantrack-api-server/internal/service/notification"
	"github.com/karavanix/karavantrack-api-server/internal/service/presence"
	"github.com/karavanix/karavantrack-api-server/internal/service/rbac"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/auth"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/companies"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/location"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/users"
	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/karavanix/karavantrack-api-server/pkg/firebase"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/nats"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"github.com/karavanix/karavantrack-api-server/pkg/redis"
	"github.com/karavanix/karavantrack-api-server/pkg/security"
	"github.com/uptrace/bun"
)

type ServerApp struct {
	config       *config.Config
	logger       *logger.Logger
	server       *http.Server
	db           *bun.DB
	redis        *redis.RedisClient
	taskQueue    *asynq.Client
	taskWorker   *asynq.Server
	bkr          broker.Broker
	shutdownOTLP func(context.Context) error
}

func NewServerApp(cfg *config.Config) (*ServerApp, error) {
	shutdownOTLP := otlp.InitTracer(
		context.Background(),
		otlp.WithServiceName(cfg.OTEL.ServiceName),
		otlp.WithEnvironment(cfg.Environment),
		otlp.WithExporterType(otlp.ExporterNameToExporterType[cfg.OTEL.Exporter.Type]),
		otlp.WithEndpoint(cfg.OTEL.Exporter.OTLP.Endpoint),
		otlp.WithExporterProtocol(otlp.ExporterProtocolNameToExporterProtocolType[cfg.OTEL.Exporter.OTLP.Protocol]),
		otlp.WithSamplerType(otlp.SamplerNameToSamplerType[cfg.OTEL.Traces.Sampler]),
		otlp.WithSamplerArg(cfg.OTEL.Traces.SamplerArg),
	)

	logger, err := logger.NewLogger(cfg.APP+".log", cfg.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	db, err := postgres.NewBunDB(
		postgres.WithHost(cfg.DB.Host),
		postgres.WithPort(cfg.DB.Port),
		postgres.WithUser(cfg.DB.User),
		postgres.WithPassword(cfg.DB.Password),
		postgres.WithDB(cfg.DB.Name),
		postgres.WithSSLMode(cfg.DB.Sslmode),
		postgres.WithDebug(cfg.LogLevel == app.Debug),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	redis, err := redis.New(
		redis.WithAddress(cfg.Redis.Host+":"+cfg.Redis.Port),
		redis.WithPassword(cfg.Redis.Password),
		redis.WithKeyPrefix(cfg.APP),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create redis: %w", err)
	}

	taskQueue := asynq.NewClientFromRedisClient(redis.GetRedisClient())

	natsClient, err := nats.NewClient(
		nats.WithHost(cfg.Nats.Host),
		nats.WithPort(cfg.Nats.Port),
		nats.WithUsername(cfg.Nats.Username),
		nats.WithPassword(cfg.Nats.Password),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create nats client: %w", err)
	}

	return &ServerApp{
		config:       cfg,
		logger:       logger,
		server:       nil,
		db:           db,
		redis:        redis,
		taskQueue:    taskQueue,
		taskWorker:   nil,
		bkr:          natsClient,
		shutdownOTLP: shutdownOTLP,
	}, nil

}

func (s *ServerApp) Run() error {
	txManager := postgres.NewTxManager(s.db)
	jwtProvider := security.NewJWTProvider(
		[]byte(s.config.JWT.AccessSecret),
		[]byte(s.config.JWT.RefreshSecret),
		s.config.JWT.AccessTTL,
		s.config.JWT.RefreshTTL,
		jwt.SigningMethodHS256,
	)
	eventFactory := events.NewFactory(s.config)
	fcmClient, err := firebase.New(context.Background(), s.config)
	if err != nil {
		return fmt.Errorf("failed to create firebase client: %w", err)
	}

	// cache
	presenceRepo := cache.NewPresenceRedisStore(s.config, s.redis)

	// repository
	usersRepo := repository.NewUsersRepo(s.db)
	companiesRepo := repository.NewCompaniesRepo(s.db)
	companyMembersRepo := repository.NewCompanyMembersRepo(s.db)
	companyCarriersRepo := repository.NewCompanyCarriersRepo(s.db)
	loadsRepo := repository.NewLoadsRepo(s.db)
	loadLocationsPointsRepo := repository.NewLoadLocationPointsRepo(s.db)
	fcmDevicesRepo := repository.NewFCMDevicesRepo(s.db)

	// service
	presenceService := presence.NewService(s.config.Context.Timeout, presenceRepo)
	notificationService := notification.NewService(fcmClient, fcmDevicesRepo)
	rbacService := rbac.NewService(s.config.Context.Timeout, companyMembersRepo)

	// usecase
	authUsecase := auth.NewUsecase(s.config.Context.Timeout, jwtProvider, usersRepo)
	usersUsecase := users.NewUsecase(s.config.Context.Timeout, usersRepo, fcmDevicesRepo)
	companiesUsecase := companies.NewUsecase(s.config.Context.Timeout, txManager, companiesRepo, companyMembersRepo, companyCarriersRepo, usersRepo, loadsRepo, rbacService)
	loadsUsecase := loads.NewUsecase(s.config.Context.Timeout, loadsRepo, usersRepo, loadLocationsPointsRepo, rbacService, s.taskQueue)
	locationUsecase := location.NewUsecase(s.config.Context.Timeout, s.bkr, eventFactory, loadLocationsPointsRepo)

	// init handlers options
	opts := &delivery.HandlerOptions{
		Config:              s.config,
		Validator:           validation.NewValidator(),
		JWTProvider:         jwtProvider,
		Broker:              s.bkr,
		PresenceService:     presenceService,
		NotificationService: notificationService,
		AuthUsecase:         authUsecase,
		UsersUsecase:        usersUsecase,
		CompaniesUsecase:    companiesUsecase,
		LoadsUsecase:        loadsUsecase,
		LocationUsecase:     locationUsecase,
		RbacService:         rbacService,
	}

	mux := worker.NewRouter(opts)
	s.taskWorker = worker.NewWorker(s.config)
	go s.taskWorker.Run(mux)

	router := api.NewRouter(opts)
	s.server = api.NewServer(s.config, router)

	s.logger.Info("Listen http server:", "address", s.config.Server.Host+":"+s.config.Server.Port)
	return s.server.ListenAndServe()
}

func (s *ServerApp) Shutdown(ctx context.Context) error {
	if s.taskWorker != nil {
		s.taskWorker.Shutdown()
	}

	if s.server != nil {
		_ = s.server.Shutdown(ctx)
	}

	if s.db != nil {
		_ = s.db.Close()
	}

	if s.redis != nil {
		_ = s.redis.Close()
	}

	if s.bkr != nil {
		_ = s.bkr.Close(ctx)
	}

	if s.shutdownOTLP != nil {
		_ = s.shutdownOTLP(ctx)
	}

	return nil
}
