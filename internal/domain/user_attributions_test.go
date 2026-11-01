package domain_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
)

func TestNewUserAttributionCleansTags(t *testing.T) {
	userID := uuid.New()
	a := domain.NewUserAttribution(userID, domain.AttributionTags{
		Source:      "  Telegram ",
		Medium:      "CPC",
		Campaign:    "Launch_RU_2026_11",
		Referrer:    "T.me",
		LandingLang: "UZ",
	}, domain.AttributionSourceDirect)

	if a.UserID != userID {
		t.Errorf("UserID = %v, want %v", a.UserID, userID)
	}
	if a.Source != "telegram" || a.Medium != "cpc" || a.Referrer != "t.me" || a.LandingLang != "uz" {
		t.Errorf("source/medium/referrer/lang = %q/%q/%q/%q, want lower-cased and trimmed",
			a.Source, a.Medium, a.Referrer, a.LandingLang)
	}
	// Campaign names keep their case: they are read by people.
	if a.Campaign != "Launch_RU_2026_11" {
		t.Errorf("Campaign = %q, want it unchanged", a.Campaign)
	}
}

func TestNewUserAttributionFallsBackWithoutSource(t *testing.T) {
	a := domain.NewUserAttribution(uuid.New(), domain.AttributionTags{Source: "   ", Medium: "qr"}, domain.AttributionSourceApp)

	if a.Source != domain.AttributionSourceApp {
		t.Errorf("Source = %q, want %q", a.Source, domain.AttributionSourceApp)
	}
	if a.Medium != "qr" {
		t.Errorf("Medium = %q, want the medium kept", a.Medium)
	}
}

func TestNewUserAttributionClipsByCharacters(t *testing.T) {
	a := domain.NewUserAttribution(uuid.New(), domain.AttributionTags{
		Campaign: strings.Repeat("я", 200),
	}, domain.AttributionSourceDirect)

	if got := utf8.RuneCountInString(a.Campaign); got != 128 {
		t.Errorf("Campaign has %d characters, want 128", got)
	}
	if !utf8.ValidString(a.Campaign) {
		t.Error("Campaign was cut mid-rune")
	}
}
