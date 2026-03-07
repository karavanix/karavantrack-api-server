package shippers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-playground/form/v4"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
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

type companiesHandler struct {
	cfg              *config.Config
	validator        *validation.Validator
	companiesUsecase *companies.Usecase
	loadsUsecase     *loads.Usecase
}

func NewCompaniesHandler(opts *delivery.HandlerOptions) *companiesHandler {
	return &companiesHandler{
		cfg:              opts.Config,
		validator:        opts.Validator,
		companiesUsecase: opts.CompaniesUsecase,
		loadsUsecase:     opts.LoadsUsecase,
	}
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
func (h *companiesHandler) Create() http.HandlerFunc {
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

// ListMine godoc
// @Security     BearerAuth
// @Summary      List companies
// @Description  List companies for the current user
// @Tags         Companies
// @Produce      json
// @Success      200  {array} query.CompanyResponse
// @Failure      401  {object} outerr.Response
// @Failure      403  {object} outerr.Response
// @Router       /companies [get]
func (h *companiesHandler) ListShipperCompanies() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		resp, err := h.companiesUsecase.Query.ListShipperCompanies(r.Context(), userID)
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
// @Param        id path string true "Company ID"
// @Success      200 {object} query.CompanyResponse
// @Failure      401 {object} outerr.Response
// @Failure      403 {object} outerr.Response
// @Failure      404 {object} outerr.Response
// @Failure      500 {object} outerr.Response
// @Router       /companies/{id} [get]
func (h *companiesHandler) GetShipperCompany() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}
		companyID := chi.URLParam(r, "id")

		resp, err := h.companiesUsecase.Query.GetShipperCompany(r.Context(), userID, companyID)
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
func (h *companiesHandler) Update() http.HandlerFunc {
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
func (h *companiesHandler) AddMember() http.HandlerFunc {
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
func (h *companiesHandler) ListMembers() http.HandlerFunc {
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
// @Param        user_id path    string  true  "User ID"
// @Success      200  {object} map[string]string
// @Failure      401  {object} outerr.Response
// @Failure      403  {object} outerr.Response
// @Router       /companies/{id}/members/{user_id} [delete]
func (h *companiesHandler) RemoveMember() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		companyID := chi.URLParam(r, "id")
		targetUserID := chi.URLParam(r, "user_id")

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
// @Param        status query []string false "Status filter" collectionFormat(multi) Enums(created, assigned, accepted, in_transit, completed, confirmed, cancelled)
// @Param        limit  query int    false "Pagination Limit"
// @Param        offset query int    false "Pagination Offset"
// @Success      200  {array} query.LoadResponse
// @Failure      401  {object} outerr.Response
// @Router       /companies/{id}/loads [get]
func (h *companiesHandler) ListLoads() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := chi.URLParam(r, "id")

		var urlForm query_loads.ListRequest
		if err := form.NewDecoder().Decode(&urlForm, r.URL.Query()); err != nil {
			outerr.BadRequest(w, r, "invalid request form: "+err.Error())
			return
		}
		urlForm.CompanyID = companyID

		resp, err := h.loadsUsecase.Query.List(r.Context(), &urlForm)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}

// GetStats godoc
// @Security     BearerAuth
// @Summary      Get load
// @Description  Get load by ID
// @Tags         Loads
// @Produce      json
// @Success      200  {object} query.GetStatsResponse
// @Failure      401  {object} outerr.Response
// @Failure      404  {object} outerr.Response
// @Router       /companies/{id}/loads/stats [get]
func (h *loadsHandler) GetLoadStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := chi.URLParam(r, "id")
		if companyID == "" {
			outerr.BadRequest(w, r, "missing company id")
			return
		}
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		resp, err := h.loadsUsecase.Query.GetStats(r.Context(), userID, companyID)
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
func (h *companiesHandler) ListCarriers() http.HandlerFunc {
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
func (h *companiesHandler) AddCarrier() http.HandlerFunc {
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
func (h *companiesHandler) RemoveCarrier() http.HandlerFunc {
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
