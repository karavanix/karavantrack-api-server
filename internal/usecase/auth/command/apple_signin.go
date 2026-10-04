package command

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/ports"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"github.com/karavanix/karavantrack-api-server/pkg/security"
	"go.opentelemetry.io/otel"
)

type AppleSignInUsecase struct {
	contextDuration   time.Duration
	jwtProvider       *security.JWTProvider
	appleClient       ports.AppleProvider
	txManager         postgres.TxManager
	usersRepo         domain.UserRepository
	oauthAccountsRepo domain.OAuthAccountRepository
}

func NewAppleSignInUsecase(
	contextDuration time.Duration,
	jwtProvider *security.JWTProvider,
	appleClient ports.AppleProvider,
	txManager postgres.TxManager,
	usersRepo domain.UserRepository,
	oauthAccountsRepo domain.OAuthAccountRepository,
) *AppleSignInUsecase {
	return &AppleSignInUsecase{
		contextDuration:   contextDuration,
		jwtProvider:       jwtProvider,
		appleClient:       appleClient,
		txManager:         txManager,
		usersRepo:         usersRepo,
		oauthAccountsRepo: oauthAccountsRepo,
	}
}

type AppleSignInRequest struct {
	IDToken   string `json:"id_token"   validate:"required"`
	Role      string `json:"role"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	// AuthorizationCode from the same sign-in: traded for Apple's refresh
	// token, which is revoked when the account is deleted.
	AuthorizationCode string `json:"authorization_code"`
}

type AppleSignInResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	// ExpiresIn is the access token's lifetime in seconds.
	ExpiresIn int    `json:"expires_in"`
	Role      string `json:"role"`
	IsNewUser bool   `json:"is_new_user"`
}

func (u *AppleSignInUsecase) AppleSignIn(ctx context.Context, req *AppleSignInRequest) (_ *AppleSignInResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("auth"), "AppleSignIn")
	defer func() { end(err) }()

	userInfo, err := u.appleClient.Verify(ctx, req.IDToken)
	if err != nil {
		logger.ErrorContext(ctx, "apple id_token verification failed", err)
		return nil, inerr.ErrorPermissionDenied
	}

	// Without it the sign-in still goes through: only revoking on account
	// deletion is lost, until the next sign-in brings a code again.
	appleRefreshToken := ""
	if req.AuthorizationCode != "" {
		appleRefreshToken, err = u.appleClient.ExchangeCode(ctx, req.AuthorizationCode)
		if err != nil {
			logger.WarnContext(ctx, "apple authorization code not exchanged", "error", err)
			appleRefreshToken = ""
			err = nil
		}
	}
	newAccount := func(userID uuid.UUID) *domain.OAuthAccount {
		account := domain.NewOAuthAccount(userID, domain.OAuthProviderApple, userInfo.Sub)
		account.ProviderRefreshToken = appleRefreshToken
		return account
	}

	oauthAccount, err := u.oauthAccountsRepo.FindByProviderAndProviderAccountID(
		ctx, domain.OAuthProviderApple, userInfo.Sub,
	)
	if err != nil && !errors.Is(err, inerr.ErrNotFound{}) {
		logger.ErrorContext(ctx, "failed to find oauth account", err)
		return nil, err
	}

	isNewUser := false
	var user *domain.User

	if oauthAccount != nil {
		user, err = u.usersRepo.FindByID(ctx, oauthAccount.UserID)
		if err != nil {
			logger.ErrorContext(ctx, "failed to find linked user", err)
			return nil, err
		}
		if appleRefreshToken != "" {
			if err = u.oauthAccountsRepo.Save(ctx, newAccount(user.ID)); err != nil {
				logger.ErrorContext(ctx, "failed to save apple refresh token", err)
				return nil, err
			}
		}
	} else {
		if userInfo.Email != "" {
			email, emailErr := shared.NewEmail(userInfo.Email)
			if emailErr == nil {
				user, err = u.usersRepo.FindByEmail(ctx, email)
				if err != nil && !errors.Is(err, inerr.ErrNotFound{}) {
					logger.ErrorContext(ctx, "failed to find user by email", err)
					return nil, err
				}
				if errors.Is(err, inerr.ErrNotFound{}) {
					user = nil
					err = nil
				}
			}
		}

		if user == nil {
			role := shared.Role(req.Role)
			if !role.IsValid() {
				return nil, inerr.NewErrValidation("role", "role is required for new Apple users: shipper or carrier")
			}

			var email shared.Email
			if userInfo.Email != "" {
				email, _ = shared.NewEmail(userInfo.Email)
			}

			var newUser *domain.User
			txErr := u.txManager.WithTx(ctx, func(ctx context.Context) error {
				newUser, err = domain.NewUserFromApple(req.FirstName, req.LastName, email, role)
				if err != nil {
					return err
				}
				if err = u.usersRepo.Save(ctx, newUser); err != nil {
					return err
				}
				return u.oauthAccountsRepo.Save(ctx, newAccount(newUser.ID))
			})
			if txErr != nil {
				logger.ErrorContext(ctx, "failed to create apple user", txErr)
				return nil, txErr
			}
			user = newUser
			isNewUser = true
		} else {
			if err = u.oauthAccountsRepo.Save(ctx, newAccount(user.ID)); err != nil {
				logger.ErrorContext(ctx, "failed to link oauth account", err)
				return nil, err
			}
		}
	}

	creds, err := u.jwtProvider.GenerateTokens(user.ID.String(), user.Role.String())
	if err != nil {
		logger.ErrorContext(ctx, "failed to generate tokens", err)
		return nil, err
	}

	return &AppleSignInResponse{
		AccessToken:  creds.AccessToken,
		RefreshToken: creds.RefreshToken,
		ExpiresIn:    int(creds.AccessTTL.Seconds()),
		Role:         user.Role.String(),
		IsNewUser:    isNewUser,
	}, nil
}
