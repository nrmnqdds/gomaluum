package errors

import (
	"errors"
	"fmt"
)

type ErrorDetails struct {
	ApplicationError error  `json:"application_error"`
	ErrorMessage     string `json:"error"`
	ErrorDetail      string `json:"message"`
	HTTPCode         int    `json:"http_code"`
}

func (e ErrorDetails) Error() string {
	errStr := fmt.Sprintf("Error: %s, Detail: %s", e.ErrorMessage, e.ErrorDetail)
	return errStr
}

var (
	ErrInvalidRequest  = errors.New("invalid request body")
	ErrInvalidToken    = errors.New("invalid token")
	ErrFailedToGoToURL = errors.New("failed to go to URL")
)
