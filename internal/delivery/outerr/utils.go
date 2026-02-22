package outerr

import (
	"net/http"

	"github.com/karavanix/karavantrack-api-server/pkg/wsrouter"
)

// Convenience helpers for the most common ad-hoc responses.

func BadRequest(w http.ResponseWriter, r *http.Request, msg string, details ...any) {
	opts := []Option{
		WithStatus(http.StatusBadRequest),
		WithCode(CodeBadRequest),
		WithMessage(msg),
	}
	if len(details) > 0 && details[0] != nil {
		opts = append(opts, WithDetails(details[0]))
	}
	HandleHTTP(w, r, nil, opts...)
}

func Unauthorized(w http.ResponseWriter, r *http.Request, msg string, details ...any) {
	opts := []Option{
		WithStatus(http.StatusUnauthorized),
		WithCode(CodeUnauthorized),
	}
	if msg != "" {
		opts = append(opts, WithMessage(msg))
	}
	if len(details) > 0 && details[0] != nil {
		opts = append(opts, WithDetails(details[0]))
	}
	HandleHTTP(w, r, nil, opts...)
}

func Forbidden(w http.ResponseWriter, r *http.Request, msg string) {
	HandleHTTP(w, r, nil,
		WithStatus(http.StatusForbidden),
		WithCode(CodeForbidden),
		WithMessage(msg),
	)
}

func NotFound(w http.ResponseWriter, r *http.Request, msg string, details ...any) {
	opts := []Option{
		WithStatus(http.StatusNotFound),
		WithCode(CodeNotFound),
	}
	if msg != "" {
		opts = append(opts, WithMessage(msg))
	}
	if len(details) > 0 && details[0] != nil {
		opts = append(opts, WithDetails(details[0]))
	}
	HandleHTTP(w, r, nil, opts...)
}

func Conflict(w http.ResponseWriter, r *http.Request, msg string, details ...any) {
	opts := []Option{
		WithStatus(http.StatusConflict),
		WithCode(CodeConflict),
		WithMessage(msg),
	}
	if len(details) > 0 && details[0] != nil {
		opts = append(opts, WithDetails(details[0]))
	}
	HandleHTTP(w, r, nil, opts...)
}

func Unprocessable(w http.ResponseWriter, r *http.Request, msg string, details ...any) {
	opts := []Option{
		WithStatus(http.StatusUnprocessableEntity),
		WithCode(CodeUnprocessableEntity),
		WithMessage(msg),
	}
	if len(details) > 0 && details[0] != nil {
		opts = append(opts, WithDetails(details[0]))
	}
	HandleHTTP(w, r, nil, opts...)
}

func TooManyRequests(w http.ResponseWriter, r *http.Request, msg string) {
	HandleHTTP(w, r, nil,
		WithStatus(http.StatusTooManyRequests),
		WithCode(CodeTooManyRequests),
		WithMessage(msg),
	)
}

func RequestEntityTooLarge(w http.ResponseWriter, r *http.Request, msg string) {
	HandleHTTP(w, r, nil,
		WithStatus(http.StatusRequestEntityTooLarge),
		WithCode(CodeRequestEntityTooLarge),
		WithMessage(msg),
	)
}

func InternalError(w http.ResponseWriter, r *http.Request, msg string, details ...any) {
	opts := []Option{
		WithStatus(http.StatusInternalServerError),
		WithCode(CodeInternalError),
		WithMessage(msg),
	}
	if len(details) > 0 && details[0] != nil {
		opts = append(opts, WithDetails(details[0]))
	}
	HandleHTTP(w, r, nil, opts...)
}

func InternalWS(conn *wsrouter.Conn, message string) {
	HandleWS(conn, nil,
		WithStatus(http.StatusInternalServerError),
		WithCode(CodeInternalError),
		WithMessage(message),
	)
}

func BadEventWS(conn *wsrouter.Conn, message string) {
	HandleWS(conn, nil,
		WithStatus(http.StatusBadRequest),
		WithCode(CodeBadEvent),
		WithMessage(message),
	)
}

func NotFoundWS(conn *wsrouter.Conn, message string) {
	HandleWS(conn, nil,
		WithStatus(http.StatusNotFound),
		WithCode(CodeNotFound),
		WithMessage(message),
	)
}

func ConflictWS(conn *wsrouter.Conn, message string) {
	HandleWS(conn, nil,
		WithStatus(http.StatusConflict),
		WithCode(CodeConflict),
		WithMessage(message),
	)
}

func UnauthorizedWS(conn *wsrouter.Conn, message string) {
	HandleWS(conn, nil,
		WithStatus(http.StatusUnauthorized),
		WithCode(CodeUnauthorized),
		WithMessage(message),
	)
}

func ForbiddenWS(conn *wsrouter.Conn, message string) {
	HandleWS(conn, nil,
		WithStatus(http.StatusForbidden),
		WithCode(CodeForbidden),
		WithMessage(message),
	)
}

func JoinChatRequiredWS(conn *wsrouter.Conn) {
	HandleWS(conn, nil,
		WithStatus(http.StatusBadRequest),
		WithCode(CodeJoinChatRequired),
		WithMessage("no active chat found in connection"),
		WithDetails(map[string]any{
			"hint": "join a chat first",
		}),
	)
}
