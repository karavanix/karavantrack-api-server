package command

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/h2non/bimg"
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

// UploadImageUsecase handles image upload with optional compression/resizing.
type UploadImageUsecase struct {
	contextDuration time.Duration
	cfg             *config.Config
	txManager       postgres.TxManager
	attachmentsRepo domain.AttachmentRepository
	s3              *s3.S3Client
	publicBucket    string
	privateBucket   string
}

func NewUploadImageUsecase(
	contextDuration time.Duration,
	cfg *config.Config,
	txManager postgres.TxManager,
	attachmentsRepo domain.AttachmentRepository,
	s3Client *s3.S3Client,
) *UploadImageUsecase {
	return &UploadImageUsecase{
		contextDuration: contextDuration,
		cfg:             cfg,
		txManager:       txManager,
		attachmentsRepo: attachmentsRepo,
		s3:              s3Client,
		publicBucket:    cfg.S3.PublicBucket,
		privateBucket:   cfg.S3.PrivateBucket,
	}
}

type UploadImageRequest struct {
	File       *multipart.FileHeader `form:"file"       validate:"required"`
	Visibility string                `form:"visibility" validate:"required,oneof=public private"`
	Folder     string                `form:"folder"`
	Width      int                   `form:"width"`
	Height     int                   `form:"height"`
	Compress   bool                  `form:"compress"`
}

type UploadImageResponse struct {
	ID         string
	Filename   string
	MimeType   string
	Visibility string
	Status     string
	PublicURL  string
	CreatedAt  time.Time
}

func (u *UploadImageUsecase) UploadImage(ctx context.Context, requesterID string, req *UploadImageRequest) (_ *UploadImageResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("internal.usecases.attachments"), "domain.UploadImage",
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
		width      int
		height     int
		compress   bool
	}
	{
		input.userID, err = uuid.Parse(requesterID)
		if err != nil {
			return nil, inerr.NewErrValidation("user_id", "user id must be uuid")
		}

		input.fileName = strings.TrimSuffix(filepath.Base(req.File.Filename), filepath.Ext(req.File.Filename))
		input.fileExt = strings.TrimPrefix(filepath.Ext(req.File.Filename), ".")
		input.fileMime = utils.DetectMimeByExt(input.fileExt)

		input.visibility = domain.Visibility(req.Visibility)
		if !input.visibility.IsValid() {
			return nil, inerr.NewErrValidation("visibility", "visibility must be public or private")
		}

		input.folder = req.Folder
		if input.folder == "" {
			input.folder = "general"
		}

		input.width = req.Width
		input.height = req.Height
		input.compress = req.Compress
	}

	// Prepare file for upload
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

	var fileReader io.ReadSeeker = srcFile
	if input.compress {
		rawBytes, err := io.ReadAll(fileReader)
		if err != nil {
			logger.ErrorContext(ctx, "failed to read file", err)
			return nil, err
		}

		imageBytes, err := bimg.NewImage(rawBytes).Process(bimg.Options{
			Width:         input.width,
			Height:        input.height,
			Quality:       80,
			Type:          bimg.WEBP,
			NoAutoRotate:  false,
			StripMetadata: true,
			Crop:          true,
			Gravity:       bimg.GravitySmart,
		})
		if err != nil {
			logger.ErrorContext(ctx, "failed to compress image", err)
			return nil, err
		}
		fileReader = bytes.NewReader(imageBytes)

		input.fileExt = "webp"
		input.fileMime = utils.DetectMimeByExt(input.fileExt)
	}

	dstFile, err := os.Create(filepath.Join(dst, fileName))
	if err != nil {
		logger.ErrorContext(ctx, "failed to create file", err)
		return nil, err
	}
	defer func() {
		_ = dstFile.Close()
		_ = os.Remove(filepath.Join(dst, fileName))
	}()

	fileSize, err := io.Copy(dstFile, fileReader)
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
		logger.ErrorContext(ctx, "failed to put image object", err)
	} else {
		attachment.MarkUploaded()
	}

	if err = u.attachmentsRepo.Save(ctx, attachment); err != nil {
		logger.ErrorContext(ctx, "failed to save attachment", err)
		return nil, err
	}

	return &UploadImageResponse{
		ID:         attachment.ID.String(),
		Filename:   attachment.FileNameWithExtension(),
		MimeType:   attachment.FileMime,
		Visibility: attachment.Visibility.String(),
		Status:     attachment.Status.String(),
		PublicURL:  utils.If(attachment.IsPublic(), fmt.Sprintf("%s/%s/%s", u.s3.EndpointURL(), attachment.ObjectBucket, attachment.ObjectKey), ""),
		CreatedAt:  attachment.CreatedAt,
	}, nil
}
