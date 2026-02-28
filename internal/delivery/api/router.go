package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	_ "github.com/karavanix/karavantrack-api-server/internal/delivery/api/docs"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/handlers/auth"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/handlers/companies"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/handlers/loads"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/handlers/users"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/handlers/ws"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// @title Karavantrack API Documentation
// @version 0.0.1
// @description Documentation of the Karavantrack API server
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @BasePath /api/v1
// @securityDefinitions.basic 	BasicAuth
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description					API Токен используется для авторизации
func NewRouter(options *delivery.HandlerOptions) http.Handler {
	router := chi.NewRouter()

	// Set real ip & recover & logger & cors & request id middlewares
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Recoverer)
	router.Use(chimiddleware.Logger)
	router.Use(chimiddleware.RequestID)
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
		r.Mount("/auth", auth.New(options))
		r.Mount("/users", users.New(options))
		r.Mount("/companies", companies.New(options))
		r.Mount("/loads", loads.New(options))
		r.Mount("/ws", ws.New(options))
	})

	// Set swagger
	router.Get("/api/swagger/*", httpSwagger.Handler())

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
