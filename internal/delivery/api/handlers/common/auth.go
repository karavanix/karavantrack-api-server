package common

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/render"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/auth"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/auth/command"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
	"github.com/karavanix/karavantrack-api-server/pkg/security"
)

type authHander struct {
	cfg         *config.Config
	validator   *validation.Validator
	authUsecase *auth.Usecase
	jwtProvider *security.JWTProvider
}

func NewAuthHandler(opts *delivery.HandlerOptions) *authHander {
	return &authHander{
		cfg:         opts.Config,
		validator:   opts.Validator,
		authUsecase: opts.AuthUsecase,
		jwtProvider: opts.JWTProvider,
	}
}

// Login godoc
// @Summary      Login
// @Description  Authenticate user with email/phone and password
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body command.LoginRequest true "Login credentials"
// @Success      200  {object} command.LoginResponse
// @Failure      400  {object} outerr.Response
// @Failure      403  {object} outerr.Response
// @Router       /auth/login [post]
func (h *authHander) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req command.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			outerr.BadRequest(w, r, "invalid request body")
			return
		}

		if err := h.validator.Validate(req); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		resp, err := h.authUsecase.Command.Login(r.Context(), &req)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp)
	}
}

// Register godoc
// @Summary      Register
// @Description  Create a new account and send a verification code to the provided email
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body command.RegisterRequest true "Registration data"
// @Success      200
// @Failure      400  {object} outerr.Response
// @Failure      409  {object} outerr.Response
// @Router       /auth/register [post]
func (h *authHander) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req command.RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			outerr.BadRequest(w, r, "invalid request body")
			return
		}

		if err := h.validator.Validate(req); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		err := h.authUsecase.Command.Register(r.Context(), &req)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
	}
}

// VerifyEmail godoc
// @Summary      Verify email
// @Description  Verify the OTP code sent to email and receive access tokens
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body command.VerifyEmailRequest true "Email and OTP code"
// @Success      200  {object} command.VerifyEmailResponse
// @Failure      400  {object} outerr.Response
// @Router       /auth/verify-email [post]
func (h *authHander) VerifyEmail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req command.VerifyEmailRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			outerr.BadRequest(w, r, "invalid request body")
			return
		}

		if err := h.validator.Validate(req); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		resp, err := h.authUsecase.Command.VerifyEmail(r.Context(), &req)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp)
	}
}

// Logout godoc
// @Summary      Logout
// @Description  Logout user (invalidate session)
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Success      200
// @Router       /auth/logout [post]
func (h *authHander) Logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		render.Status(r, http.StatusOK)
	}
}

// AppleSignIn godoc
// @Summary      Apple Sign In
// @Description  Authenticate or register a user via Apple ID token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body command.AppleSignInRequest true "Apple ID token and optional profile fields"
// @Success      200  {object} command.AppleSignInResponse
// @Failure      400  {object} outerr.Response
// @Failure      403  {object} outerr.Response
// @Router       /auth/apple [post]
func (h *authHander) AppleSignIn() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req command.AppleSignInRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			outerr.BadRequest(w, r, "invalid request body")
			return
		}

		if err := h.validator.Validate(req); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		resp, err := h.authUsecase.Command.AppleSignIn(r.Context(), &req)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp)
	}
}

// Refresh godoc
// @Summary      Refresh tokens
// @Description  Exchange a valid refresh token for a new token pair
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body command.RefreshTokenRequest true "Refresh token"
// @Success      200  {object} command.LoginResponse
// @Failure      401  {object} outerr.Response
// @Router       /auth/refresh [post]
func (h *authHander) Refresh() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req command.RefreshTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			outerr.BadRequest(w, r, "invalid request body")
			return
		}

		if err := h.validator.Validate(req); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		var resp *command.LoginResponse
		resp, err := h.authUsecase.Command.RefreshToken(r.Context(), &req)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}
