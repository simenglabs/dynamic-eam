package user

import (
	"errors"
	"strings"
	"testing"
	"time"

	"eam/services/common/errs"
)

// fakeHasher: hash = "h:" + password. Fast & deterministic for domain tests.
type fakeHasher struct{}

func (fakeHasher) Hash(pw string) ([]byte, error)   { return []byte("h:" + pw), nil }
func (fakeHasher) Compare(h []byte, pw string) bool { return string(h) == "h:"+pw }

var now = time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)

func valid() NewParams {
	return NewParams{Username: " Budi ", DisplayName: "Budi S", Email: "Budi@X.id", Roles: []string{"user"}, Password: "password1"}
}

func TestNewNormalizesAndHashes(t *testing.T) {
	u, err := New("id1", valid(), fakeHasher{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if u.Username != "Budi" || u.UsernameKey() != "budi" || u.EmailKey() != "budi@x.id" {
		t.Errorf("wrong normalization: %+v", u)
	}
	if !u.Active || u.Locked || string(u.PasswordHash) != "h:password1" || !u.CreatedAt.Equal(now) {
		t.Errorf("wrong initial state: %+v", u)
	}
}

func TestNewRejectsInvalidInput(t *testing.T) {
	cases := map[string]func(*NewParams){
		"empty username":     func(p *NewParams) { p.Username = "  " },
		"empty name":         func(p *NewParams) { p.DisplayName = "" },
		"bad email":          func(p *NewParams) { p.Email = "not-an-email" },
		"no role":            func(p *NewParams) { p.Roles = nil },
		"password too short": func(p *NewParams) { p.Password = "short" },
		"password too long":  func(p *NewParams) { p.Password = strings.Repeat("x", 73) },
	}
	for name, mutate := range cases {
		p := valid()
		mutate(&p)
		if _, err := New("id", p, fakeHasher{}, now); errs.KindOf(err) != errs.KindInvalid {
			t.Errorf("%s: err = %v, want invalid", name, err)
		}
	}
}

func TestInactiveFlagRespected(t *testing.T) {
	p := valid()
	p.Inactive = true
	u, _ := New("id", p, fakeHasher{}, now)
	if u.Active {
		t.Error("Inactive=true must make the user inactive")
	}
}

func TestCheckLogin(t *testing.T) {
	u, _ := New("id", valid(), fakeHasher{}, now)
	if err := u.CheckLogin(fakeHasher{}, "password1"); err != nil {
		t.Fatal("correct password must pass:", err)
	}
	if err := u.CheckLogin(fakeHasher{}, "wrong"); !errors.Is(err, ErrBadCredentials) {
		t.Errorf("wrong password: %v", err)
	}
	u.Locked = true
	if err := u.CheckLogin(fakeHasher{}, "password1"); !errors.Is(err, ErrLocked) {
		t.Errorf("locked: %v", err)
	}
	u.Locked, u.Active = false, false
	if err := u.CheckLogin(fakeHasher{}, "password1"); !errors.Is(err, ErrInactive) {
		t.Errorf("inactive: %v", err)
	}
}

func TestShouldLock(t *testing.T) {
	if ShouldLock(MaxFailedLogins-1) || !ShouldLock(MaxFailedLogins) {
		t.Errorf("lock threshold must be exactly %d", MaxFailedLogins)
	}
}

func TestEdit(t *testing.T) {
	u, _ := New("id", valid(), fakeHasher{}, now)
	u.Locked, u.FailedAttempts = true, 5
	later := now.Add(time.Hour)
	err := u.Edit(EditParams{DisplayName: "Budi New", Email: "new@x.id", Roles: []string{"admin"}, Locked: new(false), Password: "newpassword"}, fakeHasher{}, later)
	if err != nil {
		t.Fatal(err)
	}
	if u.DisplayName != "Budi New" || u.EmailKey() != "new@x.id" || u.Roles[0] != "admin" || !u.UpdatedAt.Equal(later) {
		t.Errorf("edit not applied: %+v", u)
	}
	if u.Locked || u.FailedAttempts != 0 {
		t.Error("unlock must reset failed attempts")
	}
	if string(u.PasswordHash) != "h:newpassword" {
		t.Error("new password must be hashed")
	}
	if err := u.Edit(EditParams{DisplayName: "x", Email: "x@x.id", Roles: []string{"admin"}}, fakeHasher{}, later); err != nil || string(u.PasswordHash) != "h:newpassword" {
		t.Error("empty password on edit = unchanged")
	}
	if err := u.Edit(EditParams{DisplayName: "", Email: "x@x.id", Roles: []string{"a"}}, fakeHasher{}, later); errs.KindOf(err) != errs.KindInvalid {
		t.Error("edit is still validated")
	}
}

func TestIsSuperAdmin(t *testing.T) {
	p := valid()
	p.Roles = []string{"user", SuperAdminRole}
	u, _ := New("id", p, fakeHasher{}, now)
	if !u.IsSuperAdmin() {
		t.Error("must be superadmin")
	}
}
