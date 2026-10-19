package app

import (
	"context"
	"slices"

	"eam/services/administration/internal/domain/user"
	"eam/services/common/errs"
)

type Users struct{ d Deps }

func NewUsers(d Deps) *Users { return &Users{d} }

func (s *Users) List(ctx context.Context) ([]*user.User, error) { return s.d.Users.List(ctx) }

func (s *Users) Get(ctx context.Context, id string) (*user.User, error) {
	return s.d.Users.Get(ctx, id)
}

func (s *Users) Create(ctx context.Context, actor Actor, p user.NewParams) (*user.User, error) {
	if err := s.checkRoles(ctx, actor, p.Roles); err != nil {
		return nil, err
	}
	u, err := user.New(s.d.NewID(), p, s.d.Hasher, s.d.Now())
	if err != nil {
		return nil, err
	}
	return u, s.d.Users.Insert(ctx, u)
}

func (s *Users) Update(ctx context.Context, actor Actor, id string, p user.EditParams) (*user.User, error) {
	u, err := s.d.Users.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if u.IsSuperAdmin() && !actor.SuperAdmin {
		return nil, errs.Forbidden("only a superadmin can edit a superadmin account")
	}
	if err := s.checkRoles(ctx, actor, p.Roles); err != nil {
		return nil, err
	}
	if id == actor.UserID {
		sameRoles := slices.Equal(slices.Sorted(slices.Values(p.Roles)), u.Roles)
		if !sameRoles || (p.Active != nil && !*p.Active) || (p.Locked != nil && *p.Locked) {
			return nil, errs.Forbidden("you cannot change roles of, deactivate, or lock your own account")
		}
	}
	if err := u.Edit(p, s.d.Hasher, s.d.Now()); err != nil {
		return nil, err
	}
	return u, s.d.Users.Update(ctx, u)
}

func (s *Users) Delete(ctx context.Context, actor Actor, id string) error {
	if id == actor.UserID {
		return errs.Forbidden("you cannot delete your own account")
	}
	u, err := s.d.Users.Get(ctx, id)
	if err != nil {
		return err
	}
	if u.IsSuperAdmin() && !actor.SuperAdmin {
		return errs.Forbidden("only a superadmin can delete a superadmin account")
	}
	return s.d.Users.Delete(ctx, id)
}

// checkRoles: all roles must exist; only a superadmin can grant the superadmin role.
func (s *Users) checkRoles(ctx context.Context, actor Actor, roles []string) error {
	if slices.Contains(roles, user.SuperAdminRole) && !actor.SuperAdmin {
		return errs.Forbidden("only a superadmin can grant the superadmin role")
	}
	unique := slices.Compact(slices.Sorted(slices.Values(roles)))
	n, err := s.d.Roles.CountExisting(ctx, unique)
	if err != nil {
		return err
	}
	if n != len(unique) {
		return errs.Invalid("one or more roles are unknown")
	}
	return nil
}
