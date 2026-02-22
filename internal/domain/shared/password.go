package shared

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type Password string

func NewPassword(password string) (Password, error) {
	if len(password) < 8 {
		return "", errors.New("password must be at least 8 characters long")
	}
	return Password(password), nil
}

func (p Password) Hash() string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func (p Password) Verify(hashedPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(p))
}

func (p Password) String() string {
	return string(p)
}
