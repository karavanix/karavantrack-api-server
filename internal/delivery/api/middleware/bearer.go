package middleware

import (
	"net/http"
	"strings"

	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/security"
)

func extractBearerToken(r *http.Request) string {
	if tok := r.URL.Query().Get("token"); tok != "" {
		return tok
	}
	auth := r.Header.Get("Authorization")
	if len(auth) > 7 && strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return auth[7:]
	}
	return ""
}

func AuthContext(jwtProvider *security.JWTProvider) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractBearerToken(r)
			claims, err := jwtProvider.ValidateAccessToken(tokenStr)
			if err != nil {
				outerr.HandleHTTP(w, r, err)
				return
			}
			if claims.Subject == "" {
				outerr.Forbidden(w, r, "jwt: invalid token")
				return
			}

			ctx := app.WithUserID(r.Context(), claims.Subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
