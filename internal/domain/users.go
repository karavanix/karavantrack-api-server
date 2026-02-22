package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
)

type UserStatus string

const (
	UserStatusPending  UserStatus = "pending"
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
)

func (u UserStatus) String() string {
	return string(u)
}

type User struct {
	ID           uuid.UUID
	FirstName    string
	LastName     string
	Email        shared.Email
	Phone        shared.Phone
	PasswordHash string
	Status       UserStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewUser(
	firstName string,
	lastName string,
	email shared.Email,
	phone shared.Phone,
	password shared.Password,
) (*User, error) {
	if email == "" && phone == "" {
		return nil, errors.New("email or phone is required")
	}

	return &User{
		ID:           uuid.New(),
		FirstName:    firstName,
		LastName:     lastName,
		Email:        email,
		Phone:        phone,
		PasswordHash: password.Hash(),
		Status:       UserStatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

func (u *User) ChangePassword(password shared.Password) error {
	u.PasswordHash = password.Hash()
	u.UpdatedAt = time.Now()
	return nil
}

func (u *User) Activate() error {
	u.Status = UserStatusActive
	u.UpdatedAt = time.Now()
	return nil
}

func (u *User) Deactivate() error {
	u.Status = UserStatusInactive
	u.UpdatedAt = time.Now()
	return nil
}

type UserRepository interface {
	Save(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email shared.Email) (*User, error)
	FindByPhone(ctx context.Context, phone shared.Phone) (*User, error)
	FindByEmailOrPhone(ctx context.Context, email shared.Email, phone shared.Phone) (*User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
}
