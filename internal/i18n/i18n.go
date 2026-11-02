// Package i18n holds the push notification texts in every language a user
// can pick (users.language). Code refers to a text by its Key and renders it
// in the recipient's language at the moment of sending, so a usecase that
// enqueues a push never has to know who reads it in what language. Emails
// are whole HTML templates per language instead, in templates/<lang>/.
package i18n

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
)

// Key names one text of the catalog.
type Key string

// ParamTitle is the parameter every load push carries: the load's title.
const ParamTitle = "title"

// Push is a push notification rendered in one language.
type Push struct {
	Title string
	Body  string
}

// RenderPush renders the push text key in lang with params (see ParamTitle).
// An unknown key or a text missing a parameter is an error: a push with a
// hole in it must not reach a phone.
func RenderPush(key Key, lang shared.Language, params map[string]string) (Push, error) {
	texts, ok := pushes[key]
	if !ok {
		return Push{}, fmt.Errorf("i18n: unknown push key %q", key)
	}
	text, ok := texts[lang]
	if !ok {
		text = texts[shared.DefaultLanguage]
	}

	title, err := render(string(key)+".title", text.title, params)
	if err != nil {
		return Push{}, err
	}
	body, err := render(string(key)+".body", text.body, params)
	if err != nil {
		return Push{}, err
	}
	return Push{Title: title, Body: body}, nil
}

// render executes text as a text/template over params. A reference to a
// parameter that was not passed is an error rather than "<no value>".
func render(name, text string, params map[string]string) (string, error) {
	tmpl, err := template.New(name).Option("missingkey=error").Parse(text)
	if err != nil {
		return "", fmt.Errorf("i18n: parse %s: %w", name, err)
	}
	if params == nil {
		params = map[string]string{}
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params); err != nil {
		return "", fmt.Errorf("i18n: render %s: %w", name, err)
	}
	return buf.String(), nil
}
