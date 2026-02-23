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

	// Driver profile (user with driver role)
	r.Get("/{id}", h.Get())

	// Company drivers management
	r.Post("/company/{companyId}", h.AddToCompany())
	r.Get("/company/{companyId}", h.ListByCompany())
	r.Delete("/company/{companyId}/{driverId}", h.RemoveFromCompany())

	return r
}

// Get godoc
// @Summary      Get driver
// @Description  Get driver profile by user ID (must be a driver-role user)
// @Tags         Drivers
// @Produce      json
// @Param        id   path      string  true  "Driver User ID"
// @Success      200  {object} query.DriverResponse
// @Failure      401  {object} outerr.Response
// @Failure      404  {object} outerr.Response
// @Security     BearerAuth
// @Router       /drivers/{id} [get]
func (h *handler) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		driverID := chi.URLParam(r, "id")

		resp, err := h.driversUsecase.Query.Get(r.Context(), userID, driverID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}

// AddToCompany godoc
// @Summary      Add driver to company
// @Description  Add an existing driver-role user to a company
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

		if err := h.driversUsecase.Command.AddToCompany(r.Context(), companyID, &req); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusCreated)
	}
}

// ListByCompany godoc
// @Summary      List drivers by company
// @Description  List driver-role users associated with a company
// @Tags         Drivers
// @Produce      json
// @Param        companyId   path      string  true  "Company ID"
// @Success      200  {array} query.CompanyDriverResponse
// @Failure      401  {object} outerr.Response
// @Security     BearerAuth
// @Router       /drivers/company/{companyId} [get]
func (h *handler) ListByCompany() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		companyID := chi.URLParam(r, "companyId")

		resp, err := h.driversUsecase.Query.ListByCompany(r.Context(), userID, companyID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}

// RemoveFromCompany godoc
// @Summary      Remove driver from company
// @Description  Remove a driver-role user from a company
// @Tags         Drivers
// @Produce      json
// @Param        companyId   path      string  true  "Company ID"
// @Param        driverId    path      string  true  "Driver User ID"
// @Success      200  {object} map[string]string
// @Failure      401  {object} outerr.Response
// @Security     BearerAuth
// @Router       /drivers/company/{companyId}/{driverId} [delete]
func (h *handler) RemoveFromCompany() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := chi.URLParam(r, "companyId")
		driverID := chi.URLParam(r, "driverId")

		if err := h.driversUsecase.Command.Remove(r.Context(), companyID, driverID); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
	}
}
