package app

import (
	"context"

	"eam/services/administration/internal/domain/user"
	"eam/services/common/errs"
)

type Auth struct{ d Deps }

func NewAuth(d Deps) *Auth { return &Auth{d} }

type LoginResult struct {
	Token             string
	ExpiresIn         int // seconds
	User              *user.User
	AttemptsRemaining int // set when the password is wrong
}

// Dummy hash so response time is the same when the username doesn't exist (prevents user enumeration).
const dummyHash = "$2a$10$7EqJtq98hPqEX7fNZaFWoOhi5BWX4Z3Z8bQxQyZ3cZ6aZbYzL2J3W"

func (a *Auth) Login(ctx context.Context, login, password string) (LoginResult, error) {
	u, err := a.d.Users.FindByLogin(ctx, login)
	if errs.KindOf(err) == errs.KindNotFound {
		a.d.Hasher.Compare([]byte(dummyHash), password)
		return LoginResult{}, user.ErrBadCredentials
	}
	if err != nil {
		return LoginResult{}, err
	}
	if err := u.CheckLogin(a.d.Hasher, password); err != nil {
		if err != user.ErrBadCredentials {
			return LoginResult{}, err
		}
		attempts, locked, rerr := a.d.Users.RecordFailedLogin(ctx, u.ID)
		if rerr != nil {
			return LoginResult{}, rerr
		}
		if locked {
			return LoginResult{}, errs.Unauthorized("account locked due to too many failed attempts")
		}
		return LoginResult{AttemptsRemaining: user.MaxFailedLogins - attempts}, err
	}
	if u.FailedAttempts > 0 {
		if err := a.d.Users.ResetFailedLogins(ctx, u.ID); err != nil {
			return LoginResult{}, err
		}
		u.FailedAttempts = 0
	}
	tok, ttl, err := a.d.Tokens.Issue(u)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Token: tok, ExpiresIn: int(ttl.Seconds()), User: u, AttemptsRemaining: user.MaxFailedLogins}, nil
}
