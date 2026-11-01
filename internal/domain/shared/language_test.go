package shared_test

import (
	"testing"

	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
)

func TestNewLanguage(t *testing.T) {
	tests := []struct {
		in      string
		want    shared.Language
		wantErr bool
	}{
		{in: "ru", want: shared.LanguageRussian},
		{in: "uz", want: shared.LanguageUzbek},
		{in: "en", want: shared.LanguageEnglish},
		{in: " EN ", want: shared.LanguageEnglish},
		{in: "uz-Latn-UZ", want: shared.LanguageUzbek},
		{in: "en_US", want: shared.LanguageEnglish},
		{in: "de", wantErr: true},
		{in: "", wantErr: true},
	}
	for _, tt := range tests {
		got, err := shared.NewLanguage(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("NewLanguage(%q) = %q, want an error", tt.in, got)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("NewLanguage(%q) = %q, %v; want %q", tt.in, got, err, tt.want)
		}
	}
}
