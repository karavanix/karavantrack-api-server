package shared

import (
	"errors"
	"regexp"
	"strings"
)

type Phone string

var phoneRegex = regexp.MustCompile(`^\d{10,15}$`)

func NewPhone(phone string) (Phone, error) {
	phone = strings.TrimPrefix(phone, "+")

	// Validate using regex (covers both digit-only check and length 10-15)
	if !phoneRegex.MatchString(phone) {
		return "", errors.New("invalid phone number format: " + phone)
	}

	return Phone(phone), nil
}

func (p Phone) IsValid() bool {
	return phoneRegex.MatchString(string(p))
}

func (p Phone) String() string {
	return string(p)
}
