package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Visibility string

const (
	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"
)

func (v Visibility) IsValid() bool {
	switch v {
	case VisibilityPublic, VisibilityPrivate:
		return true
	default:
		return false
	}
}

func (v Visibility) String() string {
	return string(v)
}

type Status string

const (
	StatusPending  Status = "pending"
	StatusUploaded Status = "uploaded"
	StatusFailed   Status = "failed"
)

func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusUploaded, StatusFailed:
		return true
	default:
		return false
	}
}

func (s Status) String() string {
	return string(s)
}

type Attachment struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Visibility   Visibility
	FileName     string
	FileExt      string
	FileSize     int64
	FileMime     string
	Status       Status
	ObjectBucket string
	ObjectKey    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

func NewAttachment(
	id uuid.UUID,
	userID uuid.UUID,
	visibility Visibility,
	fileName string,
	fileExt string,
	fileSize int64,
	fileMime string,
	objectBucket string,
	objectKey string,
) (*Attachment, error) {
	if id == uuid.Nil {
		return nil, errors.New("ID cannot be nil")
	}
	if userID == uuid.Nil {
		return nil, errors.New("user ID cannot be nil")
	}
	if fileName == "" {
		return nil, errors.New("file name cannot be empty")
	}
	if fileExt == "" {
		return nil, errors.New("file extension cannot be empty")
	}
	if fileSize <= 0 {
		return nil, errors.New("file size must be positive")
	}
	if fileMime == "" {
		return nil, errors.New("file MIME type cannot be empty")
	}
	if !visibility.IsValid() {
		return nil, errors.New("invalid visibility")
	}
	if objectBucket == "" {
		return nil, errors.New("object bucket cannot be empty")
	}
	if objectKey == "" {
		return nil, errors.New("object key cannot be empty")
	}

	now := time.Now()
	return &Attachment{
		ID:           id,
		UserID:       userID,
		Visibility:   visibility,
		FileName:     fileName,
		FileExt:      fileExt,
		FileSize:     fileSize,
		FileMime:     fileMime,
		Status:       StatusPending,
		ObjectBucket: objectBucket,
		ObjectKey:    objectKey,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (a *Attachment) UpdateStatus(status Status) error {
	if !status.IsValid() {
		return errors.New("invalid status")
	}
	a.Status = status
	a.UpdatedAt = time.Now()
	return nil
}

func (a *Attachment) UpdateVisibility(visibility Visibility) error {
	if !visibility.IsValid() {
		return errors.New("invalid visibility")
	}
	a.Visibility = visibility
	a.UpdatedAt = time.Now()
	return nil
}

func (a *Attachment) MarkUploaded() {
	a.Status = StatusUploaded
	a.UpdatedAt = time.Now()
}

func (a *Attachment) MarkFailed() {
	a.Status = StatusFailed
	a.UpdatedAt = time.Now()
}

func (a *Attachment) MarkDeleted() {
	now := time.Now()
	a.DeletedAt = &now
	a.UpdatedAt = now
}

func (a *Attachment) IsOwner(userID uuid.UUID) bool {
	return a.UserID == userID
}

func (a *Attachment) IsPublic() bool {
	return a.Visibility == VisibilityPublic
}

func (a *Attachment) IsDeleted() bool {
	return a.DeletedAt != nil
}

func (a *Attachment) FileNameWithExtension() string {
	return a.FileName + "." + a.FileExt
}

func (a *Attachment) IsUploaded() bool {
	return a.Status == StatusUploaded
}

func (a *Attachment) AttachmentURL() string {
	return a.ObjectBucket + "/" + a.ObjectKey
}

type AttachmentRepository interface {
	Save(ctx context.Context, attachment *Attachment) error
	FindByID(ctx context.Context, id uuid.UUID) (*Attachment, error)
	FindByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*Attachment, error)
	FindAllByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Attachment, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
