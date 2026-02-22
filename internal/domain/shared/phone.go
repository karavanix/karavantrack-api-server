package shared

import (
	"errors"
	"strings"
)

type Phone string

func NewPhone(phone string) (Phone, error) {
	phone = strings.TrimPrefix(phone, "+")

	if len(phone) < 10 || len(phone) > 15 {
		return "", errors.New("invalid phone number: " + phone)
	}
	return Phone(phone), nil
}

func (p Phone) IsValid() bool {
	return len(p) >= 10 && len(p) <= 15
}

func (p Phone) String() string {
	return string(p)
}
