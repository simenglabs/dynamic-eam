package app

import (
	"context"
	"testing"

	"eam/services/administration/internal/domain/user"
	"eam/services/common/errs"
)

func newUser(roles ...string) user.NewParams {
	return user.NewParams{Username: "budi", DisplayName: "Budi", Email: "budi@x.id", Password: "password1", Roles: roles}
}

func TestCreateUser(t *testing.T) {
	e := newEnv()
	u, err := e.user.Create(context.Background(), adminActor, newUser("user", "admin"))
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := e.users.Get(context.Background(), u.ID); got == nil || len(got.Roles) != 2 {
		t.Errorf("user not stored correctly: %+v", got)
	}
}

func TestCreateUserRules(t *testing.T) {
	cases := map[string]struct {
		actor Actor
		p     user.NewParams
		want  errs.Kind
	}{
		"unknown role":                    {adminActor, newUser("ghost"), errs.KindInvalid},
		"admin grants superadmin":         {adminActor, newUser(user.SuperAdminRole), errs.KindForbidden},
		"superadmin may grant superadmin": {superActor, newUser(user.SuperAdminRole), ""},
	}
	for name, c := range cases {
		e := newEnv()
		_, err := e.user.Create(context.Background(), c.actor, c.p)
		if errs.KindOf(err) != c.want {
			t.Errorf("%s: err = %v, want %q", name, err, c.want)
		}
	}
}

func TestCreateDuplicateUsername(t *testing.T) {
	e := newEnv()
	ctx := context.Background()
	e.user.Create(ctx, adminActor, newUser("user"))
	p := newUser("user")
	p.Username, p.Email = "BUDI", "other@x.id"
	if _, err := e.user.Create(ctx, adminActor, p); errs.KindOf(err) != errs.KindConflict {
		t.Errorf("err = %v", err)
	}
}

func TestUpdateUserRules(t *testing.T) {
	ctx := context.Background()
	e := newEnv()
	budi, _ := e.user.Create(ctx, adminActor, newUser("user"))
	root, _ := e.users.FindByLogin(ctx, "admin")
	self := Actor{UserID: budi.ID}
	edit := func(roles ...string) user.EditParams {
		return user.EditParams{DisplayName: "Budi", Email: "budi@x.id", Roles: roles}
	}

	if _, err := e.user.Update(ctx, adminActor, root.ID, edit(user.SuperAdminRole)); errs.KindOf(err) != errs.KindForbidden {
		t.Errorf("admin edits superadmin: %v", err)
	}
	if _, err := e.user.Update(ctx, self, budi.ID, edit("admin")); errs.KindOf(err) != errs.KindForbidden {
		t.Errorf("change own role: %v", err)
	}
	locked := edit("user")
	locked.Locked = new(true)
	if _, err := e.user.Update(ctx, self, budi.ID, locked); errs.KindOf(err) != errs.KindForbidden {
		t.Errorf("lock own account: %v", err)
	}
	if _, err := e.user.Update(ctx, self, budi.ID, edit("user")); err != nil {
		t.Errorf("editing own profile without changing roles must be allowed: %v", err)
	}
	if u, err := e.user.Update(ctx, adminActor, budi.ID, edit("user", "admin")); err != nil || len(u.Roles) != 2 {
		t.Errorf("admin changes another user's roles: %v %+v", err, u)
	}
	if _, err := e.user.Update(ctx, adminActor, "missing", edit("user")); errs.KindOf(err) != errs.KindNotFound {
		t.Errorf("user does not exist: %v", err)
	}
}

func TestDeleteUserRules(t *testing.T) {
	ctx := context.Background()
	e := newEnv()
	budi, _ := e.user.Create(ctx, adminActor, newUser("user"))
	root, _ := e.users.FindByLogin(ctx, "admin")

	if err := e.user.Delete(ctx, Actor{UserID: budi.ID}, budi.ID); errs.KindOf(err) != errs.KindForbidden {
		t.Errorf("delete self: %v", err)
	}
	if err := e.user.Delete(ctx, adminActor, root.ID); errs.KindOf(err) != errs.KindForbidden {
		t.Errorf("admin deletes superadmin: %v", err)
	}
	if err := e.user.Delete(ctx, adminActor, budi.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.users.Get(ctx, budi.ID); errs.KindOf(err) != errs.KindNotFound {
		t.Error("user must be deleted")
	}
}
