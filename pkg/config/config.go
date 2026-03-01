package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/karavanix/karavantrack-api-server/pkg/app"
)

type Config struct {
	APP         string
	Environment app.Environment
	LogLevel    app.LogLevel

	Server struct {
		Host         string
		Port         string
		ReadTimeout  time.Duration
		WriteTimeout time.Duration
		IdleTimeout  time.Duration
	}

	Admin struct {
		Username string
		Password string
	}

	Context struct {
		Timeout time.Duration
	}

	DB struct {
		Host     string
		Port     string
		Name     string
		User     string
		Password string
		Sslmode  string
	}

	Redis struct {
		Host     string
		Port     string
		Password string
	}

	JWT struct {
		AccessSecret  string
		RefreshSecret string
		AccessTTL     time.Duration
		RefreshTTL    time.Duration
	}

	Firebase struct {
		AuthKey string
	}

	OTEL struct {
		ServiceName string
		Propagators string
		Exporter    struct {
			Type string
			OTLP struct {
				Endpoint string
				Protocol string
			}
		}
		Traces struct {
			Sampler    string
			SamplerArg string
		}
	}

	Nats struct {
		Host            string
		Port            string
		Username        string
		Password        string
		StaticSubjects  struct{}
		DynamicSubjects struct {
			WebsocketConnection      string
			LoadLocationPointCreated string
		}
	}
}

func New() (*Config, error) {
	c := &Config{}

	// App & env
	c.APP = getEnv("APP", "karavantruck-api-server")
	c.Environment = app.Environment(getEnv("ENVIRONMENT", app.Local.String()))
	if !c.Environment.IsValid() {
		return nil, fmt.Errorf("invalid environment: %s", c.Environment)
	}
	c.LogLevel = app.LogLevel(getEnv("LOG_LEVEL", app.Debug.String()))
	if !c.LogLevel.IsValid() {
		return nil, fmt.Errorf("invalid log level: %s", c.LogLevel)
	}

	// Server
	c.Server.Host = getEnv("SERVER_HOST", "0.0.0.0")
	c.Server.Port = getEnv("SERVER_PORT", "8090")

	var err error
	if c.Server.ReadTimeout, err = getEnvDuration("SERVER_READ_TIMEOUT", "5s"); err != nil {
		return nil, fmt.Errorf("SERVER_READ_TIMEOUT: %w", err)
	}
	if c.Server.WriteTimeout, err = getEnvDuration("SERVER_WRITE_TIMEOUT", "10s"); err != nil {
		return nil, fmt.Errorf("SERVER_WRITE_TIMEOUT: %w", err)
	}
	if c.Server.IdleTimeout, err = getEnvDuration("SERVER_IDLE_TIMEOUT", "120s"); err != nil {
		return nil, fmt.Errorf("SERVER_IDLE_TIMEOUT: %w", err)
	}

	// Admin
	c.Admin.Username = getEnv("ADMIN_USERNAME", "")
	c.Admin.Password = getEnv("ADMIN_PASSWORD", "")

	// Context
	if c.Context.Timeout, err = getEnvDuration("CONTEXT_TIMEOUT", "30s"); err != nil {
		return nil, fmt.Errorf("CONTEXT_TIMEOUT: %w", err)
	}

	// Database
	c.DB.Host = getEnv("DB_HOST", "localhost")
	c.DB.Port = getEnv("DB_PORT", "5432")
	c.DB.Name = getEnv("DB_DATABASE", "karavantruck")
	c.DB.User = getEnv("DB_USERNAME", "postgres")
	c.DB.Password = getEnv("DB_PASSWORD", "")
	c.DB.Sslmode = getEnv("DB_SSLMODE", "disable")

	// Redis
	c.Redis.Host = getEnv("REDIS_HOST", "localhost")
	c.Redis.Port = getEnv("REDIS_PORT", "6379")
	c.Redis.Password = getEnv("REDIS_PASSWORD", "")

	// JWT
	c.JWT.AccessSecret = getEnv("JWT_ACCESS_SECRET", "secret")
	c.JWT.RefreshSecret = getEnv("JWT_REFRESH_SECRET", "secret")

	if c.JWT.AccessTTL, err = getEnvDuration("JWT_ACCESS_TTL", "15m"); err != nil {
		return nil, fmt.Errorf("JWT_ACCESS_TTL: %w", err)
	}

	if c.JWT.RefreshTTL, err = getEnvDuration("JWT_REFRESH_TTL", "670h"); err != nil {
		return nil, fmt.Errorf("JWT_REFRESH_TTL: %w", err)
	}

	// Firebase
	c.Firebase.AuthKey = getEnv("FIREBASE_AUTH_KEY", "")

	// OTLP
	c.OTEL.ServiceName = getEnv("OTEL_SERVICE_NAME", "karavantruck-api-server")
	c.OTEL.Propagators = getEnv("OTEL_PROPAGATORS", "baggage,tracecontext")
	c.OTEL.Exporter.Type = getEnv("OTEL_TRACES_EXPORTER", "otlp")
	c.OTEL.Exporter.OTLP.Endpoint = getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317")
	c.OTEL.Exporter.OTLP.Protocol = getEnv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")
	c.OTEL.Traces.Sampler = getEnv("OTEL_TRACES_SAMPLER", "traceidratio")
	c.OTEL.Traces.SamplerArg = getEnv("OTEL_TRACES_SAMPLER_ARG", "1.0")

	c.Nats.Host = getEnv("NATS_HOST", "localhost")
	c.Nats.Port = getEnv("NATS_PORT", "4222")
	c.Nats.Username = getEnv("NATS_USERNAME", "karavantruck")
	c.Nats.Password = getEnv("NATS_PASSWORD", "karavantrack-password")

	c.Nats.DynamicSubjects.WebsocketConnection = "websocket.connection.%s"
	c.Nats.DynamicSubjects.LoadLocationPointCreated = "load.location.point.%s"

	return c, nil
}

func getEnv(key string, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvDuration(key string, defaultValue string) (time.Duration, error) {
	value, err := time.ParseDuration(getEnv(key, defaultValue))
	if err != nil {
		return time.Duration(0), err
	}
	return value, nil
}
