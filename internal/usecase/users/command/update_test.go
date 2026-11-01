package command_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/users/command"
)

// profileUsers implements domain.UserRepository; only FindByID and Update are used.
type profileUsers struct {
	domain.UserRepository
	user    *domain.User
	updated *domain.User
}

func (r *profileUsers) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return r.user, nil
}

func (r *profileUsers) Update(ctx context.Context, u *domain.User) error {
	r.updated = u
	return nil
}

func newProfileUser(t *testing.T) *domain.User {
	t.Helper()
	user, err := domain.NewUserFromTelegram("Ali", "Valiev", shared.Phone("998901234567"), shared.RoleShipper)
	if err != nil {
		t.Fatalf("NewUserFromTelegram: %v", err)
	}
	return user
}

func TestUpdateChangesLanguage(t *testing.T) {
	repo := &profileUsers{user: newProfileUser(t)}

	err := command.NewUpdateUsecase(time.Second, repo).Update(context.Background(), repo.user.ID.String(),
		&command.UpdateRequest{Language: "uz"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if repo.updated == nil || repo.updated.Language != shared.LanguageUzbek {
		t.Fatalf("updated user = %+v, want language uz", repo.updated)
	}
	// Names left empty in the request stay as they were.
	if repo.updated.FirstName != "Ali" || repo.updated.LastName != "Valiev" {
		t.Errorf("names = %q %q, want them kept", repo.updated.FirstName, repo.updated.LastName)
	}
}

func TestUpdateWithoutLanguageKeepsIt(t *testing.T) {
	repo := &profileUsers{user: newProfileUser(t)}
	repo.user.ChangeLanguage(shared.LanguageEnglish)

	err := command.NewUpdateUsecase(time.Second, repo).Update(context.Background(), repo.user.ID.String(),
		&command.UpdateRequest{FirstName: "Vali"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if repo.updated.Language != shared.LanguageEnglish {
		t.Errorf("Language = %q, want en kept", repo.updated.Language)
	}
}

func TestUpdateRejectsUnsupportedLanguage(t *testing.T) {
	repo := &profileUsers{user: newProfileUser(t)}

	err := command.NewUpdateUsecase(time.Second, repo).Update(context.Background(), repo.user.ID.String(),
		&command.UpdateRequest{Language: "de"})

	var validation inerr.ErrValidation
	if !errors.As(err, &validation) {
		t.Fatalf("Update = %v, want a validation error", err)
	}
	if repo.updated != nil {
		t.Error("user was saved despite the invalid language")
	}
}

func TestNewUsersSpeakRussianByDefault(t *testing.T) {
	if got := newProfileUser(t).Language; got != shared.DefaultLanguage {
		t.Errorf("Language = %q, want %q", got, shared.DefaultLanguage)
	}
}
