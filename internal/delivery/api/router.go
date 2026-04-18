package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/metrics"
	"github.com/go-chi/render"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/handlers/carriers"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/handlers/common"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/handlers/shippers"
	apimiddleware "github.com/karavanix/karavantrack-api-server/internal/delivery/api/middleware"
	"github.com/riandyrn/otelchi"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/karavanix/karavantrack-api-server/internal/delivery/api/docs/carrier"
	_ "github.com/karavanix/karavantrack-api-server/internal/delivery/api/docs/shipper"
)

func NewRouter(options *delivery.HandlerOptions) http.Handler {
	router := chi.NewRouter()

	// OpenTelemetry trace middleware
	router.Use(
		otelchi.Middleware(options.Config.APP,
			otelchi.WithFilter(func(r *http.Request) bool {
				if r.URL.Path == "/healthz" || r.URL.Path == "/metrics" || strings.HasPrefix(r.URL.Path, "/api/swagger") {
					return false
				}
				return true
			}),
			otelchi.WithChiRoutes(router),
		),
	)

	// Prometheus metrics middleware
	router.Use(
		metrics.Collector(metrics.CollectorOpts{
			Host:  false,
			Proto: true,
		}),
	)

	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Recoverer)
	router.Use(chimiddleware.RequestID)
	router.Use(apimiddleware.StructuredLogger)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Accept-Language", "Content-Type", "X-CSRF-Token", "X-Request-Id", "X-Client-Id"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Mount the handlers under the /api/v1 path
	router.Route("/api/v1", func(r chi.Router) {
		common.RegisterRoutes(r, options)
		carriers.RegisterRoutes(r, options)
		shippers.RegisterRoutes(r, options)
	})

	router.Handle("/metrics", metrics.Handler())
	router.Get("/api/swagger/carrier/*", httpSwagger.Handler(
		httpSwagger.InstanceName("carrier"),
	))
	router.Get("/api/swagger/shipper/*", httpSwagger.Handler(
		httpSwagger.InstanceName("shipper"),
	))

	router.Get("/api/swagger/*", httpSwagger.Handler(
		httpSwagger.BeforeScript(`
			var link = document.createElement('link');
			link.rel = 'stylesheet';
			link.href = 'https://cdn.jsdelivr.net/npm/swagger-ui-themes@3.0.1/themes/3.x/theme-material.css';
			document.head.appendChild(link);
		`),
		httpSwagger.UIConfig(map[string]string{
			"urls": `[
				{
					"url": "/api/swagger/carrier/doc.json",
					"name": "Carriers API"
				},
				{
					"url": "/api/swagger/shipper/doc.json",
					"name": "Shippers API"
				}
			]`,
			"filter":                 `true`,
			"displayRequestDuration": `true`,
		}),
		httpSwagger.DefaultModelsExpandDepth(httpSwagger.HideModel),
		httpSwagger.Layout(httpSwagger.StandaloneLayout),
		httpSwagger.PersistAuthorization(true),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("list"),
	))

	// Healthcheck
	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		render.JSON(w, r, map[string]any{
			"status":    "healthy",
			"timestamp": time.Now(),
			"service:":  "karavantrack-api-server",
			"version":   "1.0.0",
		})
	})

	return router
}
