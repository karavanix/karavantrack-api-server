package common

import (
	"net/http"

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

// Get godoc
// @Security     BearerAuth
// @Summary      Get load
// @Description  Get load by ID
// @Tags         Loads
// @Produce      json
// @Param        id   path      string  true  "Load ID"
// @Success      200  {object} query.LoadDetailResponse
// @Failure      401  {object} outerr.Response
// @Failure      404  {object} outerr.Response
// @Router       /loads/{id} [get]
func (h *loadsHandler) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loadID := chi.URLParam(r, "id")

		var resp *query.LoadDetailResponse
		resp, err := h.loadsUsecase.Query.Get(r.Context(), loadID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}
