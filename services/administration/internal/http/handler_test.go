package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"eam/services/administration/internal/app"
	"eam/services/administration/internal/domain/user"
	"eam/services/administration/internal/infra/memory"
	"eam/services/common"
	"github.com/gin-gonic/gin"
)

type plainHasher struct{}

func (plainHasher) Hash(pw string) ([]byte, error)   { return []byte("h:" + pw), nil }
func (plainHasher) Compare(h []byte, pw string) bool { return string(h) == "h:"+pw }

type tokens struct{}

func (tokens) Issue(u *user.User) (string, time.Duration, error) {
	t, err := common.Sign(u.ID, u.Username, u.DisplayName, time.Hour)
	return t, time.Hour, err
}

// fakeAuthz: izin per user ID diatur langsung di test (pengganti common.Authz yang butuh Mongo).
type fakeAuthz struct{ perms map[string]*common.Access }

func (f *fakeAuthz) Access(c *gin.Context) (*common.Access, error) {
	id := common.CurrentUser(c).Subject
	if a, ok := f.perms[id]; ok {
		return a, nil
	}
	return &common.Access{UserID: id, Permissions: map[string][]string{}}, nil
}

func (f *fakeAuthz) check(c *gin.Context, ok func(*common.Access) bool) {
	a, _ := f.Access(c)
	if !ok(a) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	c.Next()
}

func (f *fakeAuthz) Require(key, action string) gin.HandlerFunc {
	return func(c *gin.Context) { f.check(c, func(a *common.Access) bool { return a.Can(key, action) }) }
}

func (f *fakeAuthz) RequireAny(action string, keys ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		f.check(c, func(a *common.Access) bool {
			return slices.ContainsFunc(keys, func(k string) bool { return a.Can(k, action) })
		})
	}
}

func (f *fakeAuthz) Forget(string) {}

type env struct {
	r     *gin.Engine
	authz *fakeAuthz
	root  string // token superadmin
}

func newEnv(t *testing.T) *env {
	gin.SetMode(gin.TestMode)
	seq := 0
	d := app.Deps{Users: memory.NewUsers(), Roles: memory.NewRoles(), Hasher: plainHasher{}, Tokens: tokens{},
		Now: time.Now, NewID: func() string { seq++; return "id" + string(rune('a'+seq)) }}
	if _, err := app.Seed(context.Background(), d, "rootpassword"); err != nil {
		t.Fatal(err)
	}
	e := &env{r: gin.New(), authz: &fakeAuthz{perms: map[string]*common.Access{}}}
	Register(e.r.Group("/api/administration"), app.NewAuth(d), app.NewUsers(d), app.NewRoles(d), e.authz)
	root, _ := d.Users.FindByLogin(context.Background(), "admin")
	e.authz.perms[root.ID] = &common.Access{UserID: root.ID, SuperAdmin: true}
	e.root = e.login(t, "admin", "rootpassword")
	return e
}

func (e *env) do(method, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/administration"+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	return w
}

func (e *env) login(t *testing.T, u, p string) string {
	w := e.do("POST", "/auth/login", "", `{"username":"`+u+`","password":"`+p+`"}`)
	var res struct{ Token string }
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Token == "" {
		t.Fatalf("login %s gagal: %d %s", u, w.Code, w.Body)
	}
	return res.Token
}

func decode[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	var v T
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatalf("json: %v (%s)", err, w.Body)
	}
	return v
}

func TestLoginContract(t *testing.T) {
	e := newEnv(t)
	w := e.do("POST", "/auth/login", "", `{"username":"admin","password":"salah"}`)
	body := decode[map[string]any](t, w)
	if w.Code != 401 || body["error"] == "" || body["attemptsRemaining"] != float64(4) {
		t.Errorf("password salah: %d %v", w.Code, body)
	}
	w = e.do("POST", "/auth/login", "", `{"username":"admin"}`)
	if w.Code != 400 {
		t.Errorf("payload kurang: %d", w.Code)
	}
	w = e.do("POST", "/auth/login", "", `{"username":"admin","password":"rootpassword"}`)
	ok := decode[map[string]any](t, w)
	if w.Code != 200 || ok["token"] == nil || ok["expiresIn"] != float64(3600) || ok["user"].(map[string]any)["username"] != "admin" {
		t.Errorf("login benar: %d %v", w.Code, ok)
	}
}

func TestUserJSONNeverLeaksPasswordHash(t *testing.T) {
	e := newEnv(t)
	w := e.do("GET", "/users", e.root, "")
	if w.Code != 200 || strings.Contains(strings.ToLower(w.Body.String()), "hash") {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	users := decode[[]map[string]any](t, w)
	for _, k := range []string{"id", "username", "displayName", "email", "roles", "isActive", "isLocked", "failedAttempts", "createdAt", "updatedAt"} {
		if _, ok := users[0][k]; !ok {
			t.Errorf("field %q hilang dari kontrak user", k)
		}
	}
}

func TestUsersRequirePermission(t *testing.T) {
	e := newEnv(t)
	w := e.do("POST", "/users", e.root, `{"username":"budi","displayName":"Budi","email":"budi@x.id","roles":["user"],"password":"password1"}`)
	if w.Code != 201 {
		t.Fatalf("create: %d %s", w.Code, w.Body)
	}
	budi := e.login(t, "budi", "password1")
	if w := e.do("GET", "/users", budi, ""); w.Code != 403 {
		t.Errorf("tanpa izin: %d", w.Code)
	}
	if w := e.do("GET", "/users", "", ""); w.Code != 401 {
		t.Errorf("tanpa token: %d", w.Code)
	}
	id := decode[map[string]any](t, e.do("GET", "/auth/me", budi, ""))["id"].(string)
	e.authz.perms[id] = &common.Access{UserID: id, Permissions: map[string][]string{"administration/user-management": {"read"}}}
	if w := e.do("GET", "/users", budi, ""); w.Code != 200 {
		t.Errorf("dengan izin read: %d", w.Code)
	}
	if w := e.do("DELETE", "/users/x", budi, ""); w.Code != 403 {
		t.Errorf("read tidak boleh delete: %d", w.Code)
	}
}

func TestErrorMapping(t *testing.T) {
	e := newEnv(t)
	cases := []struct {
		method, path, body string
		want               int
	}{
		{"POST", "/users", `{"username":"x","displayName":"X","email":"salah","roles":["user"],"password":"password1"}`, 400},
		{"POST", "/users", `{bukan json`, 400},
		{"GET", "/users/tidak-ada", "", 404},
		{"POST", "/roles", `{"id":"admin","name":"Admin Lagi"}`, 409},
		{"PUT", "/roles/superadmin", `{"name":"Hack"}`, 403},
		{"DELETE", "/roles/superadmin", "", 403},
	}
	for _, c := range cases {
		w := e.do(c.method, c.path, e.root, c.body)
		if w.Code != c.want || decode[map[string]any](t, w)["error"] == "" {
			t.Errorf("%s %s: %d %s, want %d", c.method, c.path, w.Code, w.Body, c.want)
		}
	}
}

func TestRoleContract(t *testing.T) {
	e := newEnv(t)
	w := e.do("POST", "/roles", e.root, `{"id":"teknisi","name":"Teknisi","permissions":{"assets/tnd":["read","download"]}}`)
	if w.Code != 200 && w.Code != 201 {
		t.Fatalf("create role: %d %s", w.Code, w.Body)
	}
	r := decode[map[string]any](t, e.do("GET", "/roles/teknisi", e.root, ""))
	if r["name"] != "Teknisi" || r["system"] != false || len(r["permissions"].(map[string]any)["assets/tnd"].([]any)) != 2 {
		t.Errorf("kontrak role: %v", r)
	}
	if w := e.do("PUT", "/roles/teknisi", e.root, `{"name":"Teknisi 2","permissions":{}}`); w.Code != 200 {
		t.Errorf("update role: %d %s", w.Code, w.Body)
	}
	if w := e.do("DELETE", "/roles/teknisi", e.root, ""); w.Code != 204 {
		t.Errorf("delete role: %d %s", w.Code, w.Body)
	}
}

func TestMeReturnsAccess(t *testing.T) {
	e := newEnv(t)
	me := decode[map[string]any](t, e.do("GET", "/auth/me", e.root, ""))
	if me["username"] != "admin" || me["access"].(map[string]any)["superAdmin"] != true {
		t.Errorf("me: %v", me)
	}
}
