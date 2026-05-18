package command

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"github.com/karavanix/karavantrack-api-server/pkg/security"
	"github.com/karavanix/karavantrack-api-server/pkg/telegram"
	"go.opentelemetry.io/otel"
)

type TelegramSignInUsecase struct {
	contextDuration   time.Duration
	jwtProvider       *security.JWTProvider
	telegramClient    *telegram.Client
	txManager         postgres.TxManager
	usersRepo         domain.UserRepository
	oauthAccountsRepo domain.OAuthAccountRepository
}

func NewTelegramSignInUsecase(
	contextDuration time.Duration,
	jwtProvider *security.JWTProvider,
	telegramClient *telegram.Client,
	txManager postgres.TxManager,
	usersRepo domain.UserRepository,
	oauthAccountsRepo domain.OAuthAccountRepository,
) *TelegramSignInUsecase {
	return &TelegramSignInUsecase{
		contextDuration:   contextDuration,
		jwtProvider:       jwtProvider,
		telegramClient:    telegramClient,
		txManager:         txManager,
		usersRepo:         usersRepo,
		oauthAccountsRepo: oauthAccountsRepo,
	}
}

type TelegramSignInRequest struct {
	ID        string `json:"id"         validate:"required"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	PhotoURL  string `json:"photo_url"`
	AuthDate  string `json:"auth_date"  validate:"required"`
	Hash      string `json:"hash"       validate:"required"`
	Role      string `json:"role"`
}

type TelegramSignInResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Role         string `json:"role"`
	IsNewUser    bool   `json:"is_new_user"`
}

func (u *TelegramSignInUsecase) TelegramSignIn(ctx context.Context, req *TelegramSignInRequest) (_ *TelegramSignInResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("auth"), "TelegramSignIn")
	defer func() { end(err) }()

	data := map[string]string{
		"id":        req.ID,
		"auth_date": req.AuthDate,
		"hash":      req.Hash,
	}
	if req.FirstName != "" {
		data["first_name"] = req.FirstName
	}
	if req.LastName != "" {
		data["last_name"] = req.LastName
	}
	if req.Username != "" {
		data["username"] = req.Username
	}
	if req.PhotoURL != "" {
		data["photo_url"] = req.PhotoURL
	}

	userInfo, err := u.telegramClient.Verify(data)
	if err != nil {
		logger.ErrorContext(ctx, "telegram auth verification failed", err)
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

		var newUser *domain.User
		txErr := u.txManager.WithTx(ctx, func(ctx context.Context) error {
			newUser, err = domain.NewUserFromTelegram(userInfo.FirstName, userInfo.LastName, role)
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
