package shippers

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

// CreateInviteLink godoc
// @Security     BearerAuth
// @Summary      Create load invite link
// @Description  Generate a shareable link a driver can open to log in/register as a carrier and accept this load, without the shipper needing their phone/email upfront. Idempotent: returns the existing active invite if one exists.
// @Tags         Loads
// @Produce      json
// @Param        id   path      string  true  "Load ID"
// @Success      200  {object} command.CreateInviteLinkResponse
// @Failure      400  {object} outerr.Response
// @Failure      401  {object} outerr.Response
// @Failure      403  {object} outerr.Response
// @Failure      404  {object} outerr.Response
// @Router       /loads/{id}/invite-link [post]
func (h *invitesHandler) CreateInviteLink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		loadID := chi.URLParam(r, "id")

		resp, err := h.invitesUsecase.Command.CreateInviteLink(r.Context(), userID, loadID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp)
	}
}
