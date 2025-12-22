package app

import "fmt"

type httpError struct {
	code        int
	message     string
	internalErr error
}

func (e *httpError) Error() string {
	if e.internalErr != nil {
		return fmt.Sprintf("%s: %v", e.message, e.internalErr)
	}
	return e.message
}

func (e *httpError) Unwrap() error {
	return e.internalErr
}

func newHTTPError(code int, message string) *httpError {
	return &httpError{
		code:    code,
		message: message,
	}
}

func wrapHTTPError(code int, message string, err error) *httpError {
	return &httpError{
		code:        code,
		message:     message,
		internalErr: err,
	}
}
