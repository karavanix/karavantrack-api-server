package app

import (
	"context"
	"fmt"
	"time"

	pkgapp "github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/infrastructure/persistence/repository"
	"github.com/karavanix/karavantrack-api-server/internal/service/notification"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/karavanix/karavantrack-api-server/pkg/firebase"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/uptrace/bun"
)

const gpsTimeoutThreshold = 15 * time.Minute

type GpsNotifyApp struct {
	config              *config.Config
	db                  *bun.DB
	loadsRepo           domain.LoadRepository
	notificationService notification.Service
}

func NewGpsNotifyApp(cfg *config.Config) (*GpsNotifyApp, error) {
	db, err := postgres.NewBunDB(
		postgres.WithHost(cfg.DB.Host),
		postgres.WithPort(cfg.DB.Port),
		postgres.WithUser(cfg.DB.User),
		postgres.WithPassword(cfg.DB.Password),
		postgres.WithDB(cfg.DB.Name),
		postgres.WithSSLMode(cfg.DB.Sslmode),
		postgres.WithDebug(cfg.LogLevel == pkgapp.Debug),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	fcmClient, err := firebase.New(context.Background(), cfg)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to create firebase client: %w", err)
	}

	fcmDevicesRepo := repository.NewFCMDevicesRepo(db)
	notificationService := notification.NewService(fcmClient, fcmDevicesRepo)

	return &GpsNotifyApp{
		config:              cfg,
		db:                  db,
		loadsRepo:           repository.NewLoadsRepo(db),
		notificationService: notificationService,
	}, nil
}

func (a *GpsNotifyApp) Run(ctx context.Context) error {
	_, _ = logger.NewLogger("", a.config.LogLevel)

	loads, err := a.loadsRepo.FindWithStaleGps(ctx, gpsTimeoutThreshold)
	if err != nil {
		return fmt.Errorf("failed to fetch stale-GPS loads: %w", err)
	}

	logger.Info("stale GPS loads found", "count", len(loads))

	for _, load := range loads {
		notifyErr := a.notificationService.SendToUser(ctx, load.CarrierID.String(), &firebase.Notification{
			Title: "GPS не получен",
			Body:  "Нет обновления местоположения более 15 минут: " + load.Title,
			Metadata: map[string]string{
				"load_id": load.ID.String(),
				"action":  "gps_timeout",
			},
		})
		if notifyErr != nil {
			logger.Error("failed to send GPS timeout notification", notifyErr,
				"load_id", load.ID.String(),
				"carrier_id", load.CarrierID.String(),
			)
		}
	}

	return nil
}

func (a *GpsNotifyApp) Close() {
	if a.db != nil {
		_ = a.db.Close()
	}
}
