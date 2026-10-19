package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"eam/services/common"
	"eam/services/dynamic/internal/app"
	"eam/services/dynamic/internal/domain/content"
	"eam/services/dynamic/internal/infra/memory"
	"github.com/gin-gonic/gin"
)

// fakeAuthz: per-user permissions are set directly in the test.
type fakeAuthz struct{ perms map[string]*common.Access }

func (f *fakeAuthz) access(c *gin.Context) *common.Access {
	id := common.CurrentUser(c).Subject
	if a, ok := f.perms[id]; ok {
		return a
	}
	return &common.Access{UserID: id, Permissions: map[string][]string{}}
}

func (f *fakeAuthz) Access(c *gin.Context) (*common.Access, error) { return f.access(c), nil }

func (f *fakeAuthz) Can(c *gin.Context, key, action string) bool {
	if f.access(c).Can(key, action) {
		return true
	}
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	return false
}

func (f *fakeAuthz) SuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !f.access(c).SuperAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "khusus superadmin"})
			return
		}
		c.Next()
	}
}

type env struct {
	t       *testing.T
	r       *gin.Engine
	authz   *fakeAuthz
	index   *memory.Index
	schemas *app.Schemas
	entries *app.Entries
}

func newEnv(t *testing.T) *env {
	gin.SetMode(gin.TestMode)
	seq := 0
	d := app.Deps{Schemas: memory.NewSchemas(), Entries: memory.NewEntries(), Index: memory.NewIndex(), MediaRepo: memory.NewMedia(), Files: memory.NewFiles(),
		Now: func() time.Time { return time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC) }, NewID: func() string { seq++; return fmt.Sprintf("ID%d", seq) }}
	e := &env{t: t, r: gin.New(), authz: &fakeAuthz{perms: map[string]*common.Access{}}, index: d.Index.(*memory.Index), schemas: app.NewSchemas(d), entries: app.NewEntries(d)}
	Register(e.r.Group("/api/dynamic"), e.schemas, e.entries, app.NewMedia(d), e.authz)
	e.authz.perms["root"] = &common.Access{UserID: "root", SuperAdmin: true}
	return e
}

// grant: a user with specific permissions ("ct/vehicle" -> [read]).
func (e *env) grant(user string, perms map[string][]string) {
	e.authz.perms[user] = &common.Access{UserID: user, Permissions: perms}
}

func (e *env) do(user, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/dynamic"+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	tok, _ := common.Sign(user, user, user, time.Hour)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	return w
}

func (e *env) json(w *httptest.ResponseRecorder) map[string]any {
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		e.t.Fatalf("json %d: %s", w.Code, w.Body)
	}
	return m
}

const vehicleSchema = `{"uid":"vehicle","label":"Kendaraan","parent":"assets","displayField":"plate",
 "fields":[{"name":"plate","label":"Plat","type":"text","required":true,"unique":true},{"name":"status","label":"Kondisi","type":"select","options":["Baik","Rusak"]}],
 "layout":{"sections":[{"id":"utama","title":"Data Utama","rows":[{"columns":2,"fields":["plate","status"]}]}]},
 "workflow":{"states":[{"key":"draft","label":"Draft","color":"neutral","editable":true},{"key":"submitted","label":"Menunggu","color":"yellow"},{"key":"approved","label":"Disetujui","color":"green"}],
   "initial":"draft","transitions":[{"key":"submit","label":"Submit","from":["draft"],"to":"submitted"},{"key":"approve","label":"Approve","from":["submitted"],"to":"approved"}],
   "views":[{"key":"approval","label":"Approval Kendaraan","parent":"assets","states":["submitted"]}]}}`

func (e *env) createVehicle() {
	if w := e.do("root", "POST", "/content-types", vehicleSchema); w.Code != 201 {
		e.t.Fatalf("create schema: %d %s", w.Code, w.Body)
	}
}

func TestBuilderIsSuperAdminOnly(t *testing.T) {
	e := newEnv(t)
	e.grant("admin", map[string][]string{"*": common.Actions})
	if w := e.do("admin", "POST", "/content-types", vehicleSchema); w.Code != 403 {
		t.Errorf("admin creates schema: %d", w.Code)
	}
	if w := e.do("admin", "POST", "/components", `{}`); w.Code != 403 {
		t.Errorf("admin creates component: %d", w.Code)
	}
}

func TestSchemaContractRoundTrip(t *testing.T) {
	e := newEnv(t)
	e.createVehicle()
	ct := e.json(e.do("root", "GET", "/content-types/vehicle", ""))
	layout := ct["layout"].(map[string]any)["sections"].([]any)[0].(map[string]any)
	wf := ct["workflow"].(map[string]any)
	if layout["title"] != "Data Utama" || layout["rows"].([]any)[0].(map[string]any)["columns"] != float64(2) ||
		wf["initial"] != "draft" || len(wf["transitions"].([]any)) != 2 || wf["views"].([]any)[0].(map[string]any)["parent"] != "assets" {
		t.Errorf("schema contract: %v", ct)
	}
	if w := e.do("root", "POST", "/content-types", `{"uid":"x"}`); w.Code != 400 {
		t.Errorf("schema invalid: %d", w.Code)
	}
	if w := e.do("root", "POST", "/content-types", vehicleSchema); w.Code != 409 {
		t.Errorf("duplicate uid: %d", w.Code)
	}
}

func TestEntryPermissions(t *testing.T) {
	e := newEnv(t)
	e.createVehicle()
	e.grant("viewer", map[string][]string{"ct/vehicle": {"read"}})
	e.grant("editor", map[string][]string{"ct/vehicle": {"read", "create", "update", "delete"}})

	if w := e.do("viewer", "POST", "/content-types/vehicle/entries", `{"plate":"L 1"}`); w.Code != 403 {
		t.Errorf("viewer create: %d", w.Code)
	}
	w := e.do("editor", "POST", "/content-types/vehicle/entries", `{"plate":"L 1"}`)
	if w.Code != 201 {
		t.Fatalf("editor create: %d %s", w.Code, w.Body)
	}
	created := e.json(w)
	if created["plate"] != "L 1" || created["workflowStatus"] != "draft" || created["createdBy"] != "editor" {
		t.Errorf("entry contract: %v", created)
	}
	id := created["id"].(string)
	if w := e.do("viewer", "GET", "/content-types/vehicle/entries?q=l", ""); w.Code != 200 || e.json(w)["total"] != float64(1) {
		t.Errorf("viewer list: %d %s", w.Code, w.Body)
	}
	if w := e.do("viewer", "DELETE", "/content-types/vehicle/entries/"+id, ""); w.Code != 403 {
		t.Errorf("viewer delete: %d", w.Code)
	}
	if w := e.do("nobody", "GET", "/content-types/vehicle/entries", ""); w.Code != 403 {
		t.Errorf("list without permission: %d", w.Code)
	}
	if w := e.do("editor", "PUT", "/content-types/vehicle/entries/"+id, `{"plate":"L 1A"}`); w.Code != 200 {
		t.Errorf("editor update: %d %s", w.Code, w.Body)
	}
}

func TestWorkflowTransitionsAndViews(t *testing.T) {
	e := newEnv(t)
	e.createVehicle()
	e.grant("maker", map[string][]string{"ct/vehicle": {"read", "create", "update"}, "ct/vehicle/t/submit": {"update"}})
	e.grant("approver", map[string][]string{"ct/vehicle/v/approval": {"read"}, "ct/vehicle/t/approve": {"update"}})

	id := e.json(e.do("maker", "POST", "/content-types/vehicle/entries", `{"plate":"L 1"}`))["id"].(string)
	e.do("maker", "POST", "/content-types/vehicle/entries", `{"plate":"L 2"}`)

	if w := e.do("maker", "POST", "/content-types/vehicle/entries/"+id+"/transitions/approve", `{}`); w.Code != 403 {
		t.Errorf("maker approve: %d", w.Code)
	}
	w := e.do("maker", "POST", "/content-types/vehicle/entries/"+id+"/transitions/submit", `{"note":"please check"}`)
	if w.Code != 200 || e.json(w)["workflowStatus"] != "submitted" {
		t.Fatalf("submit: %d %s", w.Code, w.Body)
	}
	if w := e.do("maker", "PUT", "/content-types/vehicle/entries/"+id, `{"plate":"L 1B"}`); w.Code != 403 {
		t.Errorf("edit while submitted (locked): %d", w.Code)
	}

	// approver only has approval view permission: list without view is rejected, with view only status submitted.
	if w := e.do("approver", "GET", "/content-types/vehicle/entries", ""); w.Code != 403 {
		t.Errorf("approver regular list: %d", w.Code)
	}
	w = e.do("approver", "GET", "/content-types/vehicle/entries?view=approval", "")
	if w.Code != 200 || e.json(w)["total"] != float64(1) {
		t.Fatalf("approver view: %d %s", w.Code, w.Body)
	}
	if w := e.do("approver", "GET", "/content-types/vehicle/entries/"+id+"?view=approval", ""); w.Code != 200 {
		t.Errorf("approver detail via view: %d", w.Code)
	}
	if w := e.do("approver", "GET", "/content-types/vehicle", ""); w.Code != 200 {
		t.Errorf("approver reads schema: %d", w.Code)
	}
	w = e.do("approver", "POST", "/content-types/vehicle/entries/"+id+"/transitions/approve", `{}`)
	got := e.json(w)
	if w.Code != 200 || got["workflowStatus"] != "approved" || len(got["workflowHistory"].([]any)) != 2 {
		t.Errorf("approve: %d %v", w.Code, got)
	}
	if w := e.do("approver", "GET", "/content-types/vehicle/entries/"+id+"?view=approval", ""); w.Code != 403 {
		t.Errorf("after approval, the entry leaves the approval view: %d", w.Code)
	}
}

func TestExportCSV(t *testing.T) {
	e := newEnv(t)
	e.createVehicle()
	e.grant("dl", map[string][]string{"ct/vehicle": {"read", "download"}})
	e.do("root", "POST", "/content-types/vehicle/entries", `{"plate":"=HYPERLINK(\"x\")"}`)
	if w := e.do("root", "GET", "/content-types/vehicle/export", ""); w.Code != 200 {
		t.Fatalf("export: %d", w.Code)
	}
	e.grant("ro", map[string][]string{"ct/vehicle": {"read"}})
	if w := e.do("ro", "GET", "/content-types/vehicle/export", ""); w.Code != 403 {
		t.Errorf("without download: %d", w.Code)
	}
	w := e.do("dl", "GET", "/content-types/vehicle/export", "")
	body := w.Body.String()
	if !strings.HasPrefix(body, "\xEF\xBB\xBF") || !strings.Contains(body, "Plat") || !strings.Contains(body, `'=HYPERLINK`) || !strings.Contains(body, "workflowStatus") {
		t.Errorf("csv: %q", body)
	}
	if !strings.Contains(w.Header().Get("Content-Disposition"), "attachment") {
		t.Error("must be an attachment")
	}
}

func TestMediaUploadAndServeHeaders(t *testing.T) {
	e := newEnv(t)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", `evil".svg`)
	fw.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`))
	mw.Close()
	req := httptest.NewRequest("POST", "/api/dynamic/media", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	tok, _ := common.Sign("u1", "u1", "u1", time.Hour)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("upload: %d %s", w.Code, w.Body)
	}
	id := e.json(w)["id"].(string)
	w = e.do("u2", "GET", "/media/"+id, "")
	h := w.Header()
	if w.Code != 200 || h.Get("X-Content-Type-Options") != "nosniff" || h.Get("Content-Security-Policy") != "sandbox" ||
		!strings.HasPrefix(h.Get("Content-Disposition"), "attachment") || strings.Count(h.Get("Content-Disposition"), `"`) != 2 {
		t.Errorf("security headers: %d %v", w.Code, h)
	}
	if w := e.do("u2", "GET", "/media?ids="+id+",x", ""); w.Code != 200 || !strings.Contains(w.Body.String(), `"name"`) {
		t.Errorf("meta: %d %s", w.Code, w.Body)
	}
}

func TestIndexFailureIs502(t *testing.T) {
	e := newEnv(t)
	e.createVehicle()
	e.index.Fail = true
	if w := e.do("root", "GET", "/content-types/vehicle/entries", ""); w.Code != 502 {
		t.Errorf("list while ES is down: %d", w.Code)
	}
}

func TestListContentTypesForSidebar(t *testing.T) {
	e := newEnv(t)
	e.createVehicle()
	var list []map[string]any
	json.Unmarshal(e.do("nobody", "GET", "/content-types", "").Body.Bytes(), &list)
	if len(list) != 1 || list[0]["uid"] != "vehicle" || list[0]["workflow"] == nil {
		t.Errorf("list: %v", list)
	}
}

func TestListFilterByStatus(t *testing.T) {
	e := newEnv(t)
	e.createVehicle()
	id := e.json(e.do("root", "POST", "/content-types/vehicle/entries", `{"plate":"L 1"}`))["id"].(string)
	e.do("root", "POST", "/content-types/vehicle/entries", `{"plate":"L 2"}`)
	e.do("root", "POST", "/content-types/vehicle/entries/"+id+"/transitions/submit", `{}`)
	e.grant("viewer", map[string][]string{"ct/vehicle": {"read"}})
	for status, want := range map[string]float64{"draft": 1, "submitted": 1, "draft,submitted": 2, "approved": 0} {
		w := e.do("viewer", "GET", "/content-types/vehicle/entries?status="+status, "")
		if w.Code != 200 || e.json(w)["total"] != want {
			t.Errorf("status=%s: %d %s", status, w.Code, w.Body)
		}
	}
}

func TestHistoryAndSoftDelete(t *testing.T) {
	e := newEnv(t)
	e.createVehicle()
	e.grant("editor", map[string][]string{"ct/vehicle": {"read", "create", "update", "delete"}})
	created := e.json(e.do("editor", "POST", "/content-types/vehicle/entries", `{"plate":"L 1","status":"Baik"}`))
	id := created["id"].(string)
	if created["version"] != float64(1) || created["createdBy"] != "editor" || created["updatedBy"] != "editor" {
		t.Errorf("meta create: %v", created)
	}
	e.do("editor", "PUT", "/content-types/vehicle/entries/"+id, `{"plate":"L 1","status":"Rusak"}`)
	if w := e.do("editor", "DELETE", "/content-types/vehicle/entries/"+id, ""); w.Code != 204 {
		t.Fatalf("delete: %d %s", w.Code, w.Body)
	}
	if w := e.do("editor", "GET", "/content-types/vehicle/entries/"+id, ""); w.Code != 404 {
		t.Errorf("deleted entry: %d", w.Code)
	}

	w := e.do("editor", "GET", "/content-types/vehicle/entries/"+id+"/history", "")
	var hist []map[string]any
	json.Unmarshal(w.Body.Bytes(), &hist)
	if w.Code != 200 || len(hist) != 3 {
		t.Fatalf("history: %d %s", w.Code, w.Body)
	}
	upd := hist[1]
	ch := upd["changes"].([]any)[0].(map[string]any)
	if hist[0]["action"] != "create" || upd["action"] != "update" || hist[2]["action"] != "delete" ||
		upd["byName"] != "editor" || upd["version"] != float64(2) || ch["field"] != "status" || ch["from"] != "Baik" || ch["to"] != "Rusak" {
		t.Errorf("history content: %v", hist)
	}
	if w := e.do("nobody", "GET", "/content-types/vehicle/entries/"+id+"/history", ""); w.Code != 403 {
		t.Errorf("history without read permission: %d", w.Code)
	}
}

func TestDomainEndpointsAndField(t *testing.T) {
	e := newEnv(t)
	comps, cts, entries, err := DecodeSeed(seedMaster(t))
	if err != nil {
		t.Fatal(err)
	}
	e.ensure(comps, cts, entries)

	var codes []map[string]any
	json.Unmarshal(e.do("viewer", "GET", "/domains", "").Body.Bytes(), &codes)
	if len(codes) < 2 || codes[0]["code"] == nil {
		t.Fatalf("domains: %v", codes)
	}
	var vals []map[string]any
	w := e.do("viewer", "GET", "/domains/PRIORITY/values", "")
	json.Unmarshal(w.Body.Bytes(), &vals)
	if w.Code != 200 || len(vals) != 3 || vals[0]["valueId"] != "LOW" || vals[0]["valueDescription"] == nil {
		t.Fatalf("values: %d %s", w.Code, w.Body)
	}
	if w := e.do("viewer", "GET", "/domains/NOPE/values", ""); w.Code != 404 {
		t.Errorf("missing domain: %d", w.Code)
	}

	schema := `{"uid":"task","label":"Task","parent":"renkon","displayField":"name","fields":[
	 {"name":"name","label":"Nama","type":"text","required":true},
	 {"name":"priority","label":"Priority","type":"domain","domain":"PRIORITY","required":true},
	 {"name":"by","label":"Oleh","type":"text","autoFill":{"source":"user","lock":true}},
	 {"name":"no","label":"No","type":"text","autoFill":{"source":"pattern","pattern":"V-{seq:3}","lock":true}},
	 {"name":"parentTask","label":"Induk","type":"relation","target":"task","where":[{"field":"priority","value":"{priority}"}]}]}`
	if w := e.do("root", "POST", "/content-types", schema); w.Code != 201 {
		t.Fatalf("schema domain: %d %s", w.Code, w.Body)
	}
	ct := e.json(e.do("root", "GET", "/content-types/task", ""))
	if f := ct["fields"].([]any)[1].(map[string]any); f["domain"] != "PRIORITY" || f["type"] != "domain" {
		t.Errorf("domain field contract: %v", f)
	}
	if af, _ := ct["fields"].([]any)[2].(map[string]any)["autoFill"].(map[string]any); af["source"] != "user" || af["lock"] != true {
		t.Errorf("autoFill contract: %v", af)
	}
	if af, _ := ct["fields"].([]any)[3].(map[string]any)["autoFill"].(map[string]any); af["pattern"] != "V-{seq:3}" {
		t.Errorf("code pattern contract: %v", af)
	}
	if w, _ := ct["fields"].([]any)[4].(map[string]any)["where"].([]any); len(w) != 1 || w[0].(map[string]any)["value"] != "{priority}" {
		t.Errorf("where contract: %v", w)
	}
	if w := e.do("viewer", "GET", "/classifications/nope/attributes", ""); w.Code != 404 {
		t.Errorf("missing classification attributes: %d", w.Code)
	}
	if got := e.json(e.do("root", "POST", "/content-types/task/autofill", `{"data":{"name":"A"}}`)); got["no"] != "V-001" || got["by"] == nil || got["name"] != nil {
		t.Errorf("preview autofill: %v", got)
	}
	if w := e.do("viewer", "POST", "/content-types/task/autofill", `{"data":{}}`); w.Code != 403 {
		t.Errorf("preview without create permission: %d", w.Code)
	}
	if w := e.do("root", "POST", "/content-types/task/entries", `{"name":"A","priority":"NOPE"}`); w.Code != 400 {
		t.Errorf("value outside the domain: %d", w.Code)
	}
	if got := e.json(e.do("root", "POST", "/content-types/task/entries", `{"name":"A"}`)); got["priority"] != "MEDIUM" {
		t.Errorf("default domain: %v", got)
	}
}

func TestUserProfileEndpoints(t *testing.T) {
	e := newEnv(t)
	schema := `{"uid":"user-profile","label":"Profil User","parent":"administration","hidden":true,"displayField":"note",
	 "fields":[{"name":"note","label":"Catatan","type":"text"}]}`
	if w := e.do("root", "POST", "/content-types", schema); w.Code != 201 {
		t.Fatalf("profile schema: %d %s", w.Code, w.Body)
	}
	if ct := e.json(e.do("root", "GET", "/content-types/user-profile", "")); ct["hidden"] != true {
		t.Errorf("hidden missing from contract: %v", ct)
	}
	e.grant("hr", map[string][]string{"ct/user-profile": {"read", "update"}})

	if w := e.do("budi", "PUT", "/profiles/budi", `{"note":"x"}`); w.Code != 403 {
		t.Errorf("a regular user must not change profiles (including their own): %d", w.Code)
	}
	w := e.do("hr", "PUT", "/profiles/budi", `{"note":"Teknisi lapangan"}`)
	if w.Code != 200 || e.json(w)["id"] != "budi" {
		t.Fatalf("hr upsert profile: %d %s", w.Code, w.Body)
	}
	if got := e.json(e.do("budi", "GET", "/profiles/me", "")); got["note"] != "Teknisi lapangan" || got["id"] != "budi" {
		t.Errorf("own profile: %v", got)
	}
	if got := e.json(e.do("ani", "GET", "/profiles/me", "")); got["id"] != "ani" || got["version"] != float64(0) {
		t.Errorf("no profile yet = empty object version 0: %v", got)
	}
	if w := e.do("ani", "GET", "/profiles/budi", ""); w.Code != 403 {
		t.Errorf("another user without permission reads someone else's profile: %d", w.Code)
	}
	if w := e.do("hr", "GET", "/profiles/budi", ""); w.Code != 200 {
		t.Errorf("hr reads profile: %d", w.Code)
	}
}

func TestSiteScopingFromUserProfile(t *testing.T) {
	e := newEnv(t)
	super := content.Actor{ID: "root", Super: true}
	ctx := context.Background()
	site := &content.ContentType{UID: content.SiteUID, Label: "Sites", Parent: "master", DisplayField: "code", Fields: []content.Field{{Name: "code", Label: "Kode", Type: content.TText}}}
	pole := &content.ContentType{UID: "pole", Label: "Tiang", Parent: "assets", DisplayField: "code", ScopeField: "site", Fields: []content.Field{
		{Name: "code", Label: "Kode", Type: content.TText, Required: true}, {Name: "site", Label: "Site", Type: content.TRelation, Target: content.SiteUID}}}
	profile := &content.ContentType{UID: content.ProfileUID, Label: "Profil", Parent: "administration", Hidden: true, DisplayField: "defaultSite", Fields: []content.Field{
		{Name: "sites", Label: "Sites", Type: content.TRelation, Target: content.SiteUID, Multiple: true},
		{Name: "defaultSite", Label: "Default", Type: content.TRelation, Target: content.SiteUID}}}
	if _, err := e.schemas.Ensure(ctx, nil, []*content.ContentType{site, pole, profile}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"S1", "S2", "S3"} {
		e.entries.Upsert(ctx, content.SiteUID, id, map[string]any{"code": id}, super)
	}
	e.entries.Upsert(ctx, content.ProfileUID, "tek", map[string]any{"defaultSite": "S1", "sites": []any{"S3"}}, super)
	for _, s := range []string{"S1", "S2", "S3"} {
		e.entries.Create(ctx, "pole", map[string]any{"code": "P-" + s, "site": s}, super)
	}
	e.grant("tek", map[string][]string{"ct/pole": {"read", "create"}})

	if ct := e.json(e.do("root", "GET", "/content-types/pole", "")); ct["scopeField"] != "site" {
		t.Errorf("scopeField missing from contract: %v", ct["scopeField"])
	}
	if got := e.json(e.do("tek", "GET", "/content-types/pole/entries", "")); got["total"] != float64(2) {
		t.Errorf("technician sees only accessible sites (default S1 + S3): %v", got["total"])
	}
	if got := e.json(e.do("root", "GET", "/content-types/pole/entries", "")); got["total"] != float64(3) {
		t.Errorf("superadmin sees everything: %v", got["total"])
	}
	created := e.json(e.do("tek", "POST", "/content-types/pole/entries", `{"code":"BARU"}`))
	if created["site"] != "S1" {
		t.Errorf("site auto filled from the profile default site: %v", created)
	}
	if w := e.do("tek", "POST", "/content-types/pole/entries", `{"code":"X","site":"S2"}`); w.Code != 403 {
		t.Errorf("site outside access: %d %s", w.Code, w.Body)
	}
	e.grant("baru", map[string][]string{"ct/pole": {"read"}}) // no profile yet
	if got := e.json(e.do("baru", "GET", "/content-types/pole/entries", "")); got["total"] != float64(0) {
		t.Errorf("user without profile/site sees no records: %v", got["total"])
	}
}

func TestTransitionInputs(t *testing.T) {
	e := newEnv(t)
	schema := `{"uid":"dok","label":"Dokumen","parent":"assets","displayField":"name","fields":[{"name":"name","label":"Nama","type":"text"}],
	 "workflow":{"states":[{"key":"draft","label":"Draft","color":"neutral","editable":true},{"key":"sent","label":"Dikirim","color":"yellow"}],"initial":"draft",
	 "transitions":[{"key":"submit","label":"Submit","from":["draft"],"to":"sent","fields":[{"name":"jenis","label":"Jenis","type":"select","options":["BA","Foto"],"required":true}]}]}}`
	if w := e.do("root", "POST", "/content-types", schema); w.Code != 201 {
		t.Fatalf("schema: %d %s", w.Code, w.Body)
	}
	tf := e.json(e.do("root", "GET", "/content-types/dok", ""))["workflow"].(map[string]any)["transitions"].([]any)[0].(map[string]any)["fields"].([]any)
	if len(tf) != 1 || tf[0].(map[string]any)["name"] != "jenis" {
		t.Errorf("transition input contract: %v", tf)
	}
	id := e.json(e.do("root", "POST", "/content-types/dok/entries", `{"name":"A"}`))["id"].(string)
	if w := e.do("root", "POST", "/content-types/dok/entries/"+id+"/transitions/submit", `{}`); w.Code != 400 {
		t.Errorf("empty required input: %d %s", w.Code, w.Body)
	}
	if w := e.do("root", "POST", "/content-types/dok/entries/"+id+"/transitions/submit", `{"note":"ok","data":{"jenis":"BA"}}`); w.Code != 200 {
		t.Fatalf("transition: %d %s", w.Code, w.Body)
	}
	var hist []map[string]any
	json.Unmarshal(e.do("root", "GET", "/content-types/dok/entries/"+id+"/history", "").Body.Bytes(), &hist)
	last := hist[len(hist)-1]
	if last["transition"] != "submit" || last["data"].(map[string]any)["jenis"] != "BA" {
		t.Errorf("transition input history: %v", last)
	}
}
