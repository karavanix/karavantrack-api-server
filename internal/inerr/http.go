package inerr

import (
	"encoding/json"
	"time"
)

type ErrHttp struct {
	StatusCode int
	Method     string
	Endpoint   string
	Duration   time.Duration
	Message    string
	Body       json.RawMessage
}

func (e ErrHttp) Error() string {
	return e.Message + ": " + string(e.Body)
}

func (e ErrHttp) Is(target error) bool {
	_, ok := target.(ErrHttp)
	if ok {
		return true
	}
	_, ok = target.(*ErrHttp)
	return ok
}

func NewErrHttp(statusCode int, method, endpoint, message string, body json.RawMessage) error {
	return &ErrHttp{
		StatusCode: statusCode,
		Message:    message,
		Body:       body,
	}
}
