package carriers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/companies"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/companies/query"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads"
	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
)

type companyHandler struct {
	cfg              *config.Config
	validator        *validation.Validator
	loadsUsecase     *loads.Usecase
	companiesUsecase *companies.Usecase
}

func NewCompanyHandler(opts *delivery.HandlerOptions) *companyHandler {
	return &companyHandler{
		cfg:              opts.Config,
		validator:        opts.Validator,
		loadsUsecase:     opts.LoadsUsecase,
		companiesUsecase: opts.CompaniesUsecase,
	}
}

// GetCarrierCompany godoc
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
// @Router       /carriers/companies/{id} [get]
func (h *companyHandler) GetCarrierCompany() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}
		companyID := chi.URLParam(r, "id")

		var resp *query.CompanyResponse
		resp, err := h.companiesUsecase.Query.GetCarrierCompany(r.Context(), userID, companyID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}
