package fail

import (
	"errors"
	"fmt"
)

const (
	ExitOK        = 0
	ExitGeneric   = 1
	ExitUsage     = 2
	ExitAuth      = 3
	ExitPartial   = 4
	ExitCancelled = 5
	ExitNotImpl   = 6
)

type Error struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"http_status,omitempty"`
	Exit       int    `json:"-"`
}

func (e *Error) Error() string {
	if e.HTTPStatus != 0 {
		return fmt.Sprintf("%s: %s (http %d)", e.Code, e.Message, e.HTTPStatus)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func New(code, msg string, exit int) *Error {
	return &Error{Code: code, Message: msg, Exit: exit}
}

func Wrap(err error, code string, exit int) *Error {
	if err == nil {
		return nil
	}
	return &Error{Code: code, Message: err.Error(), Exit: exit}
}

func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	var fe *Error
	if errors.As(err, &fe) {
		if fe.Exit == 0 {
			return ExitGeneric
		}
		return fe.Exit
	}
	return ExitGeneric
}
