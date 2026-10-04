package carriers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/render"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/location"
	locationcmd "github.com/karavanix/karavantrack-api-server/internal/usecase/location/command"
	"github.com/karavanix/karavantrack-api-server/pkg/app"
)

// maxLocationsBodyBytes guards against garbage, not real batches: the phone
// sends at most 250 records of about half a kilobyte each.
const maxLocationsBodyBytes = 2 << 20

type trackingHandler struct {
	locationUsecase *location.Usecase
}

func NewTrackingHandler(opts *delivery.HandlerOptions) *trackingHandler {
	return &trackingHandler{
		locationUsecase: opts.LocationUsecase,
	}
}

// RegisterLocations godoc
// @Security     BearerAuth
// @Summary      Register GPS points from the tracking library
// @Description  Takes a batch of the phone's tracking library records. Each point carries its load_id; points outside their load's tracking window (from acceptance to confirmation, a cancelled load keeps taking them), of other drivers' loads or malformed are dropped, not rejected. Any parsable body gets 200, so the phone can clear its queue; the response's load_status and stop_tracking describe the load of the batch's latest point.
// @Tags         Tracking
// @Accept       json
// @Produce      json
// @Param        body body locationcmd.RegisterLocationsRequest true "Tracking library records"
// @Success      200  {object} locationcmd.RegisterLocationsResponse
// @Failure      400  {object} outerr.Response
// @Failure      401  {object} outerr.Response
// @Failure      413  {object} outerr.Response
// @Router       /tracking/locations [post]
func (h *trackingHandler) RegisterLocations() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxLocationsBodyBytes)

		var req locationcmd.RegisterLocationsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				outerr.RequestEntityTooLarge(w, r, "request body exceeds 2MB")
				return
			}
			outerr.BadRequest(w, r, "invalid request body")
			return
		}

		req.CarrierID = userID

		resp, err := h.locationUsecase.Command.RegisterLocations(r.Context(), &req)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp)
	}
}
