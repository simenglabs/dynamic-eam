package app

import (
	"context"

	"eam/services/administration/internal/domain/role"
)

type Roles struct{ d Deps }

func NewRoles(d Deps) *Roles { return &Roles{d} }

type RoleInput struct {
	ID, Name, Description string
	Permissions           role.Permissions
}

func (s *Roles) List(ctx context.Context) ([]*role.Role, error) { return s.d.Roles.List(ctx) }

func (s *Roles) Get(ctx context.Context, id string) (*role.Role, error) {
	return s.d.Roles.Get(ctx, id)
}

func (s *Roles) Create(ctx context.Context, in RoleInput) (*role.Role, error) {
	r, err := role.New(in.ID, in.Name, in.Description, in.Permissions, s.d.Now())
	if err != nil {
		return nil, err
	}
	return r, s.d.Roles.Insert(ctx, r)
}

func (s *Roles) Update(ctx context.Context, id string, in RoleInput) (*role.Role, error) {
	r, err := s.d.Roles.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := r.Edit(in.Name, in.Description, in.Permissions, s.d.Now()); err != nil {
		return nil, err
	}
	return r, s.d.Roles.Update(ctx, r)
}

func (s *Roles) Delete(ctx context.Context, id string) error {
	r, err := s.d.Roles.Get(ctx, id)
	if err != nil {
		return err
	}
	n, err := s.d.Users.CountWithRole(ctx, id)
	if err != nil {
		return err
	}
	if err := r.CanDelete(n); err != nil {
		return err
	}
	return s.d.Roles.Delete(ctx, id)
}
