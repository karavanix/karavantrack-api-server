package outerr

import (
	"errors"
	"net/http"
	"sync"

	"github.com/go-playground/validator/v10"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
)

type Mapping struct {
	HTTPStatus int
	Code       string
	Message    string
	Details    map[string]any
}

type regItem struct {
	match   func(error) bool
	builder func(error) Mapping
}

type Registry struct {
	mu   sync.RWMutex
	maps []regItem
}

func NewRegistry() *Registry {
	r := &Registry{}

	r.RegisterMatch(
		func(err error) bool {
			var ve validator.ValidationErrors
			return errors.As(err, &ve)
		},
		func(err error) Mapping {
			var ve validator.ValidationErrors
			if !errors.As(err, &ve) {
				return Mapping{
					HTTPStatus: http.StatusBadRequest,
					Code:       CodeBadRequest,
				}
			}

			// Формируем details для фронта/клиента
			violations := make([]map[string]any, 0, len(ve))
			for _, fe := range ve {
				violations = append(violations, map[string]any{
					"field": fe.Field(),         // имя поля в структуре
					"tag":   fe.Tag(),           // required, email, min, etc
					"param": fe.Param(),         // параметр тега (например 10 для min=10)
					"value": fe.Value(),         // фактическое значение
					"kind":  fe.Kind().String(), // string/int/...
					"type":  fe.Type().String(), // тип Go
				})
			}

			return Mapping{
				HTTPStatus: http.StatusBadRequest,
				Code:       CodeBadRequest, // или сделай отдельный CodeValidationFailed
				Message:    "validation failed",
				Details: map[string]any{
					"violations": violations,
				},
			}
		},
	)

	r.RegisterMatch(
		func(err error) bool {
			var ive *validator.InvalidValidationError
			return errors.As(err, &ive)
		},
		func(err error) Mapping {
			return Mapping{
				HTTPStatus: http.StatusInternalServerError,
				Code:       CodeInternalError,
				Message:    "invalid validation configuration",
				Details: map[string]any{
					"error": err.Error(),
				},
			}
		},
	)

	// Register mappings for your inerr package (extend as needed).
	r.RegisterMatch(func(err error) bool { return errors.Is(err, inerr.ErrNotFound{}) },
		func(err error) Mapping {
			return Mapping{
				HTTPStatus: http.StatusNotFound,
				Code:       CodeNotFound,
			}
		},
	)

	r.RegisterMatch(func(err error) bool { return errors.Is(err, inerr.ErrConflict{}) },
		func(err error) Mapping {
			return Mapping{
				HTTPStatus: http.StatusConflict,
				Code:       CodeConflict,
			}
		},
	)

	r.RegisterMatch(func(err error) bool { return errors.Is(err, inerr.ErrNoChanges{}) },
		func(err error) Mapping {
			return Mapping{
				HTTPStatus: http.StatusBadRequest,
				Code:       CodeNoChanges,
			}
		},
	)

	r.RegisterMatch(func(err error) bool { return errors.Is(err, inerr.ErrorExpiredToken) },
		func(err error) Mapping {
			return Mapping{
				HTTPStatus: http.StatusUnauthorized,
				Code:       CodeUnauthorized,
				Message:    "token expired",
				Details: map[string]any{
					"error": err.Error(),
				},
			}
		},
	)

	r.RegisterMatch(func(err error) bool { return errors.Is(err, inerr.ErrorWrongAlgo) },
		func(err error) Mapping {
			return Mapping{
				HTTPStatus: http.StatusUnauthorized,
				Code:       CodeUnauthorized,
				Message:    "token expired",
				Details: map[string]any{
					"error": err.Error(),
				},
			}
		},
	)

	r.RegisterMatch(func(err error) bool { return errors.Is(err, inerr.ErrInvalidToken{}) },
		func(err error) Mapping {
			return Mapping{
				HTTPStatus: http.StatusUnauthorized,
				Code:       CodeUnauthorized,
			}
		},
	)

	r.RegisterMatch(func(err error) bool { return errors.Is(err, inerr.ErrValidation{}) },
		func(err error) Mapping {
			return Mapping{
				HTTPStatus: http.StatusBadRequest,
				Code:       CodeBadRequest,
			}
		},
	)

	r.RegisterMatch(func(err error) bool { return errors.Is(err, inerr.ErrorPermissionDenied) },
		func(err error) Mapping {
			return Mapping{
				HTTPStatus: http.StatusForbidden,
				Code:       CodeForbidden,
			}
		},
	)

	r.RegisterMatch(
		func(err error) bool {
			return errors.Is(err, inerr.ErrHttp{})
		},
		func(err error) Mapping {
			httpErr, ok := err.(*inerr.ErrHttp)
			if !ok {
				return Mapping{
					HTTPStatus: http.StatusInternalServerError,
					Code:       CodeInternalError,
				}
			}

			return Mapping{
				HTTPStatus: httpErr.StatusCode,
				Code:       CodeExternalService,
				Message:    httpErr.Message,
				Details: map[string]any{
					"body": httpErr.Body,
				},
			}
		},
	)

	return r
}

// RegisterMatch registers how to map an error to an API response.
func (r *Registry) RegisterMatch(match func(error) bool, builder func(error) Mapping) {
	r.mu.Lock()
	r.maps = append(r.maps, regItem{match: match, builder: builder})
	r.mu.Unlock()
}

// FindMapping looks up a mapping on THIS registry.
func (r *Registry) FindMapping(err error) (Mapping, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, it := range r.maps {
		if it.match(err) {
			return it.builder(err), true
		}
	}
	return Mapping{}, false
}

// Singleton
var DefaultRegistry = NewRegistry()
