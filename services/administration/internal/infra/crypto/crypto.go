// Package crypto: adapter bcrypt (user.Hasher) dan JWT (app.TokenIssuer).
package crypto

import (
	"time"

	"eam/services/administration/internal/domain/user"
	"eam/services/common"
	"golang.org/x/crypto/bcrypt"
)

type Bcrypt struct{ Cost int }

func (b Bcrypt) Hash(pw string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(pw), max(b.Cost, bcrypt.MinCost))
}

func (Bcrypt) Compare(hash []byte, pw string) bool {
	return bcrypt.CompareHashAndPassword(hash, []byte(pw)) == nil
}

// JWT menerbitkan token sesi yang diverifikasi semua service lewat common.Auth.
type JWT struct{ TTL time.Duration }

func (j JWT) Issue(u *user.User) (string, time.Duration, error) {
	tok, err := common.Sign(u.ID, u.Username, u.DisplayName, j.TTL)
	return tok, j.TTL, err
}
