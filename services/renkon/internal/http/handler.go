// Package http: HTTP adapter for service renkon.
// DDD layout when adding features: internal/domain/<aggregate> (entity + repository port, tests first),
// internal/app (use cases), internal/infra (mongo, etc.), then handlers here with
// az.Require("renkon/<sub menu>", common.Read) — key = path in packages/ui/src/menu.ts. Full example: services/administration.
package http

import (
	"net/http"

	"eam/services/common"
	"github.com/gin-gonic/gin"
)

func Register(rg *gin.RouterGroup) {
	a := rg.Group("", common.Auth())
	a.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"zone": "renkon"}) })
}
