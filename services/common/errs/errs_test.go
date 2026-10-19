package errs

import (
	"errors"
	"fmt"
	"testing"
)

func TestKindOf(t *testing.T) {
	cases := []struct {
		err  error
		want Kind
	}{
		{Invalid("name is required"), KindInvalid},
		{NotFound("user"), KindNotFound},
		{Conflict("username is already taken"), KindConflict},
		{Forbidden("superadmin only"), KindForbidden},
		{Unauthorized("wrong password"), KindUnauthorized},
		{Unavailable("search index failed"), KindUnavailable},
		{fmt.Errorf("wrap: %w", Invalid("x")), KindInvalid},
		{errors.New("mongo down"), KindInternal},
		{nil, ""},
	}
	for _, c := range cases {
		if got := KindOf(c.err); got != c.want {
			t.Errorf("KindOf(%v) = %q, want %q", c.err, got, c.want)
		}
	}
}

func TestMessageIsUserFacing(t *testing.T) {
	if Invalid("email is invalid").Error() != "email is invalid" {
		t.Error("Invalid message must be returned as-is")
	}
	if NotFound("user").Error() != "user not found" {
		t.Errorf("got %q", NotFound("user").Error())
	}
}
