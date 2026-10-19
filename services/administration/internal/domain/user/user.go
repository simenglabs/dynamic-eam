// Package user: aggregate User (akun login) beserta aturan validasi & login.
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
	maxPassword     = 72 // batas bcrypt
)

var (
	ErrBadCredentials = errs.Unauthorized("username atau password salah")
	ErrLocked         = errs.Unauthorized("akun terkunci, hubungi administrator")
	ErrInactive       = errs.Unauthorized("akun tidak aktif")
)

// Hasher: port hashing password (implementasi bcrypt di infra/crypto).
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

// EditParams: Password kosong = tidak diubah; Active/Locked nil = tidak diubah.
type EditParams struct {
	DisplayName, Email, Password string
	Roles                        []string
	Active, Locked               *bool
}

func New(id string, p NewParams, h Hasher, now time.Time) (*User, error) {
	u := &User{ID: id, Username: strings.TrimSpace(p.Username), Active: !p.Inactive, CreatedAt: now}
	if u.Username == "" {
		return nil, errs.Invalid("username wajib diisi")
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
		return errs.Invalid("nama wajib diisi")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return errs.Invalid("email tidak valid")
	}
	if len(roles) == 0 {
		return errs.Invalid("minimal satu role")
	}
	u.DisplayName, u.Email, u.Roles, u.UpdatedAt = displayName, email, slices.Compact(slices.Sorted(slices.Values(roles))), now
	return nil
}

func (u *User) setPassword(h Hasher, pw string) error {
	if len(pw) < minPassword {
		return errs.Invalid("password minimal %d karakter", minPassword)
	}
	if len(pw) > maxPassword {
		return errs.Invalid("password maksimal %d karakter", maxPassword)
	}
	hash, err := h.Hash(pw)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	return nil
}

// CheckLogin: urutan cek = terkunci, nonaktif, lalu password.
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
