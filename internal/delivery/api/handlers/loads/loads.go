package loads

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/middleware"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads/command"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads/query"
	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
)

type handler struct {
	cfg          *config.Config
	validator    *validation.Validator
	loadsUsecase *loads.Usecase
}

func New(opts *delivery.HandlerOptions) http.Handler {
	h := &handler{
		cfg:          opts.Config,
		validator:    opts.Validator,
		loadsUsecase: opts.LoadsUsecase,
	}

	r := chi.NewRouter()
	r.Use(middleware.AuthContext(opts.JWTProvider))

	// CRUD
	r.Post("/", h.Create())
	r.Get("/", h.List())
	r.Get("/{id}", h.Get())

	// Status transitions
	r.Post("/{id}/assign", h.Assign())
	r.Post("/{id}/accept", h.Accept())
	r.Post("/{id}/start", h.Start())
	r.Post("/{id}/complete", h.Complete())
	r.Post("/{id}/confirm", h.Confirm())
	r.Post("/{id}/cancel", h.Cancel())

	// Location tracking REST endpoints
	r.Get("/{id}/track", h.GetTrack())
	r.Get("/{id}/position", h.GetPosition())

	return r
}

// Create godoc
// @Summary      Create load
// @Description  Create a new load
// @Tags         Loads
// @Accept       json
// @Produce      json
// @Param        body body command.CreateRequest true "Create Load Request"
// @Success      201  {object} command.CreateResponse
// @Failure      400  {object} outerr.Response
// @Failure      401  {object} outerr.Response
// @Security     BearerAuth
// @Router       /loads [post]
func (h *handler) Create() http.HandlerFunc {
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

		resp, err := h.loadsUsecase.Command.Create.Create(r.Context(), userID, &req)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, resp)
	}
}

// List godoc
// @Summary      List loads
// @Description  List loads with optional filters
// @Tags         Loads
// @Produce      json
// @Param        company_id query string false "Company ID filter"
// @Param        driver_id  query string false "Driver ID filter"
// @Param        status     query string false "Status filter"
// @Param        limit      query int    false "Pagination Limit"
// @Param        offset     query int    false "Pagination Offset"
// @Success      200  {array} query.LoadResponse
// @Failure      401  {object} outerr.Response
// @Security     BearerAuth
// @Router       /loads [get]
func (h *handler) List() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

		req := &query.ListRequest{
			CompanyID: r.URL.Query().Get("company_id"),
			DriverID:  r.URL.Query().Get("driver_id"),
			Status:    r.URL.Query().Get("status"),
			Limit:     limit,
			Offset:    offset,
		}

		resp, err := h.loadsUsecase.Query.List.List(r.Context(), req)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}

// Get godoc
// @Summary      Get load
// @Description  Get load by ID
// @Tags         Loads
// @Produce      json
// @Param        id   path      string  true  "Load ID"
// @Success      200  {object} query.LoadResponse
// @Failure      401  {object} outerr.Response
// @Failure      404  {object} outerr.Response
// @Security     BearerAuth
// @Router       /loads/{id} [get]
func (h *handler) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loadID := chi.URLParam(r, "id")

		resp, err := h.loadsUsecase.Query.Get.Get(r.Context(), loadID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}

// Assign godoc
// @Summary      Assign load
// @Description  Assign load to a driver and truck
// @Tags         Loads
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Load ID"
// @Param        body body command.AssignRequest true "Assign Load Request"
// @Success      200  {object} map[string]string
// @Failure      400  {object} outerr.Response
// @Failure      401  {object} outerr.Response
// @Security     BearerAuth
// @Router       /loads/{id}/assign [post]
func (h *handler) Assign() http.HandlerFunc {
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

		if err := h.loadsUsecase.Command.Assign.Assign(r.Context(), loadID, &req); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, map[string]string{"status": "assigned"})
	}
}

// Accept godoc
// @Summary      Accept load
// @Description  Accept an assigned load (by driver)
// @Tags         Loads
// @Produce      json
// @Param        id   path      string  true  "Load ID"
// @Success      200  {object} map[string]string
// @Failure      401  {object} outerr.Response
// @Security     BearerAuth
// @Router       /loads/{id}/accept [post]
func (h *handler) Accept() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loadID := chi.URLParam(r, "id")

		if err := h.loadsUsecase.Command.Accept.Accept(r.Context(), loadID); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, map[string]string{"status": "accepted"})
	}
}

// Start godoc
// @Summary      Start load
// @Description  Start a load (by driver, transitions to in-transit)
// @Tags         Loads
// @Produce      json
// @Param        id   path      string  true  "Load ID"
// @Success      200  {object} map[string]string
// @Failure      401  {object} outerr.Response
// @Security     BearerAuth
// @Router       /loads/{id}/start [post]
func (h *handler) Start() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loadID := chi.URLParam(r, "id")

		if err := h.loadsUsecase.Command.Start.Start(r.Context(), loadID); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, map[string]string{"status": "in_transit"})
	}
}

// Complete godoc
// @Summary      Complete load
// @Description  Complete a load (by driver)
// @Tags         Loads
// @Produce      json
// @Param        id   path      string  true  "Load ID"
// @Success      200  {object} map[string]string
// @Failure      401  {object} outerr.Response
// @Security     BearerAuth
// @Router       /loads/{id}/complete [post]
func (h *handler) Complete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loadID := chi.URLParam(r, "id")

		if err := h.loadsUsecase.Command.Complete.Complete(r.Context(), loadID); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, map[string]string{"status": "completed_by_driver"})
	}
}

// Confirm godoc
// @Summary      Confirm load delivery
// @Description  Confirm a completed load delivery (by company/dispatcher)
// @Tags         Loads
// @Produce      json
// @Param        id   path      string  true  "Load ID"
// @Success      200  {object} map[string]string
// @Failure      401  {object} outerr.Response
// @Security     BearerAuth
// @Router       /loads/{id}/confirm [post]
func (h *handler) Confirm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loadID := chi.URLParam(r, "id")

		if err := h.loadsUsecase.Command.Confirm.Confirm(r.Context(), loadID); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, map[string]string{"status": "confirmed"})
	}
}

// Cancel godoc
// @Summary      Cancel load
// @Description  Cancel a load
// @Tags         Loads
// @Produce      json
// @Param        id   path      string  true  "Load ID"
// @Success      200  {object} map[string]string
// @Failure      401  {object} outerr.Response
// @Security     BearerAuth
// @Router       /loads/{id}/cancel [post]
func (h *handler) Cancel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loadID := chi.URLParam(r, "id")

		if err := h.loadsUsecase.Command.Cancel.Cancel(r.Context(), loadID); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, map[string]string{"status": "cancelled"})
	}
}

// GetTrack godoc
// @Summary      Get load track
// @Description  Get location history tracking for a load
// @Tags         Loads
// @Produce      json
// @Param        id   path      string  true  "Load ID"
// @Success      200  {object} map[string]string
// @Failure      401  {object} outerr.Response
// @Security     BearerAuth
// @Router       /loads/{id}/track [get]
func (h *handler) GetTrack() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: implement location history query via LocationPointsRepository
		_ = chi.URLParam(r, "id")
		render.JSON(w, r, map[string]string{"status": "not_implemented"})
	}
}

// GetPosition godoc
// @Summary      Get current position
// @Description  Get current tracked position for a load
// @Tags         Loads
// @Produce      json
// @Param        id   path      string  true  "Load ID"
// @Success      200  {object} map[string]string
// @Failure      401  {object} outerr.Response
// @Security     BearerAuth
// @Router       /loads/{id}/position [get]
func (h *handler) GetPosition() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: implement current driver position query via LocationPointsRepository
		_ = chi.URLParam(r, "id")
		render.JSON(w, r, map[string]string{"status": "not_implemented"})
	}
}
