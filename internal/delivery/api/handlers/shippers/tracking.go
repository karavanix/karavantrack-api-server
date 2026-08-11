package shippers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/tracking"
	"github.com/karavanix/karavantrack-api-server/pkg/app"
)

type trackingHandler struct {
	trackingUsecase *tracking.Usecase
}

func NewTrackingHandler(opts *delivery.HandlerOptions) *trackingHandler {
	return &trackingHandler{trackingUsecase: opts.TrackingUsecase}
}

// CreateTrackingLink godoc
// @Security     BearerAuth
// @Summary      Create public cargo tracking link
// @Description  Generate a public, no-login, read-only link a broker can hand to their client to follow this load. No expiry. Idempotent: returns the existing active link if one exists.
// @Tags         Loads
// @Produce      json
// @Param        id   path      string  true  "Load ID"
// @Success      200  {object} command.CreateTrackingLinkResponse
// @Failure      400  {object} outerr.Response
// @Failure      401  {object} outerr.Response
// @Failure      403  {object} outerr.Response
// @Failure      404  {object} outerr.Response
// @Router       /loads/{id}/tracking-link [post]
func (h *trackingHandler) CreateTrackingLink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		loadID := chi.URLParam(r, "id")

		resp, err := h.trackingUsecase.Command.CreateTrackingLink(r.Context(), userID, loadID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp)
	}
}
