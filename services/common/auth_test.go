package common

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/any", Auth(), func(c *gin.Context) { c.String(200, CurrentUser(c).Username) })

	user, _ := Sign("1", "budi", "Budi", time.Hour)
	expired, _ := Sign("1", "budi", "Budi", -time.Minute)

	cases := []struct {
		header, cookie string
		want           int
	}{
		{"", "", 401},
		{"Bearer garbage", "", 401},
		{"Bearer " + expired, "", 401},
		{"Bearer " + user, "", 200},
		{"", user, 200},
	}
	for i, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "/any", nil)
		if tc.header != "" {
			req.Header.Set("Authorization", tc.header)
		}
		if tc.cookie != "" {
			req.AddCookie(&http.Cookie{Name: CookieName, Value: tc.cookie})
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tc.want {
			t.Errorf("case %d: got %d want %d", i, w.Code, tc.want)
		}
	}
}

func TestAccessCan(t *testing.T) {
	a := &Access{Permissions: map[string][]string{"assets/tnd": {Read, Download}, AllMenus: {Read}}}
	checks := []struct {
		key, action string
		want        bool
	}{
		{"assets/tnd", Download, true},
		{"assets/tnd", Delete, false},
		{"renkon/prk", Read, true}, // from wildcard
		{"renkon/prk", Create, false},
	}
	for _, c := range checks {
		if a.Can(c.key, c.action) != c.want {
			t.Errorf("%s %s: want %v", c.key, c.action, c.want)
		}
	}
	if !(&Access{SuperAdmin: true}).Can("x", Delete) {
		t.Error("superadmin must always be allowed")
	}
}
