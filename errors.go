package scambus

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrAuthentication = errors.New("scambus: authentication failed")
	ErrValidation     = errors.New("scambus: request validation failed")
	ErrNotFound       = errors.New("scambus: resource not found")
	ErrServer         = errors.New("scambus: server error")
	ErrRateLimited    = errors.New("scambus: rate limited")
	ErrCursorExpired  = errors.New("scambus: cursor outside retention window")
	ErrNoCredentials  = errors.New("scambus: no authentication provided")
)

type APIError struct {
	StatusCode int
	Message    string
	Method     string
	Endpoint   string
	Body       []byte
	Data       map[string]any

	kind error
}

func (e *APIError) Error() string {
	if e.Method != "" {
		return fmt.Sprintf("scambus: %s %s: %d %s", e.Method, e.Endpoint, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("scambus: %d %s", e.StatusCode, e.Message)
}

func (e *APIError) Unwrap() error { return e.kind }

func kindForStatus(status int) error {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return ErrAuthentication
	case status == http.StatusBadRequest:
		return ErrValidation
	case status == http.StatusNotFound:
		return ErrNotFound
	case status == http.StatusTooManyRequests:
		return ErrRateLimited
	case status == http.StatusGone || status == http.StatusRequestedRangeNotSatisfiable:
		return ErrCursorExpired
	case status >= 500:
		return ErrServer
	default:
		return nil
	}
}
