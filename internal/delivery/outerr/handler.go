package outerr

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/render"
	"github.com/karavanix/karavantrack-api-server/pkg/wsrouter"
)

type Response struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func HandleHTTP(w http.ResponseWriter, r *http.Request, err error, opts ...Option) {
	status := http.StatusInternalServerError
	resp := Response{
		Code:    CodeInternalError,
		Message: "Something went wrong!",
	}

	if err != nil {
		if m, ok := DefaultRegistry.FindMapping(err); ok {
			status = m.HTTPStatus
			if m.Message == "" {
				m.Message = err.Error()
			}
			resp = Response{
				Code:    m.Code,
				Message: m.Message,
				Details: m.Details,
			}
		}
	}

	for _, o := range opts {
		o(&resp, &status)
	}

	render.Status(r, status)
	render.JSON(w, r, resp)
}

func HandleWS(conn *wsrouter.Conn, err error, opts ...Option) {
	status := http.StatusInternalServerError
	resp := Response{
		Code:    CodeInternalError,
		Message: "Something went wrong!",
	}

	if err != nil {
		if m, ok := DefaultRegistry.FindMapping(err); ok {
			status = m.HTTPStatus
			if m.Message == "" {
				m.Message = err.Error()
			}
			resp = Response{
				Code:    m.Code,
				Message: m.Message,
				Details: m.Details,
			}
		}
	}

	for _, o := range opts {
		o(&resp, &status)
	}

	raw, _ := json.Marshal(resp)
	_ = conn.WriteJSON(wsrouter.Message{Event: wsrouter.ErrorEvent, Data: json.RawMessage(raw)})
}

// Is is a thin wrapper to keep call sites tidy (mirrors errors.Is).
func Is(target, err error) bool { return errors.Is(err, target) }

type Option func(*Response, *int)

func WithMessage(msg string) Option {
	return func(r *Response, _ *int) { r.Message = msg }
}

func WithCode(code string) Option {
	return func(r *Response, _ *int) { r.Code = code }
}

func WithDetail(key string, value any) Option {
	return func(r *Response, _ *int) {
		m, ok := r.Details.(map[string]any)
		if !ok || m == nil {
			m = map[string]any{}
		}
		m[key] = value
		r.Details = m
	}
}

func WithDetails(v any) Option {
	return func(r *Response, _ *int) { r.Details = v }
}

func WithStatus(status int) Option {
	return func(_ *Response, s *int) { *s = status }
}
