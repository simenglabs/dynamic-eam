package app

import (
	"context"
	"crypto/rand"

	"eam/services/administration/internal/domain/role"
	"eam/services/administration/internal/domain/user"
)

// Seed creates the built-in roles (if missing) and superadmin "admin" if no superadmin exists.
// Empty adminPassword = generated randomly; the created password is returned (empty if no user was created).
func Seed(ctx context.Context, d Deps, adminPassword string) (generated string, err error) {
	for _, r := range role.Defaults(d.Now()) {
		if err := d.Roles.InsertIfMissing(ctx, r); err != nil {
			return "", err
		}
	}
	n, err := d.Users.CountWithRole(ctx, role.SuperAdmin)
	if err != nil || n > 0 {
		return "", err
	}
	pw := adminPassword
	if pw == "" {
		pw = rand.Text()[:16]
		generated = pw
	}
	u, err := user.New(d.NewID(), user.NewParams{
		Username: "admin", DisplayName: "Super Administrator", Email: "admin@eam.local",
		Password: pw, Roles: []string{role.SuperAdmin},
	}, d.Hasher, d.Now())
	if err != nil {
		return "", err
	}
	return generated, d.Users.Insert(ctx, u)
}
