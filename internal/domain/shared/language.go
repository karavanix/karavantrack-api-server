package shared

import (
	"errors"
	"strings"
)

// Language is the user's language for emails and push notifications, the
// same one the web cabinet and the mobile app are switched to.
type Language string

const (
	LanguageRussian Language = "ru"
	LanguageUzbek   Language = "uz"
	LanguageEnglish Language = "en"

	// DefaultLanguage is the language of a user who never told us theirs.
	DefaultLanguage = LanguageRussian
)

// NewLanguage accepts ru, uz or en, also as a locale tag such as "uz-Latn-UZ"
// or "en_US".
func NewLanguage(language string) (Language, error) {
	tag := strings.ToLower(strings.TrimSpace(language))
	if i := strings.IndexAny(tag, "-_"); i >= 0 {
		tag = tag[:i]
	}

	l := Language(tag)
	if !l.IsValid() {
		return "", errors.New("unsupported language: " + language)
	}
	return l, nil
}

// Languages lists every language a user can pick; each one needs its own
// email templates (templates/<lang>/) and push texts (internal/i18n).
func Languages() []Language {
	return []Language{LanguageRussian, LanguageUzbek, LanguageEnglish}
}

func (l Language) IsValid() bool {
	switch l {
	case LanguageRussian, LanguageUzbek, LanguageEnglish:
		return true
	default:
		return false
	}
}

func (l Language) String() string {
	return string(l)
}
