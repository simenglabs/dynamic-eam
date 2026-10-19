package app

import (
	"context"
	"testing"
	"time"

	"eam/services/administration/internal/domain/role"
	"eam/services/administration/internal/infra/memory"
	"eam/services/common/errs"
)

func TestRoleLifecycle(t *testing.T) {
	ctx := context.Background()
	e := newEnv()
	r, err := e.role.Create(ctx, RoleInput{ID: "teknisi", Name: "Teknisi", Permissions: role.Permissions{"assets/tnd": {"read"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.role.Create(ctx, RoleInput{ID: "teknisi", Name: "Other"}); errs.KindOf(err) != errs.KindConflict {
		t.Errorf("duplicate ID: %v", err)
	}
	upd, err := e.role.Update(ctx, r.ID, RoleInput{Name: "Teknisi", Permissions: role.Permissions{"assets/tnd": {"read", "download"}}})
	if err != nil || len(upd.Permissions["assets/tnd"]) != 2 {
		t.Fatalf("update: %v %+v", err, upd)
	}

	e.user.Create(ctx, adminActor, newUser("teknisi"))
	if err := e.role.Delete(ctx, r.ID); errs.KindOf(err) != errs.KindConflict {
		t.Errorf("delete role in use: %v", err)
	}
}

func TestSystemRoleProtected(t *testing.T) {
	ctx := context.Background()
	e := newEnv()
	if _, err := e.role.Update(ctx, role.SuperAdmin, RoleInput{Name: "Hack"}); errs.KindOf(err) != errs.KindForbidden {
		t.Errorf("update system role: %v", err)
	}
	if err := e.role.Delete(ctx, role.SuperAdmin); errs.KindOf(err) != errs.KindForbidden {
		t.Errorf("delete system role: %v", err)
	}
}

func TestSeedIsIdempotent(t *testing.T) {
	e := newEnv()
	d := Deps{Users: e.users, Roles: e.roles, Hasher: fakeHasher{}, Tokens: fakeTokens{}, Now: func() time.Time { return t0 }, NewID: func() string { return "x" }}
	pw, err := Seed(context.Background(), d, "")
	if err != nil || pw != "" {
		t.Errorf("second seed must not create another superadmin: pw=%q err=%v", pw, err)
	}
	list, _ := e.users.List(context.Background())
	if len(list) != 1 {
		t.Errorf("user count = %d, want 1", len(list))
	}
}

func TestSeedGeneratesPasswordWhenEmpty(t *testing.T) {
	users, roles := memory.NewUsers(), memory.NewRoles()
	d := Deps{Users: users, Roles: roles, Hasher: fakeHasher{}, Tokens: fakeTokens{}, Now: func() time.Time { return t0 }, NewID: func() string { return "x" }}
	pw, err := Seed(context.Background(), d, "")
	if err != nil || len(pw) < 12 {
		t.Fatalf("pw=%q err=%v", pw, err)
	}
	u, _ := users.FindByLogin(context.Background(), "admin")
	if !u.IsSuperAdmin() || !(fakeHasher{}).Compare(u.PasswordHash, pw) {
		t.Error("superadmin must be created with the returned password")
	}
}
