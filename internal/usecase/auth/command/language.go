package command

import (
	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
)

// parseLanguage reads the language a sign-up request carries; none means
// the default one.
func parseLanguage(raw string) (shared.Language, error) {
	if raw == "" {
		return shared.DefaultLanguage, nil
	}
	language, err := shared.NewLanguage(raw)
	if err != nil {
		return "", inerr.NewErrValidation("language", err.Error())
	}
	return language, nil
}
