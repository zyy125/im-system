package apperr

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Error struct {
	Code    Code
	Message string
	Cause   error
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

func Wrap(code Code, message string, cause error) *Error {
	return &Error{Code: code, Message: message, Cause: cause}
}

func From(err error) *Error {
	if err == nil {
		return nil
	}

	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return NotFound(CodeNotFound, "resource not found")
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return Conflict(CodeConflict, "resource already exists")
	case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
		return InvalidCredentials()
	case errors.Is(err, context.DeadlineExceeded):
		return New(CodeTimeout, "request timed out")
	case errors.Is(err, context.Canceled):
		return New(CodeServiceUnavailable, "request canceled")
	}

	return Internal("internal server error", err)
}

func CodeOf(err error) Code {
	if err == nil {
		return CodeOK
	}
	return From(err).Code
}

func IsCode(err error, code Code) bool {
	if err == nil {
		return false
	}
	return CodeOf(err) == code
}
