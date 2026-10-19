// Package errs: domain errors with a Kind that the http layer maps to an HTTP status.
// Invalid/Conflict/Forbidden/Unauthorized messages are safe to show to users; other errors are internal.
package errs

import (
	"errors"
	"fmt"
)

type Kind string

const (
	KindInvalid      Kind = "invalid"
	KindNotFound     Kind = "not_found"
	KindConflict     Kind = "conflict"
	KindForbidden    Kind = "forbidden"
	KindUnauthorized Kind = "unauthorized"
	KindUnavailable  Kind = "unavailable" // external dependency (e.g. Elasticsearch) failed; message is safe to show
	KindInternal     Kind = "internal"
)

type Error struct {
	Kind Kind
	Msg  string
}

func (e *Error) Error() string { return e.Msg }

func Invalid(format string, a ...any) error  { return &Error{KindInvalid, fmt.Sprintf(format, a...)} }
func Conflict(format string, a ...any) error { return &Error{KindConflict, fmt.Sprintf(format, a...)} }
func Forbidden(format string, a ...any) error {
	return &Error{KindForbidden, fmt.Sprintf(format, a...)}
}
func Unauthorized(format string, a ...any) error {
	return &Error{KindUnauthorized, fmt.Sprintf(format, a...)}
}
func Unavailable(format string, a ...any) error {
	return &Error{KindUnavailable, fmt.Sprintf(format, a...)}
}
func NotFound(what string) error { return &Error{KindNotFound, what + " not found"} }

// KindOf: the error kind (sees through wrapping). nil -> "", unknown error -> KindInternal.
func KindOf(err error) Kind {
	if err == nil {
		return ""
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return KindInternal
}
