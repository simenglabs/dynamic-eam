// Service sigv2. main = composition root.
package main

import (
	"net/http"

	"eam/services/common"
	zonehttp "eam/services/sigv2/internal/http"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	zonehttp.Register(r.Group("/api/sigv2"))
	r.Run(":" + common.Env("PORT", "8083"))
}
