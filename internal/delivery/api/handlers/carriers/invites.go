package carriers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/invites"
	"github.com/karavanix/karavantrack-api-server/pkg/app"
)

type invitesHandler struct {
	invitesUsecase *invites.Usecase
}

func NewInvitesHandler(opts *delivery.HandlerOptions) *invitesHandler {
	return &invitesHandler{invitesUsecase: opts.InvitesUsecase}
}

// Accept godoc
// @Security     BearerAuth
// @Summary      Accept a load invite
// @Description  Accept the load an invite link points to. The authenticated carrier is assigned to the load (same transition as the shipper-initiated assign flow).
// @Tags         Invites
// @Produce      json
// @Param        token path string true "Invite token"
// @Success      200  {object} command.AcceptInviteResponse
// @Failure      401  {object} outerr.Response
// @Failure      404  {object} outerr.Response
// @Failure      409  {object} outerr.Response
// @Router       /invites/{token}/accept [post]
func (h *invitesHandler) Accept() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		token := chi.URLParam(r, "token")

		resp, err := h.invitesUsecase.Command.AcceptInvite(r.Context(), token, userID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp)
	}
}
