package common

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads/query"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/location"
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

// List godoc
// @Security     BearerAuth
// @Summary      List loads
// @Description  List loads with optional filters
// @Tags         Loads
// @Produce      json
// @Param        company_id query string false "Company ID filter"
// @Param        carrier_id  query string false "Carrier ID filter"
// @Param        status     query string false "Status filter"
// @Param        limit      query int    false "Pagination Limit"
// @Param        offset     query int    false "Pagination Offset"
// @Success      200  {array} query.LoadResponse
// @Failure      401  {object} outerr.Response
// @Router       /loads [get]
func (h *loadsHandler) List() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

		req := &query.ListRequest{
			CompanyID: r.URL.Query().Get("company_id"),
			CarrierID: r.URL.Query().Get("carrier_id"),
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

// Get godoc
// @Security     BearerAuth
// @Summary      Get load
// @Description  Get load by ID
// @Tags         Loads
// @Produce      json
// @Param        id   path      string  true  "Load ID"
// @Success      200  {object} query.LoadResponse
// @Failure      401  {object} outerr.Response
// @Failure      404  {object} outerr.Response
// @Router       /loads/{id} [get]
func (h *loadsHandler) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loadID := chi.URLParam(r, "id")

		resp, err := h.loadsUsecase.Query.Get(r.Context(), loadID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}
