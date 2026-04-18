package carriers

import (
	"github.com/go-chi/chi/v5"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/middleware"
	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
)

// @title Carriers API
// @BasePath /api/v1
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description					API Токен используется для авторизации
func RegisterRoutes(r chi.Router, opts *delivery.HandlerOptions) {
	loadsH := NewLoadsHandler(opts)
	companyH := NewCompanyHandler(opts)

	// Carrier-specific routes (RoleCarrier only)
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthorizeByRole(opts.JWTProvider, shared.RoleCarrier))

		// Company actions
		r.Get("/carriers/companies/{id}", companyH.GetCarrierCompany())

		// Load actions
		r.Get("/loads/pending", loadsH.ListPending())
		r.Get("/loads/active", loadsH.GetActive())
		r.Post("/loads/{id}/accept", loadsH.Accept())
		r.Post("/loads/{id}/pickup/begin", loadsH.BeginPickup())
		r.Post("/loads/{id}/pickup/confirm", loadsH.ConfirmPickup())
		r.Post("/loads/{id}/start", loadsH.Start())
		r.Post("/loads/{id}/dropoff/begin", loadsH.BeginDropoff())
		r.Post("/loads/{id}/dropoff/confirm", loadsH.ConfirmDropoff())
		r.Post("/loads/{id}/location", loadsH.RegisterLocation())
	})
}
