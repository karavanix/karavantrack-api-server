package common

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/render"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/users"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/users/command"
	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
)

type usersHandler struct {
	cfg          *config.Config
	validator    *validation.Validator
	usersUsecase *users.Usecase
}

func NewUsersHandler(opts *delivery.HandlerOptions) *usersHandler {
	return &usersHandler{
		cfg:          opts.Config,
		validator:    opts.Validator,
		usersUsecase: opts.UsersUsecase,
	}
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
func (h *usersHandler) GetMe() http.HandlerFunc {
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
func (h *usersHandler) UpdateMe() http.HandlerFunc {
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
func (h *usersHandler) RegisterDevice() http.HandlerFunc {
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
