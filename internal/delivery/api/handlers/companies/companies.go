package companies

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/middleware"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/companies"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/companies/command"
	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
)

type handler struct {
	cfg              *config.Config
	validator        *validation.Validator
	companiesUsecase *companies.Usecase
}

func New(opts *delivery.HandlerOptions) http.Handler {
	h := &handler{
		cfg:              opts.Config,
		validator:        opts.Validator,
		companiesUsecase: opts.CompaniesUsecase,
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

		resp, err := h.companiesUsecase.Command.Create.Create(r.Context(), userID, &req)
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

		resp, err := h.companiesUsecase.Query.ListByUser.List(r.Context(), userID)
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

		resp, err := h.companiesUsecase.Query.Get.Get(r.Context(), companyID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}

// Update godoc
// @Summary      Update company
// @Description  Update company details by ID
// @Tags         Companies
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Company ID"
// @Param        body body command.UpdateRequest true "Update Company Request"
// @Success      200  {object} map[string]string
// @Failure      400  {object} outerr.Response
// @Failure      401  {object} outerr.Response
// @Failure      404  {object} outerr.Response
// @Security     BearerAuth
// @Router       /companies/{id} [put]
func (h *handler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		if err := h.companiesUsecase.Command.Update.Update(r.Context(), companyID, &req); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, map[string]string{"status": "updated"})
	}
}

// AddMember godoc
// @Summary      Add member
// @Description  Add a new member to the company
// @Tags         Companies
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Company ID"
// @Param        body body command.AddMemberRequest true "Add Member Request"
// @Success      201  {object} map[string]string
// @Failure      400  {object} outerr.Response
// @Failure      401  {object} outerr.Response
// @Security     BearerAuth
// @Router       /companies/{id}/members [post]
func (h *handler) AddMember() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		if err := h.companiesUsecase.Command.AddMember.AddMember(r.Context(), companyID, &req); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, map[string]string{"status": "added"})
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

		resp, err := h.companiesUsecase.Query.ListMembers.List(r.Context(), companyID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}

// RemoveMember godoc
// @Summary      Remove member
// @Description  Remove a member from the company
// @Tags         Companies
// @Produce      json
// @Param        id   path      string  true  "Company ID"
// @Param        userId path    string  true  "User ID"
// @Success      200  {object} map[string]string
// @Failure      401  {object} outerr.Response
// @Security     BearerAuth
// @Router       /companies/{id}/members/{userId} [delete]
func (h *handler) RemoveMember() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := chi.URLParam(r, "id")
		userID := chi.URLParam(r, "userId")

		if err := h.companiesUsecase.Command.RemoveMember.RemoveMember(r.Context(), companyID, userID); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, map[string]string{"status": "removed"})
	}
}
