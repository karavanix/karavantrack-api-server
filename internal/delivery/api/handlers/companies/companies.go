package companies

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
	"github.com/karavanix/karavantrack-api-server/internal/usecase/companies"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/companies/command"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/drivers"
	driverscmd "github.com/karavanix/karavantrack-api-server/internal/usecase/drivers/command"
	driversquery "github.com/karavanix/karavantrack-api-server/internal/usecase/drivers/query"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads/query"
	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
)

type handler struct {
	cfg              *config.Config
	validator        *validation.Validator
	companiesUsecase *companies.Usecase
	driversUsecase   *drivers.Usecase
	loadsUsecase     *loads.Usecase
}

func New(opts *delivery.HandlerOptions) http.Handler {
	h := &handler{
		cfg:              opts.Config,
		validator:        opts.Validator,
		companiesUsecase: opts.CompaniesUsecase,
		driversUsecase:   opts.DriversUsecase,
		loadsUsecase:     opts.LoadsUsecase,
	}

	r := chi.NewRouter()
	r.Use(middleware.AuthContext(opts.JWTProvider))

	r.Post("/", h.Create())
	r.Get("/", h.List())
	r.Get("/{id}", h.Get())
	r.Put("/{id}", h.Update())

	// Members sub-routes
	r.Post("/{id}/members", h.AddMember())
	r.Get("/{id}/members", h.ListMembers())
	r.Delete("/{id}/members/{userId}", h.RemoveMember())

	// Company-scoped loads and drivers
	r.Get("/{id}/loads", h.ListLoads())
	r.Get("/{id}/drivers", h.ListDrivers())
	r.Post("/{id}/drivers", h.AddDriver())

	return r
}

// Create godoc
// @Summary      Create company
// @Description  Create a new company
// @Tags         Companies
// @Accept       json
// @Produce      json
// @Param        body body command.CreateRequest true "Create Company Request"
// @Success      201  {object} command.CreateResponse
// @Failure      400  {object} outerr.Response
// @Failure      401  {object} outerr.Response
// @Failure      403  {object} outerr.Response
// @Security     BearerAuth
// @Router       /companies [post]
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

		resp, err := h.companiesUsecase.Command.Create(r.Context(), userID, &req)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, resp)
	}
}

// List godoc
// @Summary      List companies
// @Description  List companies for the current user
// @Tags         Companies
// @Produce      json
// @Success      200  {array} query.CompanyResponse
// @Failure      401  {object} outerr.Response
// @Failure      403  {object} outerr.Response
// @Security     BearerAuth
// @Router       /companies [get]
func (h *handler) List() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		resp, err := h.companiesUsecase.Query.ListByUser(r.Context(), userID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}

// Get godoc
// @Summary      Get company
// @Description  Get company by ID
// @Tags         Companies
// @Produce      json
// @Param        id   path      string  true  "Company ID"
// @Success      200  {object} query.CompanyResponse
// @Failure      401  {object} outerr.Response
// @Failure      404  {object} outerr.Response
// @Security     BearerAuth
// @Router       /companies/{id} [get]
func (h *handler) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := chi.URLParam(r, "id")

		resp, err := h.companiesUsecase.Query.Get(r.Context(), companyID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}

// Update godoc
// @Summary      Update company
// @Description  Update company details by ID (owner/admin only)
// @Tags         Companies
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Company ID"
// @Param        body body command.UpdateRequest true "Update Company Request"
// @Success      200  {object} map[string]string
// @Failure      400  {object} outerr.Response
// @Failure      401  {object} outerr.Response
// @Failure      403  {object} outerr.Response
// @Security     BearerAuth
// @Router       /companies/{id} [put]
func (h *handler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		companyID := chi.URLParam(r, "id")

		var req command.UpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			outerr.BadRequest(w, r, "invalid request body")
			return
		}
		if err := h.validator.Validate(req); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		if err := h.companiesUsecase.Command.Update(r.Context(), userID, companyID, &req); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
	}
}

// AddMember godoc
// @Summary      Add member
// @Description  Add a new member to the company (owner/admin only)
// @Tags         Companies
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Company ID"
// @Param        body body command.AddMemberRequest true "Add Member Request"
// @Success      201  {object} map[string]string
// @Failure      400  {object} outerr.Response
// @Failure      401  {object} outerr.Response
// @Failure      403  {object} outerr.Response
// @Security     BearerAuth
// @Router       /companies/{id}/members [post]
func (h *handler) AddMember() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		companyID := chi.URLParam(r, "id")

		var req command.AddMemberRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			outerr.BadRequest(w, r, "invalid request body")
			return
		}
		if err := h.validator.Validate(req); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		if err := h.companiesUsecase.Command.AddMember(r.Context(), userID, companyID, &req); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusCreated)
	}
}

// ListMembers godoc
// @Summary      List members
// @Description  List members of a company
// @Tags         Companies
// @Produce      json
// @Param        id   path      string  true  "Company ID"
// @Success      200  {array} query.MemberResponse
// @Failure      401  {object} outerr.Response
// @Security     BearerAuth
// @Router       /companies/{id}/members [get]
func (h *handler) ListMembers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := chi.URLParam(r, "id")

		resp, err := h.companiesUsecase.Query.ListMembers(r.Context(), companyID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}

// RemoveMember godoc
// @Summary      Remove member
// @Description  Remove a member from the company (owner only)
// @Tags         Companies
// @Produce      json
// @Param        id   path      string  true  "Company ID"
// @Param        userId path    string  true  "User ID"
// @Success      200  {object} map[string]string
// @Failure      401  {object} outerr.Response
// @Failure      403  {object} outerr.Response
// @Security     BearerAuth
// @Router       /companies/{id}/members/{userId} [delete]
func (h *handler) RemoveMember() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		companyID := chi.URLParam(r, "id")
		targetUserID := chi.URLParam(r, "userId")

		if err := h.companiesUsecase.Command.RemoveMember(r.Context(), userID, companyID, targetUserID); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
	}
}

// ListLoads godoc
// @Summary      List company loads
// @Description  List loads for a specific company with filters
// @Tags         Companies
// @Produce      json
// @Param        id     path  string false "Company ID"
// @Param        status query string false "Status filter"
// @Param        limit  query int    false "Pagination Limit"
// @Param        offset query int    false "Pagination Offset"
// @Success      200  {array} query.LoadResponse
// @Failure      401  {object} outerr.Response
// @Security     BearerAuth
// @Router       /companies/{id}/loads [get]
func (h *handler) ListLoads() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := chi.URLParam(r, "id")
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

		req := &query.ListRequest{
			CompanyID: companyID,
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

// ListDrivers godoc
// @Summary      List company drivers
// @Description  List drivers for a specific company
// @Tags         Companies
// @Produce      json
// @Param        id   path      string  true  "Company ID"
// @Success      200  {array} driversquery.CompanyDriverResponse
// @Failure      401  {object} outerr.Response
// @Security     BearerAuth
// @Router       /companies/{id}/drivers [get]
func (h *handler) ListDrivers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := chi.URLParam(r, "id")

		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		var resp []*driversquery.CompanyDriverResponse
		resp, err := h.driversUsecase.Query.ListByCompany(r.Context(), userID, companyID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}

// AddDriver godoc
// @Summary      Add driver to company
// @Description  Add an existing driver to the company
// @Tags         Companies
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Company ID"
// @Param        body body driverscmd.AddToCompanyRequest true "Add Driver Request"
// @Success      201  {object} map[string]string
// @Failure      400  {object} outerr.Response
// @Failure      401  {object} outerr.Response
// @Security     BearerAuth
// @Router       /companies/{id}/drivers [post]
func (h *handler) AddDriver() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := chi.URLParam(r, "id")

		var req driverscmd.AddToCompanyRequest
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
