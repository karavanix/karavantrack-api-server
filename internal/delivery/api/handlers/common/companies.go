package common

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/companies"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads"
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
func (h *companiesHandler) Get() http.HandlerFunc {
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
