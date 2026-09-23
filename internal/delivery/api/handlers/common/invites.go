package common

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/invites"
	"github.com/karavanix/karavantrack-api-server/pkg/security"
)

type publicInvitesHandler struct {
	invitesUsecase *invites.Usecase
	jwtProvider    *security.JWTProvider
}

func NewPublicInvitesHandler(opts *delivery.HandlerOptions) *publicInvitesHandler {
	return &publicInvitesHandler{invitesUsecase: opts.InvitesUsecase, jwtProvider: opts.JWTProvider}
}

// optionalViewerID best-effort decodes a bearer token if the request happens
// to carry one — this endpoint is public and unauthenticated, so a missing
// or invalid token is not an error, it just means an anonymous viewer.
func (h *publicInvitesHandler) optionalViewerID(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if len(auth) <= 7 || !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return ""
	}
	claims, err := h.jwtProvider.ValidateAccessToken(auth[7:])
	if err != nil {
		return ""
	}
	return claims.Subject
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
		viewerID := h.optionalViewerID(r)

		resp, err := h.invitesUsecase.Query.GetInvite(r.Context(), token, viewerID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}
