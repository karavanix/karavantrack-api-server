package command

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"github.com/karavanix/karavantrack-api-server/pkg/s3"
	"github.com/karavanix/karavantrack-api-server/pkg/utils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// UploadFileUsecase handles arbitrary file uploads to S3.
type UploadFileUsecase struct {
	contextDuration time.Duration
	cfg             *config.Config
	txManager       postgres.TxManager
	attachmentsRepo domain.AttachmentRepository
	s3              *s3.S3Client
	publicBucket    string
	privateBucket   string
}

func NewUploadFileUsecase(
	contextDuration time.Duration,
	cfg *config.Config,
	txManager postgres.TxManager,
	attachmentsRepo domain.AttachmentRepository,
	s3Client *s3.S3Client,
) *UploadFileUsecase {
	return &UploadFileUsecase{
		contextDuration: contextDuration,
		cfg:             cfg,
		txManager:       txManager,
		attachmentsRepo: attachmentsRepo,
		s3:              s3Client,
		publicBucket:    cfg.S3.PublicBucket,
		privateBucket:   cfg.S3.PrivateBucket,
	}
}

type UploadFileRequest struct {
	File       *multipart.FileHeader `form:"file"      validate:"required"`
	Visibility string                `form:"visibility" validate:"required,oneof=public private"`
	Folder     string                `form:"folder"`
}

type UploadFileResponse struct {
	ID         string
	Filename   string
	MimeType   string
	Visibility string
	Status     string
	PublicURL  string
	CreatedAt  time.Time
}

func (u *UploadFileUsecase) UploadFile(ctx context.Context, requesterID string, req *UploadFileRequest) (_ *UploadFileResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("internal.usecases.attachments"), "domain.UploadFile",
		attribute.String("user_id", requesterID),
	)
	defer func() { end(err) }()

	var input struct {
		userID     uuid.UUID
		fileName   string
		fileExt    string
		fileMime   string
		visibility domain.Visibility
		folder     string
	}
	{
		input.userID, err = uuid.Parse(requesterID)
		if err != nil {
			return nil, inerr.NewErrValidation("user_id", "user id must be uuid")
		}

		input.fileName = strings.TrimSuffix(filepath.Base(req.File.Filename), filepath.Ext(req.File.Filename))
		input.fileExt = strings.TrimPrefix(filepath.Ext(req.File.Filename), ".")
		input.fileMime = req.File.Header.Get("Content-Type")
		if input.fileMime == "" {
			input.fileMime = utils.DetectMimeByExt(input.fileExt)
		}

		input.visibility = domain.Visibility(req.Visibility)
		if !input.visibility.IsValid() {
			return nil, inerr.NewErrValidation("visibility", "visibility must be public or private")
		}

		input.folder = req.Folder
		if input.folder == "" {
			input.folder = "general"
		}
	}

	fileID := uuid.New()
	fileName := fmt.Sprintf("%s_%s.%s", fileID.String(), input.fileName, input.fileExt)
	objectKey := fmt.Sprintf("%s/%s/%s", input.folder, input.userID.String(), fileName)
	objectBucket := utils.If(input.visibility == domain.VisibilityPublic, u.publicBucket, u.privateBucket)
	dst, _ := os.Getwd()

	srcFile, err := req.File.Open()
	if err != nil {
		logger.ErrorContext(ctx, "failed to open file", err)
		return nil, err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(filepath.Join(dst, fileName))
	if err != nil {
		logger.ErrorContext(ctx, "failed to create file", err)
		return nil, err
	}
	defer func() {
		_ = dstFile.Close()
		_ = os.Remove(filepath.Join(dst, fileName))
	}()

	fileSize, err := io.Copy(dstFile, srcFile)
	if err != nil {
		logger.ErrorContext(ctx, "failed to copy file", err)
		return nil, err
	}

	attachment, err := domain.NewAttachment(
		fileID,
		input.userID,
		input.visibility,
		input.fileName,
		input.fileExt,
		fileSize,
		input.fileMime,
		objectBucket,
		objectKey,
	)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create attachment", err)
		return nil, err
	}

	if err = u.attachmentsRepo.Save(ctx, attachment); err != nil {
		logger.ErrorContext(ctx, "failed to save attachment", err)
		return nil, err
	}

	_, _ = dstFile.Seek(0, io.SeekStart)
	_ = u.s3.EnsureBucket(ctx, objectBucket)
	_, err = u.s3.PutObject(ctx, objectBucket, objectKey, dstFile, fileSize, s3.PutOptions{
		ContentType: input.fileMime,
	})
	if err != nil {
		attachment.MarkFailed()
		logger.ErrorContext(ctx, "failed to put object", err)
	} else {
		attachment.MarkUploaded()
	}

	if err = u.attachmentsRepo.Save(ctx, attachment); err != nil {
		logger.ErrorContext(ctx, "failed to save attachment", err)
		return nil, err
	}

	return &UploadFileResponse{
		ID:         attachment.ID.String(),
		Filename:   attachment.FileNameWithExtension(),
		MimeType:   attachment.FileMime,
		Visibility: attachment.Visibility.String(),
		Status:     attachment.Status.String(),
		PublicURL:  utils.If(attachment.IsPublic(), fmt.Sprintf("%s/%s/%s", u.s3.EndpointURL(), attachment.ObjectBucket, attachment.ObjectKey), ""),
		CreatedAt:  attachment.CreatedAt,
	}, nil
}
