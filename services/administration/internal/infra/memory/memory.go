// Package memory: in-memory adapter for user.Repository & role.Repository.
// Used by tests (app, http) and usable for dev without Mongo. Same contract as infra/mongo.
package memory

import (
	"context"
	"slices"
	"strings"
	"sync"

	"eam/services/administration/internal/domain/role"
	"eam/services/administration/internal/domain/user"
	"eam/services/common/errs"
)

type Users struct {
	mu sync.Mutex
	m  map[string]user.User
}

func NewUsers() *Users { return &Users{m: map[string]user.User{}} }

func (r *Users) Get(_ context.Context, id string) (*user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.m[id]
	if !ok {
		return nil, errs.NotFound("user")
	}
	return &u, nil
}

func (r *Users) FindByLogin(_ context.Context, login string) (*user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	l := strings.ToLower(strings.TrimSpace(login))
	for _, u := range r.m {
		if u.UsernameKey() == l || u.EmailKey() == l {
			return &u, nil
		}
	}
	return nil, errs.NotFound("user")
}

func (r *Users) List(context.Context) ([]*user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []*user.User{}
	for _, u := range r.m {
		out = append(out, &u)
	}
	slices.SortFunc(out, func(a, b *user.User) int { return strings.Compare(a.UsernameKey(), b.UsernameKey()) })
	return out, nil
}

func (r *Users) dup(u *user.User) bool {
	for id, o := range r.m {
		if id != u.ID && (o.UsernameKey() == u.UsernameKey() || o.EmailKey() == u.EmailKey()) {
			return true
		}
	}
	return false
}

func (r *Users) Insert(_ context.Context, u *user.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.dup(u) {
		return errs.Conflict("username or email is already taken")
	}
	r.m[u.ID] = *u
	return nil
}

func (r *Users) Update(_ context.Context, u *user.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.dup(u) {
		return errs.Conflict("email is already taken")
	}
	r.m[u.ID] = *u
	return nil
}

func (r *Users) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.m, id)
	return nil
}

func (r *Users) RecordFailedLogin(_ context.Context, id string) (int, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u := r.m[id]
	u.FailedAttempts++
	u.Locked = u.Locked || user.ShouldLock(u.FailedAttempts)
	r.m[id] = u
	return u.FailedAttempts, u.Locked, nil
}

func (r *Users) ResetFailedLogins(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	u := r.m[id]
	u.FailedAttempts = 0
	r.m[id] = u
	return nil
}

func (r *Users) CountWithRole(_ context.Context, roleID string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, u := range r.m {
		if slices.Contains(u.Roles, roleID) {
			n++
		}
	}
	return n, nil
}

type Roles struct {
	mu sync.Mutex
	m  map[string]role.Role
}

func NewRoles() *Roles { return &Roles{m: map[string]role.Role{}} }

func (r *Roles) List(context.Context) ([]*role.Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []*role.Role{}
	for _, x := range r.m {
		out = append(out, &x)
	}
	return out, nil
}

func (r *Roles) Get(_ context.Context, id string) (*role.Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	x, ok := r.m[id]
	if !ok {
		return nil, errs.NotFound("role")
	}
	return &x, nil
}

func (r *Roles) Insert(_ context.Context, x *role.Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, o := range r.m {
		if o.ID == x.ID || o.Name == x.Name {
			return errs.Conflict("role ID or name is already taken")
		}
	}
	r.m[x.ID] = *x
	return nil
}

func (r *Roles) Update(_ context.Context, x *role.Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[x.ID] = *x
	return nil
}

func (r *Roles) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.m, id)
	return nil
}

func (r *Roles) CountExisting(_ context.Context, ids []string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, id := range ids {
		if _, ok := r.m[id]; ok {
			n++
		}
	}
	return n, nil
}

func (r *Roles) InsertIfMissing(_ context.Context, x *role.Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.m[x.ID]; !ok {
		r.m[x.ID] = *x
	}
	return nil
}
