package users

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/middleware"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/users"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/users/command"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/users/query"
	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
)

type handler struct {
	cfg          *config.Config
	validator    *validation.Validator
	usersUsecase *users.Usecase
}

func New(opts *delivery.HandlerOptions) http.Handler {
	h := &handler{
		cfg:          opts.Config,
		validator:    opts.Validator,
		usersUsecase: opts.UsersUsecase,
	}

	r := chi.NewRouter()
	r.Use(middleware.AuthorizeAny(opts.JWTProvider))

	r.Get("/me", h.GetMe())
	r.Put("/me", h.UpdateMe())
	r.Post("/me/devices", h.RegisterDevice())

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthorizeByRole(opts.JWTProvider, shared.RoleShipper))
		r.Get("/carriers/search", h.SearchCarriers())
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthorizeByRole(opts.JWTProvider, shared.RoleCarrier))
		r.Get("/shippers/search", h.SearchShippers())
	})

	return r
}

// GetMe godoc
// @Security     BearerAuth
// @Summary      Get current user
// @Description  Get the profile of the authenticated user (includes role)
// @Tags         Users
// @Produce      json
// @Success      200  {object} query.MeResponse
// @Failure      401  {object} outerr.Response
// @Router       /users/me [get]
func (h *handler) GetMe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		resp, err := h.usersUsecase.Query.GetMe(r.Context(), userID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}

// UpdateMe godoc
// @Security     BearerAuth
// @Summary      Update current user
// @Description  Update the profile of the authenticated user
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        body body command.UpdateRequest true "Update User Request"
// @Success      200  {object} map[string]string
// @Failure      400  {object} outerr.Response
// @Failure      401  {object} outerr.Response
// @Router       /users/me [put]
func (h *handler) UpdateMe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		var req command.UpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			outerr.BadRequest(w, r, "invalid request body")
			return
		}

		if err := h.usersUsecase.Command.Update(r.Context(), userID, &req); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
	}
}

// SearchCarriers godoc
// @Security     BearerAuth
// @Summary      Search carriers
// @Description  Search for carrier users by name, email, or phone
// @Tags         Users
// @Produce      json
// @Param        q query string true "Search query (name, email, or phone)"
// @Success      200  {array} query.UsersResponse
// @Failure      401  {object} outerr.Response
// @Failure      403  {object} outerr.Response
// @Router       /users/carriers/search [get]
func (h *handler) SearchCarriers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")

		resp, err := h.usersUsecase.Query.SearchCarriers(r.Context(), &query.SearchCarriersRequest{
			Query: q,
		})
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}

// SearchShippers godoc
// @Security     BearerAuth
// @Summary      Search shippers
// @Description  Search for shipper users by name, email, or phone
// @Tags         Users
// @Produce      json
// @Param        q query string true "Search query (name, email, or phone)"
// @Success      200  {array} query.UsersResponse
// @Failure      401  {object} outerr.Response
// @Failure      403  {object} outerr.Response
// @Router       /users/shippers/search [get]
func (h *handler) SearchShippers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")

		resp, err := h.usersUsecase.Query.SearchShippers(r.Context(), &query.SearchShippersRequest{
			Query: q,
		})
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}

// RegisterDevice godoc
// @Security     BearerAuth
// @Summary      Register FCM device
// @Description  Register or update a device for push notifications
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        body body command.RegisterDeviceRequest true "Device registration"
// @Success      200
// @Failure      400  {object} outerr.Response
// @Failure      401  {object} outerr.Response
// @Router       /users/me/devices [post]
func (h *handler) RegisterDevice() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		var req command.RegisterDeviceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			outerr.BadRequest(w, r, "invalid request body")
			return
		}

		if err := h.usersUsecase.Command.RegisterDevice(r.Context(), userID, &req); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
	}
}
