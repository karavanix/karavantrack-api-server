package shared

import (
	"errors"
	"net/mail"
)


type Email string

func NewEmail(email string) (Email, error) {
	if _, err := mail.ParseAddress(email); err != nil {
		return "", errors.New("invalid email address: " + err.Error())
	}
	return Email(email), nil
}

func (e Email) IsValid() bool {
	_, err := mail.ParseAddress(string(e))
	return err == nil
}

func (e Email) String() string {
	return string(e)
}