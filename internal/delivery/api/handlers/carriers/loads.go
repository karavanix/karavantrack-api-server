package carriers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-playground/form/v4"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads/query"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/location"
	locationcmd "github.com/karavanix/karavantrack-api-server/internal/usecase/location/command"
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

// LatestAssigned godoc
// @Security     BearerAuth
// @Summary      Get latest assigned load
// @Tags         Loads
// @Produce      json
// @Param        limit      query int    false "Pagination Limit"
// @Param        offset     query int    false "Pagination Offset"
// @Success      200 {object} query.ListResponse
// @Failure      400 {object} outerr.Response
// @Failure      401 {object} outerr.Response
// @Failure      403 {object} outerr.Response
// @Failure      404 {object} outerr.Response
// @Failure      500 {object} outerr.Response
// @Router       /loads/pending [get]
func (h *loadsHandler) ListPending() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		var urlForm query.ListRequest
		if err := form.NewDecoder().Decode(&urlForm, r.URL.Query()); err != nil {
			outerr.BadRequest(w, r, "invalid request form: "+err.Error())
			return
		}

		urlForm.CarrierID = userID
		urlForm.Status = []string{domain.LoadStatusAssigned.String()}

		var resp *query.ListResponse
		resp, err := h.loadsUsecase.Query.List(r.Context(), &urlForm)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp)
	}
}

// GetActive godoc
// @Security     BearerAuth
// @Summary      Get active load
// @Description  Get the current active load for the authenticated carrier (accepted, in_transit, or completed)
// @Tags         Loads
// @Produce      json
// @Success      200 {object} query.LoadResponse
// @Failure      401 {object} outerr.Response
// @Failure      403 {object} outerr.Response
// @Failure      404 {object} outerr.Response
// @Failure      500 {object} outerr.Response
// @Router       /loads/active [get]
func (h *loadsHandler) GetActive() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		resp, err := h.loadsUsecase.Query.GetActive(r.Context(), userID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp)
	}
}

// Accept godoc
// @Security     BearerAuth
// @Summary      Accept load
// @Description  Accept an assigned load (by carrier)
// @Tags         Loads
// @Produce      json
// @Param        id   path      string  true  "Load ID"
// @Success      200  {object} map[string]string
// @Failure      401  {object} outerr.Response
// @Router       /loads/{id}/accept [post]
func (h *loadsHandler) Accept() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		loadID := chi.URLParam(r, "id")

		if err := h.loadsUsecase.Command.Accept(r.Context(), loadID, userID); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, map[string]string{"status": "accepted"})
	}
}

// Start godoc
// @Security     BearerAuth
// @Summary      Start load
// @Description  Start a load (by carrier, transitions to in-transit)
// @Tags         Loads
// @Produce      json
// @Param        id   path      string  true  "Load ID"
// @Success      200
// @Failure      401  {object} outerr.Response
// @Router       /loads/{id}/start [post]
func (h *loadsHandler) Start() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loadID := chi.URLParam(r, "id")

		if err := h.loadsUsecase.Command.Start(r.Context(), loadID); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
	}
}

// BeginPickup godoc
// @Security     BearerAuth
// @Summary      Begin pickup
// @Description  Carrier is driving to pickup location (accepted → picking_up)
// @Tags         Loads
// @Produce      json
// @Param        id   path      string  true  "Load ID"
// @Success      200
// @Failure      401  {object} outerr.Response
// @Router       /loads/{id}/pickup/begin [post]
func (h *loadsHandler) BeginPickup() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loadID := chi.URLParam(r, "id")

		if err := h.loadsUsecase.Command.BeginPickup(r.Context(), loadID); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
	}
}

// ConfirmPickup godoc
// @Security     BearerAuth
// @Summary      Confirm pickup
// @Description  Cargo loaded onto truck (picking_up → picked_up)
// @Tags         Loads
// @Produce      json
// @Param        id   path      string  true  "Load ID"
// @Success      200
// @Failure      401  {object} outerr.Response
// @Router       /loads/{id}/pickup/confirm [post]
func (h *loadsHandler) ConfirmPickup() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loadID := chi.URLParam(r, "id")

		if err := h.loadsUsecase.Command.ConfirmPickup(r.Context(), loadID); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
	}
}

// BeginDropoff godoc
// @Security     BearerAuth
// @Summary      Begin dropoff
// @Description  Carrier arrived at destination (in_transit → dropping_off)
// @Tags         Loads
// @Produce      json
// @Param        id   path      string  true  "Load ID"
// @Success      200
// @Failure      401  {object} outerr.Response
// @Router       /loads/{id}/dropoff/begin [post]
func (h *loadsHandler) BeginDropoff() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loadID := chi.URLParam(r, "id")

		if err := h.loadsUsecase.Command.BeginDropoff(r.Context(), loadID); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
	}
}

// ConfirmDropoff godoc
// @Security     BearerAuth
// @Summary      Confirm dropoff
// @Description  Cargo unloaded at destination (dropping_off → dropped_off)
// @Tags         Loads
// @Produce      json
// @Param        id   path      string  true  "Load ID"
// @Success      200
// @Failure      401  {object} outerr.Response
// @Router       /loads/{id}/dropoff/confirm [post]
func (h *loadsHandler) ConfirmDropoff() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loadID := chi.URLParam(r, "id")

		if err := h.loadsUsecase.Command.ConfirmDropoff(r.Context(), loadID); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
	}
}

// RegisterLocation godoc
// @Security     BearerAuth
// @Summary      Register location point
// @Description  Register a GPS location point for an in-transit load (MVP REST alternative to WebSocket)
// @Tags         Loads
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Load ID"
// @Param        body body locationcmd.RegisterLoadLocationRequest true "Location data"
// @Success      200
// @Failure      400  {object} outerr.Response
// @Failure      401  {object} outerr.Response
// @Router       /loads/{id}/location [post]
func (h *loadsHandler) RegisterLocation() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		loadID := chi.URLParam(r, "id")

		var req locationcmd.RegisterLoadLocationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			outerr.BadRequest(w, r, "invalid request body")
			return
		}

		req.LoadID = loadID
		req.CarrierID = userID

		if err := h.locationUsecase.Command.RegisterLoadLocation(r.Context(), &req); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
	}
}
