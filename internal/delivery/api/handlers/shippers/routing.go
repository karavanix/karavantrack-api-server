package shippers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/render"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/routing"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/routing/query"
)

type routingHandler struct {
	routingUsecase *routing.Usecase
}

func NewRoutingHandler(opts *delivery.HandlerOptions) *routingHandler {
	return &routingHandler{routingUsecase: opts.RoutingUsecase}
}

// PreviewRoute godoc
// @Security     BearerAuth
// @Summary      Preview a driving route
// @Description  Driving route between two points, e.g. pickup and drop-off while creating a load: geometry as polyline6, distance and estimated duration. Nothing is stored.
// @Tags         Routes
// @Produce      json
// @Param        from query string true "Start point as lat,lng" example(41.311081,69.240562)
// @Param        to   query string true "End point as lat,lng"   example(41.299496,69.240073)
// @Success      200  {object} query.PreviewRouteResponse
// @Failure      400  {object} outerr.Response
// @Failure      401  {object} outerr.Response
// @Failure      422  {object} outerr.Response "No road route between the points"
// @Failure      503  {object} outerr.Response "Routing engine unavailable"
// @Router       /routes/preview [get]
func (h *routingHandler) PreviewRoute() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fromLat, fromLng, err := parseLatLng(r.URL.Query().Get("from"))
		if err != nil {
			outerr.BadRequest(w, r, "from: "+err.Error())
			return
		}
		toLat, toLng, err := parseLatLng(r.URL.Query().Get("to"))
		if err != nil {
			outerr.BadRequest(w, r, "to: "+err.Error())
			return
		}

		resp, err := h.routingUsecase.Query.PreviewRoute(r.Context(), &query.PreviewRouteRequest{
			FromLat: fromLat, FromLng: fromLng,
			ToLat: toLat, ToLng: toLng,
		})
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}

func parseLatLng(s string) (float64, float64, error) {
	lat, lng, ok := strings.Cut(s, ",")
	if !ok {
		return 0, 0, fmt.Errorf("expected lat,lng")
	}
	latV, err := strconv.ParseFloat(strings.TrimSpace(lat), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid latitude")
	}
	lngV, err := strconv.ParseFloat(strings.TrimSpace(lng), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid longitude")
	}
	return latV, lngV, nil
}
