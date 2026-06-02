package command

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"github.com/karavanix/karavantrack-api-server/pkg/security"
	"go.opentelemetry.io/otel"
)

type TelegramOAuthUsecase struct {
	contextDuration   time.Duration
	jwtProvider       *security.JWTProvider
	telegramClient    domain.TelegramProvider
	txManager         postgres.TxManager
	usersRepo         domain.UserRepository
	oauthAccountsRepo domain.OAuthAccountRepository
	pkceRepo          domain.PKCERepository
}

func NewTelegramOAuthUsecase(
	contextDuration time.Duration,
	jwtProvider *security.JWTProvider,
	telegramClient domain.TelegramProvider,
	txManager postgres.TxManager,
	usersRepo domain.UserRepository,
	oauthAccountsRepo domain.OAuthAccountRepository,
	pkceRepo domain.PKCERepository,
) *TelegramOAuthUsecase {
	return &TelegramOAuthUsecase{
		contextDuration:   contextDuration,
		jwtProvider:       jwtProvider,
		telegramClient:    telegramClient,
		txManager:         txManager,
		usersRepo:         usersRepo,
		oauthAccountsRepo: oauthAccountsRepo,
		pkceRepo:          pkceRepo,
	}
}

type TelegramOAuthRequest struct {
	Code        string `json:"code"         validate:"required"`
	State       string `json:"state"        validate:"required"`
	RedirectURI string `json:"redirect_uri" validate:"required"`
	Role        string `json:"role"`
}

func (u *TelegramOAuthUsecase) TelegramOAuth(ctx context.Context, req *TelegramOAuthRequest) (_ *TelegramSignInResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("auth"), "TelegramOAuth")
	defer func() { end(err) }()

	var input struct {
		state uuid.UUID
	}
	{
		input.state, err = uuid.Parse(req.State)
		if err != nil {
			return nil, inerr.NewErrValidation("state", "invalid state format")
		}
	}

	pkce, err := u.pkceRepo.FindByState(ctx, input.state)
	if err != nil {
		if errors.As(err, &inerr.ErrNotFound{}) {
			return nil, inerr.ErrorPermissionDenied
		}
		logger.ErrorContext(ctx, "failed to look up pkce state", err)
		return nil, err
	}

	if pkce.IsExpired(time.Now()) {
		_ = u.pkceRepo.Delete(ctx, input.state)
		return nil, inerr.ErrorPermissionDenied
	}

	if err = u.pkceRepo.Delete(ctx, input.state); err != nil {
		logger.ErrorContext(ctx, "failed to delete pkce state", err)
		return nil, err
	}

	idToken, err := u.telegramClient.ExchangeCode(ctx, req.Code, req.RedirectURI, pkce.Verifier.String())
	if err != nil {
		logger.ErrorContext(ctx, "telegram code exchange failed", err)
		return nil, inerr.ErrorPermissionDenied
	}

	userInfo, err := u.telegramClient.Verify(ctx, idToken)
	if err != nil {
		logger.ErrorContext(ctx, "telegram OIDC verification failed", err)
		return nil, inerr.ErrorPermissionDenied
	}

	providerAccountID := strconv.FormatInt(userInfo.ID, 10)

	oauthAccount, err := u.oauthAccountsRepo.FindByProviderAndProviderAccountID(
		ctx, domain.OAuthProviderTelegram, providerAccountID,
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
	} else {
		role := shared.Role(req.Role)
		if !role.IsValid() {
			return nil, inerr.NewErrValidation("role", "role is required for new Telegram users: shipper or carrier")
		}

		phone, err := shared.NewPhone(userInfo.PhoneNumber)
		if err != nil {
			return nil, inerr.NewErrValidation("phone", "phone number is required: ensure the Telegram client requests the 'phone' scope")
		}

		var newUser *domain.User
		txErr := u.txManager.WithTx(ctx, func(ctx context.Context) error {
			newUser, err = domain.NewUserFromTelegram(userInfo.FirstName, userInfo.LastName, phone, role)
			if err != nil {
				return err
			}
			if err = u.usersRepo.Save(ctx, newUser); err != nil {
				return err
			}
			account := domain.NewOAuthAccount(newUser.ID, domain.OAuthProviderTelegram, providerAccountID)
			return u.oauthAccountsRepo.Save(ctx, account)
		})
		if txErr != nil {
			logger.ErrorContext(ctx, "failed to create telegram user", txErr)
			return nil, txErr
		}
		user = newUser
		isNewUser = true
	}

	creds, err := u.jwtProvider.GenerateTokens(user.ID.String(), user.Role.String())
	if err != nil {
		logger.ErrorContext(ctx, "failed to generate tokens", err)
		return nil, err
	}

	return &TelegramSignInResponse{
		AccessToken:  creds.AccessToken,
		RefreshToken: creds.RefreshToken,
		Role:         user.Role.String(),
		IsNewUser:    isNewUser,
	}, nil
}
