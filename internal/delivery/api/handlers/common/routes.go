package common

import (
	"github.com/go-chi/chi/v5"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/middleware"
)

// RegisterRoutes registers all common (role-agnostic) routes on the given router.
func RegisterRoutes(r chi.Router, opts *delivery.HandlerOptions) {
	authH := NewAuthHandler(opts)
	usersH := NewUsersHandler(opts)
	loadsH := NewLoadsHandler(opts)
	wsH := NewWSHandler(opts)

	// Auth routes (public)
	r.Post("/auth/login", authH.Login())
	r.Post("/auth/register", authH.Register())
	r.Post("/auth/logout", authH.Logout())
	r.Post("/auth/refresh", authH.Refresh())

	// Common protected routes (any authenticated user)
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthorizeAny(opts.JWTProvider))

		// Users
		r.Get("/users/me", usersH.GetMe())
		r.Put("/users/me", usersH.UpdateMe())
		r.Post("/users/me/devices", usersH.RegisterDevice())

		// Loads
		r.Get("/loads/{id}", loadsH.Get())

		// WebSocket
		r.Get("/ws", wsH.WebSocketHandler)
	})
}
