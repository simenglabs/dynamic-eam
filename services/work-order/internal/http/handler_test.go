package http

import (
	"net/http/httptest"
	"testing"
	"time"

	"eam/services/common"
	"github.com/gin-gonic/gin"
)

func TestPingRequiresLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r.Group("/api/work-order"))
	tok, _ := common.Sign("u1", "u1", "U1", time.Hour)
	for _, c := range []struct {
		auth string
		want int
	}{{"", 401}, {"Bearer " + tok, 200}} {
		req := httptest.NewRequest("GET", "/api/work-order/ping", nil)
		if c.auth != "" {
			req.Header.Set("Authorization", c.auth)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != c.want {
			t.Errorf("auth=%v: %d, want %d", c.auth != "", w.Code, c.want)
		}
	}
}
