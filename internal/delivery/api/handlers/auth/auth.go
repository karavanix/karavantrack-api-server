package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/auth"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
)

type authHander struct {
	cfg         *config.Config
	validator   *validation.Validator
	authUsecase *auth.Usecase
}

func New(opts *delivery.HandlerOptions) http.Handler {
	handler := &authHander{
		cfg:         opts.Config,
		validator:   opts.Validator,
		authUsecase: opts.AuthUsecase,
	}

	r := chi.NewRouter()

	r.Post("/login", handler.Login())
	r.Post("/register", handler.Register())
	r.Post("/logout", handler.Logout())
	r.Post("/refresh", handler.Refresh())

	return r
}

func (h *authHander) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func (h *authHander) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func (h *authHander) Logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func (h *authHander) Refresh() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}
