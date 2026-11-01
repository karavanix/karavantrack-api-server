package domain

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	// AttributionSourceDirect is the source of a user who came with no tags.
	AttributionSourceDirect = "direct"
	// AttributionSourceApp is the source of a user who signed up in the
	// mobile app without tags (/auth/telegram, /auth/apple are app-only).
	AttributionSourceApp = "app"
)

// AttributionTags are the marketing tags a client carried to sign-up: the
// landing page's utm_* values, ad click ids, the referring host and the
// landing page's language.
type AttributionTags struct {
	Source      string
	Medium      string
	Campaign    string
	Content     string
	Term        string
	GCLID       string
	YCLID       string
	Referrer    string
	LandingLang string
}

// UserAttribution records where a user came from. It is written once, when
// the user is created, and never overwritten.
type UserAttribution struct {
	UserID uuid.UUID
	AttributionTags
	CreatedAt time.Time
}

// NewUserAttribution cleans what the client sent: trims and clips every tag
// to its column, lower-cases source, medium, referrer and language, and puts
// fallbackSource in place of an empty source.
func NewUserAttribution(userID uuid.UUID, tags AttributionTags, fallbackSource string) *UserAttribution {
	clean := AttributionTags{
		Source:      strings.ToLower(clip(tags.Source, 128)),
		Medium:      strings.ToLower(clip(tags.Medium, 128)),
		Campaign:    clip(tags.Campaign, 128),
		Content:     clip(tags.Content, 128),
		Term:        clip(tags.Term, 128),
		GCLID:       clip(tags.GCLID, 256),
		YCLID:       clip(tags.YCLID, 256),
		Referrer:    strings.ToLower(clip(tags.Referrer, 255)),
		LandingLang: strings.ToLower(clip(tags.LandingLang, 8)),
	}
	if clean.Source == "" {
		clean.Source = fallbackSource
	}

	return &UserAttribution{
		UserID:          userID,
		AttributionTags: clean,
		CreatedAt:       time.Now(),
	}
}

// clip trims s and cuts it to at most n characters (not bytes, so a
// Cyrillic campaign name is never split mid-rune).
func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

type UserAttributionRepository interface {
	// Save inserts the attribution; a user who already has one keeps it.
	Save(ctx context.Context, attribution *UserAttribution) error
}
