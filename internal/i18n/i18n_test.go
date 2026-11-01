package i18n

import (
	"strings"
	"testing"

	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
)

// Every push exists in every language, renders with its parameter and leaves
// no template syntax behind.
func TestPushesCompleteInEveryLanguage(t *testing.T) {
	params := map[string]string{ParamTitle: "Хлопок в Фергану"}
	for key := range pushes {
		for _, lang := range Languages() {
			if _, ok := pushes[key][lang]; !ok {
				t.Errorf("%s has no %s text", key, lang)
				continue
			}
			push, err := RenderPush(key, lang, params)
			if err != nil {
				t.Errorf("%s/%s: %v", key, lang, err)
				continue
			}
			if push.Title == "" || push.Body == "" {
				t.Errorf("%s/%s rendered empty: %+v", key, lang, push)
			}
			if !strings.Contains(push.Body, params[ParamTitle]) {
				t.Errorf("%s/%s body lacks the load title: %q", key, lang, push.Body)
			}
			if strings.Contains(push.Title+push.Body, "{{") {
				t.Errorf("%s/%s left template syntax: %+v", key, lang, push)
			}
		}
	}
}

func TestRenderPushErrors(t *testing.T) {
	if _, err := RenderPush("push.nope", shared.LanguageRussian, nil); err == nil {
		t.Error("unknown key rendered without error")
	}
	if _, err := RenderPush(PushLoadAssigned, shared.LanguageRussian, nil); err == nil {
		t.Error("missing parameter rendered without error")
	}
}

// A language the catalog does not know falls back to Russian rather than
// failing: the push still reaches the person.
func TestRenderPushFallsBackToRussian(t *testing.T) {
	got, err := RenderPush(PushLoadAssigned, shared.Language("de"), map[string]string{ParamTitle: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Новый груз" {
		t.Errorf("title = %q, want the Russian one", got.Title)
	}
}

func TestEmailsCompleteInEveryLanguage(t *testing.T) {
	for kind := range emails {
		for _, lang := range Languages() {
			if _, ok := emails[kind][lang]; !ok {
				t.Errorf("%s has no %s text", kind, lang)
				continue
			}
			text, err := RenderEmail(kind, lang, 15)
			if err != nil {
				t.Errorf("%s/%s: %v", kind, lang, err)
				continue
			}
			for name, s := range map[string]string{
				"Subject": text.Subject, "Heading": text.Heading, "BodyBefore": text.BodyBefore,
				"BodyStrong": text.BodyStrong, "BodyAfter": text.BodyAfter, "Footnote": text.Footnote, "Footer": text.Footer,
			} {
				if s == "" {
					t.Errorf("%s/%s: %s is empty", kind, lang, name)
				}
				if strings.Contains(s, "{{") {
					t.Errorf("%s/%s: %s left template syntax: %q", kind, lang, name, s)
				}
			}
			if !strings.Contains(text.BodyStrong, "15") {
				t.Errorf("%s/%s: BodyStrong lacks the minutes: %q", kind, lang, text.BodyStrong)
			}
		}
	}
}
