// Service administration: login, user, role (RBAC). main = composition root.
package main

import (
	"context"
	"crypto/rand"
	"log"
	"net/http"
	"time"

	"eam/services/administration/internal/app"
	"eam/services/administration/internal/infra/crypto"
	"eam/services/administration/internal/infra/mongo"
	adminhttp "eam/services/administration/internal/http"
	"eam/services/common"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	ctx := context.Background()
	db := common.Mongo()
	users, roles, err := mongo.New(ctx, db)
	if err != nil {
		log.Fatal(err)
	}
	d := app.Deps{
		Users: users, Roles: roles,
		Hasher: crypto.Bcrypt{Cost: bcrypt.DefaultCost},
		Tokens: crypto.JWT{TTL: 8 * time.Hour},
		Now:    func() time.Time { return time.Now().UTC().Truncate(time.Millisecond) },
		NewID:  rand.Text,
	}
	pw, err := app.Seed(ctx, d, common.Env("ADMIN_PASSWORD", ""))
	if err != nil {
		log.Fatal(err)
	}
	if pw != "" {
		log.Printf("superadmin awal: username=admin password=%s (set ADMIN_PASSWORD untuk password tetap)", pw)
	}

	r := gin.Default()
	r.GET("/health", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	adminhttp.Register(r.Group("/api/administration"), app.NewAuth(d), app.NewUsers(d), app.NewRoles(d), common.NewAuthz(db))
	r.Run(":" + common.Env("PORT", "8087"))
}
