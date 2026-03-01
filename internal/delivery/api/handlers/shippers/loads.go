package shippers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads/command"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/location"
	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
)

type loadsHandler struct {
	cfg             *config.Config
	validator       *validation.Validator
	loadsUsecase    *loads.Usecase
	locationUsecase *location.Usecase
}

func NewLoadsHandler(opts *delivery.HandlerOptions) *loadsHandler {
	return &loadsHandler{
		cfg:             opts.Config,
		validator:       opts.Validator,
		loadsUsecase:    opts.LoadsUsecase,
		locationUsecase: opts.LocationUsecase,
	}
}

// Create godoc
// @Security     BearerAuth
// @Summary      Create load
// @Description  Create a new load
// @Tags         Loads
// @Accept       json
// @Produce      json
// @Param        body body command.CreateRequest true "Create Load Request"
// @Success      201  {object} command.CreateResponse
// @Failure      400  {object} outerr.Response
// @Failure      401  {object} outerr.Response
// @Router       /loads [post]
func (h *loadsHandler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		var req command.CreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			outerr.BadRequest(w, r, "invalid request body")
			return
		}
		if err := h.validator.Validate(req); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		resp, err := h.loadsUsecase.Command.Create(r.Context(), userID, &req)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, resp)
	}
}

// Assign godoc
// @Security     BearerAuth
// @Summary      Assign load
// @Description  Assign load to a carrier and truck
// @Tags         Loads
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Load ID"
// @Param        body body command.AssignRequest true "Assign Load Request"
// @Success      200  {object} map[string]string
// @Failure      400  {object} outerr.Response
// @Failure      401  {object} outerr.Response
// @Router       /loads/{id}/assign [post]
func (h *loadsHandler) Assign() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loadID := chi.URLParam(r, "id")

		var req command.AssignRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			outerr.BadRequest(w, r, "invalid request body")
			return
		}
		if err := h.validator.Validate(req); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		if err := h.loadsUsecase.Command.Assign(r.Context(), loadID, &req); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, map[string]string{"status": "assigned"})
	}
}

// Confirm godoc
// @Security     BearerAuth
// @Summary      Confirm load delivery
// @Description  Confirm a completed load delivery (by company/dispatcher)
// @Tags         Loads
// @Produce      json
// @Param        id   path      string  true  "Load ID"
// @Success      200
// @Failure      401  {object} outerr.Response
// @Router       /loads/{id}/confirm [post]
func (h *loadsHandler) Confirm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loadID := chi.URLParam(r, "id")

		if err := h.loadsUsecase.Command.Confirm(r.Context(), loadID); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
	}
}

// Cancel godoc
// @Security     BearerAuth
// @Summary      Cancel load
// @Description  Cancel a load
// @Tags         Loads
// @Produce      json
// @Param        id   path      string  true  "Load ID"
// @Success      200
// @Failure      401  {object} outerr.Response
// @Router       /loads/{id}/cancel [post]
func (h *loadsHandler) Cancel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loadID := chi.URLParam(r, "id")

		if err := h.loadsUsecase.Command.Cancel(r.Context(), loadID); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
	}
}

// GetTrack godoc
// @Security     BearerAuth
// @Summary      Get load track
// @Description  Get location history tracking for a load
// @Tags         Loads
// @Produce      json
// @Param        id     path  string true  "Load ID"
// @Param        limit  query int    false "Max number of points (default 100, max 1000)"
// @Param        offset query int    false "Pagination offset"
// @Success      200  {object} query.GetTrackResponse
// @Failure      400  {object} outerr.Response
// @Failure      401  {object} outerr.Response
// @Failure      404  {object} outerr.Response
// @Router       /loads/{id}/track [get]
func (h *loadsHandler) GetTrack() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loadID := chi.URLParam(r, "id")
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

		resp, err := h.loadsUsecase.Query.GetTrack(r.Context(), loadID, limit, offset)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}

// GetPosition godoc
// @Security     BearerAuth
// @Summary      Get current position
// @Description  Get current tracked position for a load (latest GPS point)
// @Tags         Loads
// @Produce      json
// @Param        id   path      string  true  "Load ID"
// @Success      200  {object} query.PositionResponse
// @Failure      400  {object} outerr.Response
// @Failure      401  {object} outerr.Response
// @Failure      404  {object} outerr.Response
// @Router       /loads/{id}/position [get]
func (h *loadsHandler) GetPosition() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loadID := chi.URLParam(r, "id")

		resp, err := h.loadsUsecase.Query.GetPosition(r.Context(), loadID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}
