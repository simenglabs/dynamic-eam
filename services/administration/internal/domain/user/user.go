// Package user: User aggregate (login account) with validation & login rules.
package user

import (
	"net/mail"
	"slices"
	"strings"
	"time"

	"eam/services/common/errs"
)

const (
	MaxFailedLogins = 5
	SuperAdminRole  = "superadmin"
	minPassword     = 8
	maxPassword     = 72 // bcrypt limit
)

var (
	ErrBadCredentials = errs.Unauthorized("wrong username or password")
	ErrLocked         = errs.Unauthorized("account is locked, contact the administrator")
	ErrInactive       = errs.Unauthorized("account is inactive")
)

// Hasher: password hashing port (bcrypt implementation in infra/crypto).
type Hasher interface {
	Hash(password string) ([]byte, error)
	Compare(hash []byte, password string) bool
}

type User struct {
	ID             string
	Username       string
	DisplayName    string
	Email          string
	Roles          []string
	Active         bool
	Locked         bool
	FailedAttempts int
	PasswordHash   []byte
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type NewParams struct {
	Username, DisplayName, Email, Password string
	Roles                                  []string
	Inactive                               bool
}

// EditParams: empty Password = unchanged; nil Active/Locked = unchanged.
type EditParams struct {
	DisplayName, Email, Password string
	Roles                        []string
	Active, Locked               *bool
}

func New(id string, p NewParams, h Hasher, now time.Time) (*User, error) {
	u := &User{ID: id, Username: strings.TrimSpace(p.Username), Active: !p.Inactive, CreatedAt: now}
	if u.Username == "" {
		return nil, errs.Invalid("username is required")
	}
	if err := u.apply(p.DisplayName, p.Email, p.Roles, now); err != nil {
		return nil, err
	}
	if err := u.setPassword(h, p.Password); err != nil {
		return nil, err
	}
	return u, nil
}

func (u *User) Edit(p EditParams, h Hasher, now time.Time) error {
	if err := u.apply(p.DisplayName, p.Email, p.Roles, now); err != nil {
		return err
	}
	if p.Active != nil {
		u.Active = *p.Active
	}
	if p.Locked != nil {
		u.Locked = *p.Locked
		if !u.Locked {
			u.FailedAttempts = 0
		}
	}
	if p.Password != "" {
		return u.setPassword(h, p.Password)
	}
	return nil
}

func (u *User) apply(displayName, email string, roles []string, now time.Time) error {
	displayName, email = strings.TrimSpace(displayName), strings.TrimSpace(email)
	if displayName == "" {
		return errs.Invalid("name is required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return errs.Invalid("email is invalid")
	}
	if len(roles) == 0 {
		return errs.Invalid("at least one role is required")
	}
	u.DisplayName, u.Email, u.Roles, u.UpdatedAt = displayName, email, slices.Compact(slices.Sorted(slices.Values(roles))), now
	return nil
}

func (u *User) setPassword(h Hasher, pw string) error {
	if len(pw) < minPassword {
		return errs.Invalid("password must be at least %d characters", minPassword)
	}
	if len(pw) > maxPassword {
		return errs.Invalid("password must be at most %d characters", maxPassword)
	}
	hash, err := h.Hash(pw)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	return nil
}

// CheckLogin: check order = locked, inactive, then password.
func (u *User) CheckLogin(h Hasher, pw string) error {
	switch {
	case u.Locked:
		return ErrLocked
	case !u.Active:
		return ErrInactive
	case !h.Compare(u.PasswordHash, pw):
		return ErrBadCredentials
	}
	return nil
}

func ShouldLock(failedAttempts int) bool { return failedAttempts >= MaxFailedLogins }

func (u *User) UsernameKey() string { return strings.ToLower(u.Username) }
func (u *User) EmailKey() string    { return strings.ToLower(u.Email) }
func (u *User) IsSuperAdmin() bool  { return slices.Contains(u.Roles, SuperAdminRole) }
