package command

import (
	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
)

// AttributionRequest is where a new user came from, as the client saw it:
// the landing page's utm_* tags, ad click ids, the referring host and the
// landing page's language. Optional on every sign-up; only a newly created
// user gets it recorded.
type AttributionRequest struct {
	Source      string `json:"source"`
	Medium      string `json:"medium"`
	Campaign    string `json:"campaign"`
	Content     string `json:"content"`
	Term        string `json:"term"`
	GCLID       string `json:"gclid"`
	YCLID       string `json:"yclid"`
	Referrer    string `json:"referrer"`
	LandingLang string `json:"landing_lang"`
}

// toDomain builds the attribution for userID; a nil request or an empty
// source records fallbackSource.
func (a *AttributionRequest) toDomain(userID uuid.UUID, fallbackSource string) *domain.UserAttribution {
	if a == nil {
		return domain.NewUserAttribution(userID, domain.AttributionTags{}, fallbackSource)
	}
	return domain.NewUserAttribution(userID, domain.AttributionTags{
		Source:      a.Source,
		Medium:      a.Medium,
		Campaign:    a.Campaign,
		Content:     a.Content,
		Term:        a.Term,
		GCLID:       a.GCLID,
		YCLID:       a.YCLID,
		Referrer:    a.Referrer,
		LandingLang: a.LandingLang,
	}, fallbackSource)
}
