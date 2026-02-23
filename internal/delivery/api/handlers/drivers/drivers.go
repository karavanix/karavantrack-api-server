package drivers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/middleware"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/drivers"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/drivers/command"
	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
)

type handler struct {
	cfg            *config.Config
	validator      *validation.Validator
	driversUsecase *drivers.Usecase
}

func New(opts *delivery.HandlerOptions) http.Handler {
	h := &handler{
		cfg:            opts.Config,
		validator:      opts.Validator,
		driversUsecase: opts.DriversUsecase,
	}

	r := chi.NewRouter()
	r.Use(middleware.AuthContext(opts.JWTProvider))

	// Driver profile
	r.Post("/", h.Create())
	r.Get("/{id}", h.Get())

	// Company drivers management
	r.Post("/company/{companyId}", h.AddToCompany())
	r.Get("/company/{companyId}", h.ListByCompany())
	r.Delete("/company/{companyId}/{driverId}", h.RemoveFromCompany())

	return r
}

// Create godoc
// @Summary      Create driver
// @Description  Create a new driver profile for the current user
// @Tags         Drivers
// @Produce      json
// @Success      201  {object} command.CreateResponse
// @Failure      400  {object} outerr.Response
// @Failure      401  {object} outerr.Response
// @Security     BearerAuth
// @Router       /drivers [post]
func (h *handler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		resp, err := h.driversUsecase.Command.Create.Create(r.Context(), userID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, resp)
	}
}

// Get godoc
// @Summary      Get driver
// @Description  Get driver profile by ID
// @Tags         Drivers
// @Produce      json
// @Param        id   path      string  true  "Driver ID"
// @Success      200  {object} query.DriverResponse
// @Failure      401  {object} outerr.Response
// @Failure      404  {object} outerr.Response
// @Security     BearerAuth
// @Router       /drivers/{id} [get]
func (h *handler) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		driverID := chi.URLParam(r, "id")

		resp, err := h.driversUsecase.Query.Get.Get(r.Context(), driverID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}

// AddToCompany godoc
// @Summary      Add driver to company
// @Description  Add an existing driver to a company
// @Tags         Drivers
// @Accept       json
// @Produce      json
// @Param        companyId   path      string  true  "Company ID"
// @Param        body body command.AddToCompanyRequest true "Add To Company Request"
// @Success      201  {object} map[string]string
// @Failure      400  {object} outerr.Response
// @Failure      401  {object} outerr.Response
// @Security     BearerAuth
// @Router       /drivers/company/{companyId} [post]
func (h *handler) AddToCompany() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := chi.URLParam(r, "companyId")

		var req command.AddToCompanyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			outerr.BadRequest(w, r, "invalid request body")
			return
		}
		if err := h.validator.Validate(req); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		if err := h.driversUsecase.Command.AddToCompany.AddToCompany(r.Context(), companyID, &req); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, map[string]string{"status": "added"})
	}
}

// ListByCompany godoc
// @Summary      List drivers by company
// @Description  List drivers associated with a company
// @Tags         Drivers
// @Produce      json
// @Param        companyId   path      string  true  "Company ID"
// @Success      200  {array} query.CompanyDriverResponse
// @Failure      401  {object} outerr.Response
// @Security     BearerAuth
// @Router       /drivers/company/{companyId} [get]
func (h *handler) ListByCompany() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := chi.URLParam(r, "companyId")

		resp, err := h.driversUsecase.Query.ListByCompany.List(r.Context(), companyID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}

// RemoveFromCompany godoc
// @Summary      Remove driver from company
// @Description  Remove a driver from a company
// @Tags         Drivers
// @Produce      json
// @Param        companyId   path      string  true  "Company ID"
// @Param        driverId    path      string  true  "Driver ID"
// @Success      200  {object} map[string]string
// @Failure      401  {object} outerr.Response
// @Security     BearerAuth
// @Router       /drivers/company/{companyId}/{driverId} [delete]
func (h *handler) RemoveFromCompany() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := chi.URLParam(r, "companyId")
		driverID := chi.URLParam(r, "driverId")

		if err := h.driversUsecase.Command.RemoveFromCompany.Remove(r.Context(), companyID, driverID); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, map[string]string{"status": "removed"})
	}
}
