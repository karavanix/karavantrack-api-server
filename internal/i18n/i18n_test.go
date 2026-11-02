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
		for _, lang := range shared.Languages() {
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
