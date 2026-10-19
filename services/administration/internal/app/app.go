// Package app: administration use cases (login, manage users, manage roles).
// Depends only on domain ports; concrete implementations are injected from main.go.
package app

import (
	"time"

	"eam/services/administration/internal/domain/role"
	"eam/services/administration/internal/domain/user"
)

// TokenIssuer: session token issuer port (JWT in infra).
type TokenIssuer interface {
	Issue(u *user.User) (token string, ttl time.Duration, err error)
}

type Deps struct {
	Users  user.Repository
	Roles  role.Repository
	Hasher user.Hasher
	Tokens TokenIssuer
	Now    func() time.Time
	NewID  func() string
}

// Actor: the user performing the action (from RBAC access).
type Actor struct {
	UserID     string
	SuperAdmin bool
}
