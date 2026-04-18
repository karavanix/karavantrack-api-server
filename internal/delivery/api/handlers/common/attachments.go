package common

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/attachments"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/attachments/command"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/attachments/query"
	"github.com/karavanix/karavantrack-api-server/pkg/app"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
)

type attachmentsHandler struct {
	cfg                *config.Config
	attachmentsUsecase *attachments.Usecase
	validator          *validation.Validator
}

func NewAttachmentsHandler(opts *delivery.HandlerOptions) *attachmentsHandler {
	return &attachmentsHandler{
		cfg:                opts.Config,
		validator:          opts.Validator,
		attachmentsUsecase: opts.AttachmentsUsecase,
	}
}

// UploadAttachment godoc
// @Security ApiKeyAuth
// @Summary Upload attachment
// @Tags Attachments
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "File to upload"
// @Param visibility formData string true "Visibility of attachment" enum(public, private)
// @Param folder formData string false "Folder to upload attachment"
// @Success 200 {object} command.UploadFileResponse
// @Failure 400 {object} outerr.Response "Invalid request"
// @Failure 401 {object} outerr.Response "Unauthorized"
// @Failure 403 {object} outerr.Response "Forbidden"
// @Failure 404 {object} outerr.Response "Not found"
// @Failure 413 {object} outerr.Response "Request entity too large"
// @Failure 500 {object} outerr.Response "Internal server error"
// @Router /attachments/file [post]
func (h *attachmentsHandler) UploadFile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 30<<20) // 30MB limit

		if err := r.ParseForm(); err != nil {
			outerr.RequestEntityTooLarge(w, r, "file size limit exceeded 30MB")
			return
		}

		_, header, err := r.FormFile("file")
		if err != nil {
			outerr.BadRequest(w, r, "failed to parse file: "+err.Error())
			return
		}

		var formData = &command.UploadFileRequest{
			File:       header,
			Visibility: r.FormValue("visibility"),
			Folder:     r.FormValue("folder"),
		}

		if err := h.validator.Validate(formData); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		var response *command.UploadFileResponse
		response, err = h.attachmentsUsecase.Command.UploadFile(ctx, userID, formData)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, response)
	}
}

// UploadImage godoc
// @Security ApiKeyAuth
// @Summary Upload image attachment
// @Tags Attachments
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Image to upload"
// @Param visibility formData string true "Visibility of attachment" enum(public, private)
// @Param folder formData string false "Folder to upload attachment"
// @Param width formData int false "Width of image"
// @Param height formData int false "Height of image"
// @Param compress formData bool false "Compress image"
// @Success 200 {object} command.UploadImageResponse
// @Failure 400 {object} outerr.Response "Invalid request"
// @Failure 401 {object} outerr.Response "Unauthorized"
// @Failure 403 {object} outerr.Response "Forbidden"
// @Failure 404 {object} outerr.Response "Not found"
// @Failure 413 {object} outerr.Response "Request entity too large"
// @Failure 500 {object} outerr.Response "Internal server error"
// @Router /attachments/image [post]
func (h *attachmentsHandler) UploadImage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 30<<20) // 30MB limit

		if err := r.ParseForm(); err != nil {
			outerr.RequestEntityTooLarge(w, r, "file size limit exceeded 30MB")
			return
		}

		_, header, err := r.FormFile("file")
		if err != nil {
			outerr.BadRequest(w, r, "failed to parse file: "+err.Error())
			return
		}

		width, _ := strconv.Atoi(r.FormValue("width"))
		height, _ := strconv.Atoi(r.FormValue("height"))
		compress, _ := strconv.ParseBool(r.FormValue("compress"))

		var formData = &command.UploadImageRequest{
			File:       header,
			Visibility: r.FormValue("visibility"),
			Folder:     r.FormValue("folder"),
			Width:      width,
			Height:     height,
			Compress:   compress,
		}

		if err := h.validator.Validate(formData); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		var response *command.UploadImageResponse
		response, err = h.attachmentsUsecase.Command.UploadImage(ctx, userID, formData)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, response)
	}
}

// GetAttachment godoc
// @Security ApiKeyAuth
// @Summary Get attachment metadata
// @Tags Attachments
// @Produce json
// @Param id path string true "Attachment ID"
// @Success 200 {object} query.AttachmentResponse
// @Failure 400 {object} outerr.Response "Invalid request"
// @Failure 401 {object} outerr.Response "Unauthorized"
// @Failure 403 {object} outerr.Response "Forbidden"
// @Failure 404 {object} outerr.Response "Not found"
// @Failure 500 {object} outerr.Response "Internal server error"
// @Router /attachments/{id} [get]
func (h *attachmentsHandler) GetAttachment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		attachmentID := chi.URLParam(r, "id")

		var response *query.AttachmentResponse
		response, err := h.attachmentsUsecase.Query.Get(ctx, userID, attachmentID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, response)
	}
}

// DownloadAttachment godoc
// @Security ApiKeyAuth
// @Summary Download attachment file
// @Tags Attachments
// @Produce octet-stream
// @Param id path string true "Attachment ID"
// @Success 200 {file} binary
// @Failure 400 {object} outerr.Response "Invalid request"
// @Failure 401 {object} outerr.Response "Unauthorized"
// @Failure 403 {object} outerr.Response "Forbidden"
// @Failure 404 {object} outerr.Response "Not found"
// @Failure 500 {object} outerr.Response "Internal server error"
// @Router /attachments/{id}/download [get]
func (h *attachmentsHandler) DownloadAttachment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, ok := app.UserID[string](r.Context())
		if !ok {
			outerr.Forbidden(w, r, "missing user context")
			return
		}

		attachmentID := chi.URLParam(r, "id")

		response, reader, err := h.attachmentsUsecase.Query.DownloadFile(ctx, userID, attachmentID)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}
		if closer, ok := reader.(io.Closer); ok {
			defer closer.Close()
		}

		w.Header().Set("Content-Type", response.MimeType)
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, response.Filename))
		w.WriteHeader(http.StatusOK)

		io.Copy(w, reader)
	}
}
