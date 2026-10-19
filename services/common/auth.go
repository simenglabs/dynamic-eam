// Package common: auth shared by all services. HS256 JWTs are issued by the administration service
// and verified by every service with the same JWT_SECRET.
package common

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const CookieName = "eam_token"

// System role with full access. Other roles are dynamic (collection "roles"), see rbac.go.
const RoleSuperAdmin = "superadmin"

// ponytail: default secret is dev-only; GIN_MODE=release without JWT_SECRET = service refuses to start.
const devSecret = "dev-secret-change-me"

// Claims hold identity only; roles & permissions are read from Mongo per request (Authz) so changes apply immediately.
type Claims struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	jwt.RegisteredClaims
}

func secret() []byte {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return []byte(s)
	}
	if gin.Mode() == gin.ReleaseMode {
		log.Fatal("JWT_SECRET is required when GIN_MODE=release")
	}
	return []byte(devSecret)
}

func Sign(userID, username, name string, ttl time.Duration) (string, error) {
	c := Claims{username, name, jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
	}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(secret())
}

func parse(tok string) (*Claims, error) {
	c := &Claims{}
	_, err := jwt.ParseWithClaims(tok, c, func(*jwt.Token) (any, error) { return secret(), nil },
		jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil {
		return nil, err
	}
	return c, nil
}

// Auth verifies the token from the "Authorization: Bearer" header or the eam_token cookie.
// Per-menu permissions are checked separately with Authz.Require / Authz.Can.
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tok, _ := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if tok == "" {
			tok, _ = c.Cookie(CookieName)
		}
		claims, err := parse(tok)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Set("claims", claims)
		c.Next()
	}
}

func CurrentUser(c *gin.Context) *Claims { return c.MustGet("claims").(*Claims) }
