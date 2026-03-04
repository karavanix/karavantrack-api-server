package shippers

import (
	"github.com/go-chi/chi/v5"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/middleware"
	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
)

// @title Shippers API
// @BasePath /api/v1
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description					API Токен используется для авторизации
func RegisterRoutes(r chi.Router, opts *delivery.HandlerOptions) {
	companiesH := NewCompaniesHandler(opts)
	loadsH := NewLoadsHandler(opts)
	usersH := NewUsersHandler(opts)

	// Shipper-specific routes (RoleShipper only)
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthorizeByRole(opts.JWTProvider, shared.RoleShipper))

		// Company management
		r.Post("/companies", companiesH.Create())
		r.Get("/companies/mine", companiesH.ListMine())
		r.Put("/companies/{id}", companiesH.Update())

		r.Get("/companies/{id}/loads", companiesH.ListLoads())
		r.Get("/companies/{id}/loads/stats", loadsH.GetLoadStats())

		r.Post("/companies/{id}/members", companiesH.AddMember())
		r.Get("/companies/{id}/members", companiesH.ListMembers())
		r.Delete("/companies/{id}/members/{user_id}", companiesH.RemoveMember())

		r.Get("/companies/{id}/carriers", companiesH.ListCarriers())
		r.Post("/companies/{id}/carriers", companiesH.AddCarrier())
		r.Delete("/companies/{id}/carriers/{carrier_id}", companiesH.RemoveCarrier())

		// Load actions
		r.Post("/loads", loadsH.Create())
		r.Post("/loads/{id}/assign", loadsH.Assign())
		r.Post("/loads/{id}/confirm", loadsH.Confirm())
		r.Post("/loads/{id}/cancel", loadsH.Cancel())
		r.Get("/loads/{id}/track", loadsH.GetTrack())
		r.Get("/loads/{id}/position", loadsH.GetPosition())

		// User search
		r.Get("/users/carriers/search", usersH.SearchCarriers())
		r.Get("/users/shippers/search", usersH.SearchShippers())
	})
}
