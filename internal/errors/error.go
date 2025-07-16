package errors

import (
	"fmt"
	"net/http"

	"github.com/go-chi/render"
)

type CustomError struct {
	OriginalErr error  `json:"-"`
	Message     string `json:"message,omitempty"`
	StatusCode  int    `json:"status,omitempty"`
}

// Error returns the error message
func (e *CustomError) Error() string {
	if e.OriginalErr != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.OriginalErr)
	}
	return e.Message
}

// GetStatusCode returns the status code
func (e *CustomError) GetStatusCode() int {
	return e.StatusCode
}

func (e *CustomError) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.GetStatusCode())
	return nil
}

// WrapError wraps an original error with a predefined CustomError
func Wrap(predefError *CustomError, originalErr error) *CustomError {
	return &CustomError{
		OriginalErr: originalErr,
		Message:     predefError.Message,
		StatusCode:  predefError.StatusCode,
	}
}

func Render(w http.ResponseWriter, r *http.Request, err error) {
	re, ok := err.(*CustomError)
	if !ok {
		render.Status(r, http.StatusInternalServerError)
		render.Render(w, r, re)
	}
	render.Status(r, re.GetStatusCode())
	render.Render(w, r, re)
}

var (
	ErrInvalidRequest = &CustomError{
		Message:    "Invalid request body",
		StatusCode: 400,
	}

	ErrInvalidToken = &CustomError{
		Message:    "Invalid token",
		StatusCode: 401,
	}

	ErrFailedToGoToURL = &CustomError{
		Message:    "Failed to go to URL",
		StatusCode: 500,
	}

	ErrFailedToEncodeResponse = &CustomError{
		Message:    "Failed to encode response",
		StatusCode: 500,
	}

	ErrFailedToCreateHTTPClient = &CustomError{
		Message:    "Failed to create HTTP client",
		StatusCode: 500,
	}
)
