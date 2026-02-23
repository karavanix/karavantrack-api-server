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
	"github.com/karavanix/karavantrack-api-server/internal/usecase/companies/query"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads"
	query_loads "github.com/karavanix/karavantrack-api-server/internal/usecase/loads/query"
	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
)

type handler struct {
	cfg              *config.Config
	validator        *validation.Validator
	companiesUsecase *companies.Usecase
	loadsUsecase     *loads.Usecase
}

func New(opts *delivery.HandlerOptions) http.Handler {
	h := &handler{
		cfg:              opts.Config,
		validator:        opts.Validator,
		companiesUsecase: opts.CompaniesUsecase,
		loadsUsecase:     opts.LoadsUsecase,
	}

	r := chi.NewRouter()
	r.Use(middleware.AuthContext(opts.JWTProvider))

	r.Post("/", h.Create())
	r.Get("/mine", h.ListMine())
	r.Get("/{id}", h.Get())
	r.Put("/{id}", h.Update())

	// Members sub-routes
	r.Post("/{id}/members", h.AddMember())
	r.Get("/{id}/members", h.ListMembers())
	r.Delete("/{id}/members/{userId}", h.RemoveMember())

	// Company-scoped loads and carriers
	r.Get("/{id}/loads", h.ListLoads())
	r.Get("/{id}/carriers", h.ListCarriers())
	r.Post("/{id}/carriers", h.AddCarrier())
	r.Delete("/{id}/carriers/{carrier_id}", h.RemoveCarrier())

	return r
}

// Create godoc
// @Security     BearerAuth
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
// @Security     BearerAuth
// @Summary      List companies
// @Description  List companies for the current user
// @Tags         Companies
// @Produce      json
// @Success      200  {array} query.CompanyResponse
// @Failure      401  {object} outerr.Response
// @Failure      403  {object} outerr.Response
// @Router       /companies/mine [get]
func (h *handler) ListMine() http.HandlerFunc {
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
// @Security     BearerAuth
// @Summary      Get company
// @Description  Get company by ID
// @Tags         Companies
// @Produce      json
// @Param        id   path      string  true  "Company ID"
// @Success      200  {object} query.CompanyResponse
// @Failure      401  {object} outerr.Response
// @Failure      404  {object} outerr.Response
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
// @Security     BearerAuth
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
// @Security     BearerAuth
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
// @Security     BearerAuth
// @Summary      List members
// @Description  List members of a company
// @Tags         Companies
// @Produce      json
// @Param        id   path      string  true  "Company ID"
// @Success      200  {array} query.MemberResponse
// @Failure      401  {object} outerr.Response
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
// @Security     BearerAuth
// @Summary      Remove member
// @Description  Remove a member from the company (owner only)
// @Tags         Companies
// @Produce      json
// @Param        id   path      string  true  "Company ID"
// @Param        userId path    string  true  "User ID"
// @Success      200  {object} map[string]string
// @Failure      401  {object} outerr.Response
// @Failure      403  {object} outerr.Response
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
// @Security     BearerAuth
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
// @Router       /companies/{id}/loads [get]
func (h *handler) ListLoads() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := chi.URLParam(r, "id")
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

		req := &query_loads.ListRequest{
			CompanyID: companyID,
			Status:    r.URL.Query().Get("status"),
			Limit:     limit,
			Offset:    offset,
		}

		resp, err := h.loadsUsecase.Query.List(r.Context(), req)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}

// ListCarriers godoc
// @Security     BearerAuth
// @Summary      List company carriers
// @Description  List carriers for a specific company
// @Tags         Companies
// @Produce      json
// @Param        id   path      string  true  "Company ID"
// @Success      200  {array} query.ListCarriersResponse
// @Failure      401  {object} outerr.Response
// @Router       /companies/{id}/carriers [get]
func (h *handler) ListCarriers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := chi.URLParam(r, "id")

		_, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		var resp []*query.ListCarriersResponse
		resp, err := h.companiesUsecase.Query.ListCarriers(r.Context(), companyID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}

// AddCarrier godoc
// @Security     BearerAuth
// @Summary      Add carrier to company
// @Description  Add an existing carrier to the company
// @Tags         Companies
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Company ID"
// @Param        body body command.AddCarrierRequest true "Add Carrier Request"
// @Success      201
// @Failure      400  {object} outerr.Response
// @Failure      401  {object} outerr.Response
// @Router       /companies/{id}/carriers [post]
func (h *handler) AddCarrier() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}
		companyID := chi.URLParam(r, "id")

		var req command.AddCarrierRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			outerr.BadRequest(w, r, "invalid request body")
			return
		}
		if err := h.validator.Validate(req); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		if err := h.companiesUsecase.Command.AddCarrier(r.Context(), userID, companyID, &req); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusCreated)
	}
}

// RemoveCarrier godoc
// @Security     BearerAuth
// @Summary      Remove carrier from company
// @Description  Remove a carrier from the company
// @Tags         Companies
// @Produce      json
// @Param        id   path      string  true  "Company ID"
// @Param        carrier_id    path      string  true  "Carrier ID"
// @Success      200
// @Failure      401  {object} outerr.Response
// @Router       /companies/{id}/carriers/{carrier_id} [delete]
func (h *handler) RemoveCarrier() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := chi.URLParam(r, "id")
		carrierID := chi.URLParam(r, "carrier_id")

		if err := h.companiesUsecase.Command.RemoveCarrier(r.Context(), companyID, carrierID); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
	}
}
