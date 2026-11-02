package email

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"
	"strings"

	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
)

// Template file names; every language has its own copy of each under
// <dir>/<lang>/.
const (
	TemplateVerification  = "email_verification.html"
	TemplatePasswordReset = "email_password_reset.html"
)

// subjectBlock is the {{define}} block each template file declares with its
// email's subject line.
const subjectBlock = "subject"

var templateNames = []string{TemplateVerification, TemplatePasswordReset}

// Templates holds every email template in every language. Each file is
// parsed on its own, so each can define its own "subject" block.
type Templates struct {
	byLang map[shared.Language]map[string]*template.Template
}

// LoadTemplates parses <dir>/<lang>/<name> for every language a user can
// pick and every email. A missing file or one without a subject is an error,
// so a language without its emails fails at startup, not at a user's
// registration.
func LoadTemplates(dir string) (*Templates, error) {
	t := &Templates{byLang: make(map[shared.Language]map[string]*template.Template)}
	for _, lang := range shared.Languages() {
		t.byLang[lang] = make(map[string]*template.Template)
		for _, name := range templateNames {
			path := filepath.Join(dir, lang.String(), name)
			tmpl, err := template.ParseFiles(path)
			if err != nil {
				return nil, fmt.Errorf("email template %s: %w", path, err)
			}
			if tmpl.Lookup(subjectBlock) == nil {
				return nil, fmt.Errorf("email template %s: no {{define %q}} block", path, subjectBlock)
			}
			t.byLang[lang][name] = tmpl
		}
	}
	return t, nil
}

// render returns the subject and the HTML body of the email name in lang;
// a language without templates falls back to Russian.
func (t *Templates) render(name string, lang shared.Language, data any) (string, string, error) {
	set, ok := t.byLang[lang]
	if !ok {
		set = t.byLang[shared.DefaultLanguage]
	}
	tmpl, ok := set[name]
	if !ok {
		return "", "", fmt.Errorf("email template %s/%s not loaded", lang, name)
	}

	var subject, body bytes.Buffer
	if err := tmpl.ExecuteTemplate(&subject, subjectBlock, data); err != nil {
		return "", "", err
	}
	if err := tmpl.ExecuteTemplate(&body, name, data); err != nil {
		return "", "", err
	}
	return strings.TrimSpace(subject.String()), body.String(), nil
}
