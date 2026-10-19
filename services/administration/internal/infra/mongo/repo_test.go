package mongo

import (
	"context"
	"os"
	"testing"
	"time"

	"eam/services/administration/internal/domain/role"
	"eam/services/administration/internal/domain/user"
	"eam/services/common"
	"eam/services/common/errs"
)

type plainHasher struct{}

func (plainHasher) Hash(pw string) ([]byte, error)   { return []byte(pw), nil }
func (plainHasher) Compare(h []byte, pw string) bool { return string(h) == pw }

// Integration: MONGO_URI=... go test ./administration/internal/infra/mongo
func setup(t *testing.T) (*Users, *Roles) {
	if os.Getenv("MONGO_URI") == "" {
		t.Skip("MONGO_URI is empty")
	}
	t.Setenv("MONGO_DB", "eam_test_admin")
	db := common.Mongo()
	ctx := context.Background()
	db.Drop(ctx)
	t.Cleanup(func() { db.Drop(ctx) })
	u, r, err := New(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	return u, r
}

var t0 = time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)

func mkUser(t *testing.T, id, username, email string, roles ...string) *user.User {
	u, err := user.New(id, user.NewParams{Username: username, DisplayName: username, Email: email, Password: "password1", Roles: roles}, plainHasher{}, t0)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestUsersRepository(t *testing.T) {
	users, _ := setup(t)
	ctx := context.Background()
	budi := mkUser(t, "u1", "Budi", "budi@x.id", "user", "admin")
	if err := users.Insert(ctx, budi); err != nil {
		t.Fatal(err)
	}
	if err := users.Insert(ctx, mkUser(t, "u2", "BUDI", "other@x.id", "user")); errs.KindOf(err) != errs.KindConflict {
		t.Errorf("duplicate username with different case: %v", err)
	}

	got, err := users.FindByLogin(ctx, " BUDI@X.ID ")
	if err != nil || got.ID != "u1" || len(got.Roles) != 2 || !got.CreatedAt.Equal(t0) {
		t.Fatalf("FindByLogin email: %+v %v", got, err)
	}
	if _, err := users.Get(ctx, "nope"); errs.KindOf(err) != errs.KindNotFound {
		t.Errorf("Get missing: %v", err)
	}

	for i := 1; i <= user.MaxFailedLogins; i++ {
		n, locked, err := users.RecordFailedLogin(ctx, "u1")
		if err != nil || n != i || locked != (i == user.MaxFailedLogins) {
			t.Fatalf("attempt %d: n=%d locked=%v err=%v", i, n, locked, err)
		}
	}
	if err := users.ResetFailedLogins(ctx, "u1"); err != nil {
		t.Fatal(err)
	}
	got, _ = users.Get(ctx, "u1")
	if got.FailedAttempts != 0 || !got.Locked {
		t.Errorf("reset clears only the counter, lock stays: %+v", got)
	}

	got.DisplayName, got.Locked = "Budi New", false
	if err := users.Update(ctx, got); err != nil {
		t.Fatal(err)
	}
	if again, _ := users.Get(ctx, "u1"); again.DisplayName != "Budi New" || again.Locked {
		t.Errorf("update not saved: %+v", again)
	}
	if n, _ := users.CountWithRole(ctx, "admin"); n != 1 {
		t.Errorf("CountWithRole = %d", n)
	}
	if list, _ := users.List(ctx); len(list) != 1 {
		t.Errorf("List = %d", len(list))
	}
	if err := users.Delete(ctx, "u1"); err != nil {
		t.Fatal(err)
	}
	if err := users.Delete(ctx, "u1"); errs.KindOf(err) != errs.KindNotFound {
		t.Errorf("delete twice: %v", err)
	}
}

func TestRolesRepository(t *testing.T) {
	_, roles := setup(t)
	ctx := context.Background()
	for _, r := range role.Defaults(t0) {
		if err := roles.InsertIfMissing(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	admin, _ := roles.Get(ctx, "admin")
	admin.Description = "changed"
	roles.Update(ctx, admin)
	for _, r := range role.Defaults(t0) {
		roles.InsertIfMissing(ctx, r) // must not overwrite
	}
	if again, _ := roles.Get(ctx, "admin"); again.Description != "changed" || again.Permissions[role.AllMenus] == nil {
		t.Errorf("InsertIfMissing overwrote / permissions lost: %+v", again)
	}
	tek, _ := role.New("teknisi", "Admin", "", nil, t0)
	if err := roles.Insert(ctx, tek); errs.KindOf(err) != errs.KindConflict {
		t.Errorf("duplicate name: %v", err)
	}
	if n, _ := roles.CountExisting(ctx, []string{"admin", "user", "ghost"}); n != 2 {
		t.Errorf("CountExisting = %d", n)
	}
	sa, _ := roles.Get(ctx, role.SuperAdmin)
	if !sa.System {
		t.Error("system flag lost")
	}
	if err := roles.Delete(ctx, "user"); err != nil {
		t.Fatal(err)
	}
	if list, _ := roles.List(ctx); len(list) != 2 || list[0].ID != role.SuperAdmin {
		t.Errorf("List (system first) = %+v", list)
	}
}

// Legacy field "role" (string) is migrated to "roles" (array) by New.
func TestMigratesLegacyRoleField(t *testing.T) {
	if os.Getenv("MONGO_URI") == "" {
		t.Skip("MONGO_URI is empty")
	}
	t.Setenv("MONGO_DB", "eam_test_admin_legacy")
	db := common.Mongo()
	ctx := context.Background()
	db.Drop(ctx)
	t.Cleanup(func() { db.Drop(ctx) })
	db.Collection("users").InsertOne(ctx, map[string]any{"_id": "old", "username": "legacy", "usernameLower": "legacy", "emailLower": "l@x.id", "role": "admin"})
	users, _, err := New(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if u, _ := users.Get(ctx, "old"); u == nil || len(u.Roles) != 1 || u.Roles[0] != "admin" {
		t.Errorf("migration failed: %+v", u)
	}
}
