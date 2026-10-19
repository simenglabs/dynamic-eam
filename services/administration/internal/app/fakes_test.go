package app

import (
	"context"
	"time"

	"eam/services/administration/internal/domain/user"
	"eam/services/administration/internal/infra/memory"
)

type fakeHasher struct{}

func (fakeHasher) Hash(pw string) ([]byte, error)   { return []byte("h:" + pw), nil }
func (fakeHasher) Compare(h []byte, pw string) bool { return string(h) == "h:"+pw }

type fakeTokens struct{}

func (fakeTokens) Issue(u *user.User) (string, time.Duration, error) {
	return "tok-" + u.ID, time.Hour, nil
}

// testEnv: all use cases with fake dependencies + built-in roles + superadmin "root".
type testEnv struct {
	users *memory.Users
	roles *memory.Roles
	auth  *Auth
	user  *Users
	role  *Roles
	seq   int
}

var t0 = time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)

func newEnv() *testEnv {
	e := &testEnv{users: memory.NewUsers(), roles: memory.NewRoles()}
	d := Deps{
		Users: e.users, Roles: e.roles, Hasher: fakeHasher{}, Tokens: fakeTokens{},
		Now:   func() time.Time { return t0 },
		NewID: func() string { e.seq++; return "id" + string(rune('0'+e.seq)) },
	}
	e.auth, e.user, e.role = NewAuth(d), NewUsers(d), NewRoles(d)
	if _, err := Seed(context.Background(), d, "rootpassword"); err != nil {
		panic(err)
	}
	return e
}

var (
	superActor = Actor{UserID: "root-actor", SuperAdmin: true}
	adminActor = Actor{UserID: "admin-actor"}
)
