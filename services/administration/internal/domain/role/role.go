// Package role: Role aggregate (a set of per-menu RBAC permissions).
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

// Permissions: menu key ("assets/tnd", "ct/<uid>", "*") -> actions (common.Actions).
type Permissions map[string][]string

type Role struct {
	ID          string
	Name        string
	Description string
	System      bool // system role: cannot be edited/deleted
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
		return nil, errs.Invalid("role ID is invalid (lowercase letters, digits, '-')")
	}
	r := &Role{ID: id, CreatedAt: now}
	if err := r.apply(name, description, perms, now); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Role) Edit(name, description string, perms Permissions, now time.Time) error {
	if r.System {
		return errs.Forbidden("system role cannot be edited")
	}
	return r.apply(name, description, perms, now)
}

func (r *Role) CanDelete(usersWithRole int) error {
	if r.System {
		return errs.Forbidden("system role cannot be deleted")
	}
	if usersWithRole > 0 {
		return errs.Conflict("role is still used by %d user(s); remove it from those users first", usersWithRole)
	}
	return nil
}

func (r *Role) apply(name, description string, perms Permissions, now time.Time) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errs.Invalid("role name is required")
	}
	clean := Permissions{}
	for key, actions := range perms {
		if !keyRe.MatchString(key) {
			return errs.Invalid("menu key is invalid: %s", key)
		}
		for _, a := range actions {
			if !slices.Contains(common.Actions, a) {
				return errs.Invalid("unknown action: %s", a)
			}
		}
		if len(actions) > 0 {
			clean[key] = slices.Compact(slices.Sorted(slices.Values(actions)))
		}
	}
	r.Name, r.Description, r.Permissions, r.UpdatedAt = name, strings.TrimSpace(description), clean, now
	return nil
}

// Defaults: built-in roles seeded at service start (if missing).
func Defaults(now time.Time) []*Role {
	return []*Role{
		{ID: SuperAdmin, Name: "Super Admin", Description: "Full access, including Content-Type Builder", System: true, Permissions: Permissions{}, CreatedAt: now, UpdatedAt: now},
		{ID: "admin", Name: "Admin", Description: "All menus, all actions", Permissions: Permissions{AllMenus: common.Actions}, CreatedAt: now, UpdatedAt: now},
		{ID: "user", Name: "User", Description: "View all menus", Permissions: Permissions{AllMenus: {common.Read}}, CreatedAt: now, UpdatedAt: now},
	}
}
