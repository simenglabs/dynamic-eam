package app

import (
	"context"
	"testing"

	"eam/services/administration/internal/domain/user"
	"eam/services/common/errs"
)

func TestLoginSuccessIssuesToken(t *testing.T) {
	e := newEnv()
	res, err := e.auth.Login(context.Background(), "ADMIN", "rootpassword")
	if err != nil {
		t.Fatal(err)
	}
	if res.Token == "" || res.User.Username != "admin" || res.ExpiresIn <= 0 {
		t.Errorf("hasil login salah: %+v", res)
	}
}

func TestLoginUnknownUserIsBadCredentials(t *testing.T) {
	e := newEnv()
	_, err := e.auth.Login(context.Background(), "hantu", "x")
	if err != user.ErrBadCredentials {
		t.Errorf("err = %v", err)
	}
}

func TestLoginCountsDownThenLocks(t *testing.T) {
	e := newEnv()
	ctx := context.Background()
	for i := 1; i < user.MaxFailedLogins; i++ {
		res, err := e.auth.Login(ctx, "admin", "salah")
		if errs.KindOf(err) != errs.KindUnauthorized || res.AttemptsRemaining != user.MaxFailedLogins-i {
			t.Fatalf("percobaan %d: remaining=%d err=%v", i, res.AttemptsRemaining, err)
		}
	}
	if _, err := e.auth.Login(ctx, "admin", "salah"); err == nil || err == user.ErrBadCredentials {
		t.Fatalf("percobaan terakhir harus mengunci, err=%v", err)
	}
	if _, err := e.auth.Login(ctx, "admin", "rootpassword"); err != user.ErrLocked {
		t.Errorf("setelah terkunci password benar pun ditolak, err=%v", err)
	}
}

func TestLoginSuccessResetsFailures(t *testing.T) {
	e := newEnv()
	ctx := context.Background()
	e.auth.Login(ctx, "admin", "salah")
	e.auth.Login(ctx, "admin", "rootpassword")
	u, _ := e.users.FindByLogin(ctx, "admin")
	if u.FailedAttempts != 0 {
		t.Errorf("failed attempts = %d, want 0", u.FailedAttempts)
	}
}
