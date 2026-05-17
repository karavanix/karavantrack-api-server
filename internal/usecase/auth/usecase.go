package auth

import (
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/service/email"
	"github.com/karavanix/karavantrack-api-server/internal/service/otp"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/auth/command"
	"github.com/karavanix/karavantrack-api-server/pkg/apple"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/karavanix/karavantrack-api-server/pkg/security"
)

type Command struct {
	*command.RegisterUsecase
	*command.LoginUsecase
	*command.RefreshTokenUsecase
	*command.AppleSignInUsecase
	*command.VerifyEmailUsecase
}

type Query struct {
}

type Usecase struct {
	Command Command
	Query   Query
}

type Config struct {
	OTPService   *otp.Service
	EmailService *email.Service
}

func NewUsecase(
	contextDuration time.Duration,
	jwtProvider *security.JWTProvider,
	txManager postgres.TxManager,
	usersRepo domain.UserRepository,
	oauthAccountsRepo domain.OAuthAccountRepository,
	appleClient *apple.Client,
	cfg Config,
) *Usecase {
	return &Usecase{
		Command: Command{
			RegisterUsecase:     command.NewRegisterUsecase(contextDuration, usersRepo, cfg.OTPService, cfg.EmailService),
			LoginUsecase:        command.NewLoginUsecase(contextDuration, jwtProvider, usersRepo),
			RefreshTokenUsecase: command.NewRefreshTokenUsecase(contextDuration, jwtProvider, usersRepo),
			AppleSignInUsecase:  command.NewAppleSignInUsecase(contextDuration, jwtProvider, appleClient, txManager, usersRepo, oauthAccountsRepo),
			VerifyEmailUsecase:  command.NewVerifyEmailUsecase(contextDuration, jwtProvider, usersRepo, cfg.OTPService),
		},
		Query: Query{},
	}
}
