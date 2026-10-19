// Package app: use case administration (login, kelola user, kelola role).
// Bergantung hanya pada port domain; implementasi konkret disuntikkan dari main.go.
package app

import (
	"time"

	"eam/services/administration/internal/domain/role"
	"eam/services/administration/internal/domain/user"
)

// TokenIssuer: port penerbit token sesi (JWT di infra).
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

// Actor: user yang menjalankan aksi (dari hak akses RBAC).
type Actor struct {
	UserID     string
	SuperAdmin bool
}
