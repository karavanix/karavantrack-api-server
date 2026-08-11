package common

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/tracking"
)

type publicTrackingHandler struct {
	trackingUsecase *tracking.Usecase
}

func NewPublicTrackingHandler(opts *delivery.HandlerOptions) *publicTrackingHandler {
	return &publicTrackingHandler{trackingUsecase: opts.TrackingUsecase}
}

// GetTracking godoc
// @Summary      Get public cargo tracking snapshot
// @Description  PUBLIC, unauthenticated snapshot of a load's status and latest known position, by tracking-link token
// @Tags         Tracking
// @Produce      json
// @Param        token path string true "Tracking link token"
// @Success      200  {object} query.GetPublicTrackingResponse
// @Failure      404  {object} outerr.Response
// @Router       /public/tracking/{token} [get]
func (h *publicTrackingHandler) GetTracking() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := chi.URLParam(r, "token")

		resp, err := h.trackingUsecase.Query.GetPublicTracking(r.Context(), token)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}

// GetTrack godoc
// @Summary      Get public cargo tracking history
// @Description  PUBLIC, unauthenticated location history for a load, by tracking-link token (same shape as the authenticated GET /loads/{id}/track)
// @Tags         Tracking
// @Produce      json
// @Param        token  path  string true  "Tracking link token"
// @Param        limit  query int    false "Max number of points (default 100, max 1000)"
// @Param        offset query int    false "Pagination offset"
// @Success      200  {object} query.GetTrackResponse
// @Failure      404  {object} outerr.Response
// @Router       /public/tracking/{token}/track [get]
func (h *publicTrackingHandler) GetTrack() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := chi.URLParam(r, "token")
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

		resp, err := h.trackingUsecase.Query.GetPublicTrack(r.Context(), token, limit, offset)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}
