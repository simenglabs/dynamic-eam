// Package role: aggregate Role (kumpulan izin RBAC per menu).
package role

import (
	"regexp"
	"slices"
	"strings"
	"time"

	"eam/services/common"
	"eam/services/common/errs"
)

const (
	SuperAdmin = common.RoleSuperAdmin
	AllMenus   = common.AllMenus
)

// Permissions: kunci menu ("assets/tnd", "ct/<uid>", "*") -> aksi (common.Actions).
type Permissions map[string][]string

type Role struct {
	ID          string
	Name        string
	Description string
	System      bool // role sistem: tidak bisa diubah/dihapus
	Permissions Permissions
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

var (
	idRe  = regexp.MustCompile(`^[a-z][a-z0-9-]{1,47}$`)
	keyRe = regexp.MustCompile(`^(\*|[a-z0-9][a-z0-9/_-]{0,127})$`)
)

func New(id, name, description string, perms Permissions, now time.Time) (*Role, error) {
	if !idRe.MatchString(id) {
		return nil, errs.Invalid("ID role tidak valid (huruf kecil, angka, '-')")
	}
	r := &Role{ID: id, CreatedAt: now}
	if err := r.apply(name, description, perms, now); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Role) Edit(name, description string, perms Permissions, now time.Time) error {
	if r.System {
		return errs.Forbidden("role sistem tidak bisa diubah")
	}
	return r.apply(name, description, perms, now)
}

func (r *Role) CanDelete(usersWithRole int) error {
	if r.System {
		return errs.Forbidden("role sistem tidak bisa dihapus")
	}
	if usersWithRole > 0 {
		return errs.Conflict("role masih dipakai oleh %d user; lepas dulu dari user", usersWithRole)
	}
	return nil
}

func (r *Role) apply(name, description string, perms Permissions, now time.Time) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errs.Invalid("nama role wajib diisi")
	}
	clean := Permissions{}
	for key, actions := range perms {
		if !keyRe.MatchString(key) {
			return errs.Invalid("kunci menu tidak valid: %s", key)
		}
		for _, a := range actions {
			if !slices.Contains(common.Actions, a) {
				return errs.Invalid("aksi tidak dikenal: %s", a)
			}
		}
		if len(actions) > 0 {
			clean[key] = slices.Compact(slices.Sorted(slices.Values(actions)))
		}
	}
	r.Name, r.Description, r.Permissions, r.UpdatedAt = name, strings.TrimSpace(description), clean, now
	return nil
}

// Defaults: role bawaan yang di-seed saat service start (bila belum ada).
func Defaults(now time.Time) []*Role {
	return []*Role{
		{ID: SuperAdmin, Name: "Super Admin", Description: "Semua akses, termasuk Content-Type Builder", System: true, Permissions: Permissions{}, CreatedAt: now, UpdatedAt: now},
		{ID: "admin", Name: "Admin", Description: "Semua menu, semua aksi", Permissions: Permissions{AllMenus: common.Actions}, CreatedAt: now, UpdatedAt: now},
		{ID: "user", Name: "User", Description: "Lihat semua menu", Permissions: Permissions{AllMenus: {common.Read}}, CreatedAt: now, UpdatedAt: now},
	}
}
