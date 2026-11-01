package i18n

import (
	"fmt"
	"strconv"

	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
)

// EmailKind names one of the emails the server sends.
type EmailKind string

const (
	EmailVerification  EmailKind = "verification"
	EmailPasswordReset EmailKind = "password_reset"
)

// ParamMinutes is how long the code in an email stays valid.
const ParamMinutes = "minutes"

// EmailText is the wording of one email in one language. The sentence about
// the code's lifetime is split in three so the template can put the duration
// in bold wherever the language places it.
type EmailText struct {
	Subject    string
	Heading    string
	BodyBefore string // "Enter the code below… The code is valid for "
	BodyStrong string // "15 minutes"
	BodyAfter  string // "."
	Footnote   string
	Footer     string
}

// RenderEmail returns the wording of kind in lang for a code that lives
// minutes minutes.
func RenderEmail(kind EmailKind, lang shared.Language, minutes int) (EmailText, error) {
	texts, ok := emails[kind]
	if !ok {
		return EmailText{}, fmt.Errorf("i18n: unknown email kind %q", kind)
	}
	text, ok := texts[lang]
	if !ok {
		text = texts[shared.DefaultLanguage]
	}
	footer, ok := footers[lang]
	if !ok {
		footer = footers[shared.DefaultLanguage]
	}

	params := map[string]string{ParamMinutes: strconv.Itoa(minutes)}
	strong, err := render(string(kind)+".body_strong", text.BodyStrong, params)
	if err != nil {
		return EmailText{}, err
	}
	text.BodyStrong = strong
	text.Footer = footer
	return text, nil
}

var footers = map[shared.Language]string{
	shared.LanguageRussian: "© 2026 Yool · Все права защищены",
	shared.LanguageUzbek:   "© 2026 Yool · Barcha huquqlar himoyalangan",
	shared.LanguageEnglish: "© 2026 Yool · All rights reserved",
}

var emails = map[EmailKind]map[shared.Language]EmailText{
	EmailVerification: {
		shared.LanguageRussian: {
			Subject:    "Подтверждение почты — Yool",
			Heading:    "Подтвердите ваш email",
			BodyBefore: "Введите код ниже, чтобы завершить регистрацию. Код действителен в течение ",
			BodyStrong: "{{.minutes}} минут",
			BodyAfter:  ".",
			Footnote:   "Не запрашивали это? Просто проигнорируйте письмо — никаких действий не требуется.",
		},
		shared.LanguageUzbek: {
			Subject:    "Emailni tasdiqlash — Yool",
			Heading:    "Emailingizni tasdiqlang",
			BodyBefore: "Roʻyxatdan oʻtishni yakunlash uchun quyidagi kodni kiriting. Kod ",
			BodyStrong: "{{.minutes}} daqiqa",
			BodyAfter:  " amal qiladi.",
			Footnote:   "Bu siz emasmi? Xatni eʼtiborsiz qoldiring — hech narsa qilish shart emas.",
		},
		shared.LanguageEnglish: {
			Subject:    "Confirm your email — Yool",
			Heading:    "Confirm your email",
			BodyBefore: "Enter the code below to finish signing up. The code is valid for ",
			BodyStrong: "{{.minutes}} minutes",
			BodyAfter:  ".",
			Footnote:   "Didn't request this? Just ignore this email — no action is needed.",
		},
	},
	EmailPasswordReset: {
		shared.LanguageRussian: {
			Subject:    "Сброс пароля — Yool",
			Heading:    "Код для сброса пароля",
			BodyBefore: "Введите код ниже, чтобы задать новый пароль. Код действителен в течение ",
			BodyStrong: "{{.minutes}} минут",
			BodyAfter:  ".",
			Footnote:   "Не запрашивали сброс? Просто проигнорируйте письмо — пароль останется прежним.",
		},
		shared.LanguageUzbek: {
			Subject:    "Parolni tiklash — Yool",
			Heading:    "Parolni tiklash kodi",
			BodyBefore: "Yangi parol oʻrnatish uchun quyidagi kodni kiriting. Kod ",
			BodyStrong: "{{.minutes}} daqiqa",
			BodyAfter:  " amal qiladi.",
			Footnote:   "Parolni tiklashni soʻramaganmisiz? Xatni eʼtiborsiz qoldiring — parol oʻzgarmaydi.",
		},
		shared.LanguageEnglish: {
			Subject:    "Reset your password — Yool",
			Heading:    "Password reset code",
			BodyBefore: "Enter the code below to set a new password. The code is valid for ",
			BodyStrong: "{{.minutes}} minutes",
			BodyAfter:  ".",
			Footnote:   "Didn't request a reset? Just ignore this email — your password stays the same.",
		},
	},
}
