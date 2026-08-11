package common

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/invites"
)

type publicInvitesHandler struct {
	invitesUsecase *invites.Usecase
}

func NewPublicInvitesHandler(opts *delivery.HandlerOptions) *publicInvitesHandler {
	return &publicInvitesHandler{invitesUsecase: opts.InvitesUsecase}
}

// GetInvite godoc
// @Summary      Get invite preview
// @Description  PUBLIC, unauthenticated preview of a load invite by token. Returns 200 even for expired/accepted/revoked invites so the app can render an explanatory state.
// @Tags         Invites
// @Produce      json
// @Param        token path string true "Invite token"
// @Success      200  {object} query.GetInviteResponse
// @Failure      404  {object} outerr.Response
// @Router       /invites/{token} [get]
func (h *publicInvitesHandler) GetInvite() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := chi.URLParam(r, "token")

		resp, err := h.invitesUsecase.Query.GetInvite(r.Context(), token)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}
