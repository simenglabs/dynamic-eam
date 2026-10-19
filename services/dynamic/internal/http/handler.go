// Package http: HTTP adapter (gin) for dynamic menus. RBAC permissions are checked here before calling use cases.
package http

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"eam/services/common"
	"eam/services/common/errs"
	"eam/services/common/httpx"
	"eam/services/dynamic/internal/app"
	"eam/services/dynamic/internal/domain/content"
	"eam/services/dynamic/internal/domain/media"
	"github.com/gin-gonic/gin"
)

// Authorizer: RBAC port (common.Authz in production). Can writes the 403 itself when denied.
type Authorizer interface {
	Access(c *gin.Context) (*common.Access, error)
	Can(c *gin.Context, key, action string) bool
	SuperAdmin() gin.HandlerFunc
}

type handler struct {
	schemas *app.Schemas
	entries *app.Entries
	media   *app.Media
	az      Authorizer
}

func Register(rg *gin.RouterGroup, schemas *app.Schemas, entries *app.Entries, media *app.Media, az Authorizer) {
	h := &handler{schemas, entries, media, az}
	a := rg.Group("", common.Auth())
	super := az.SuperAdmin() // Content-Type Builder & Components are superadmin only

	a.GET("/content-types", h.listContentTypes)
	a.GET("/content-types/:uid", h.getContentType)
	a.POST("/content-types", super, h.createContentType)
	a.PUT("/content-types/:uid", super, h.updateContentType)
	a.DELETE("/content-types/:uid", super, h.deleteContentType)

	a.GET("/components", h.listComponents)
	a.GET("/components/:uid", h.getComponent)
	a.POST("/components", super, h.createComponent)
	a.PUT("/components/:uid", super, h.updateComponent)
	a.DELETE("/components/:uid", super, h.deleteComponent)

	a.GET("/content-types/:uid/entries", h.listEntries)
	a.GET("/content-types/:uid/entries/:id", h.getEntry)
	a.GET("/content-types/:uid/entries/:id/history", h.history)
	a.POST("/content-types/:uid/entries", h.createEntry)
	a.PUT("/content-types/:uid/entries/:id", h.updateEntry)
	a.DELETE("/content-types/:uid/entries/:id", h.deleteEntry)
	a.POST("/content-types/:uid/entries/:id/transitions/:key", h.transition)
	a.GET("/content-types/:uid/options", h.options)
	a.POST("/content-types/:uid/autofill", h.autofill)
	a.GET("/content-types/:uid/export", h.export)

	// User profile (content type user-profile, entry ID = user ID).
	a.GET("/profiles/me", h.myProfile)
	a.GET("/profiles/:userId", h.getProfile)
	a.PUT("/profiles/:userId", h.putProfile)

	// Domain: readable by every logged-in user (reference data for dropdowns & builder).
	a.GET("/domains", h.domains)
	a.GET("/classifications/:id/attributes", h.classAttributes)
	a.GET("/domains/:code/values", h.domainValues)

	a.POST("/media", h.upload)
	a.GET("/media", h.mediaMeta)
	a.GET("/media/:id", h.serveMedia)
}

func fail(c *gin.Context, err error) { httpx.Error(c, err) }

// ---------- schema ----------

func (h *handler) listContentTypes(c *gin.Context) {
	list, err := h.schemas.ListContentTypes(c)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, mapAll(list, toContentTypeDTO))
}

// getContentType: allowed with read permission on the content type OR permission on one of its views.
func (h *handler) getContentType(c *gin.Context) {
	ct, err := h.schemas.GetContentType(c, c.Param("uid"))
	if err != nil {
		fail(c, err)
		return
	}
	acc, err := h.az.Access(c)
	if err != nil {
		fail(c, err)
		return
	}
	ok := acc.Can(content.PermKey(ct.UID), common.Read)
	if ct.Workflow != nil {
		ok = ok || slices.ContainsFunc(ct.Workflow.Views, func(v content.View) bool { return acc.Can(content.ViewPermKey(ct.UID, v.Key), common.Read) })
	}
	if !ok {
		fail(c, errs.Forbidden("you don't have read permission on %s", ct.Label))
		return
	}
	c.JSON(http.StatusOK, toContentTypeDTO(ct))
}

func (h *handler) createContentType(c *gin.Context) {
	var in contentTypeDTO
	if !httpx.Bind(c, &in) {
		return
	}
	ct, err := h.schemas.CreateContentType(c, in.toContentType())
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, toContentTypeDTO(ct))
}

func (h *handler) updateContentType(c *gin.Context) {
	var in contentTypeDTO
	if !httpx.Bind(c, &in) {
		return
	}
	ct, err := h.schemas.UpdateContentType(c, c.Param("uid"), in.toContentType())
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toContentTypeDTO(ct))
}

func (h *handler) deleteContentType(c *gin.Context) {
	if err := h.schemas.DeleteContentType(c, c.Param("uid")); err != nil {
		fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *handler) listComponents(c *gin.Context) {
	list, err := h.schemas.ListComponents(c)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, mapAll(list, toComponentDTO))
}

func (h *handler) getComponent(c *gin.Context) {
	comp, err := h.schemas.GetComponent(c, c.Param("uid"))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toComponentDTO(comp))
}

func (h *handler) bindComponent(c *gin.Context) (*content.Component, bool) {
	var in componentDTO
	if !httpx.Bind(c, &in) {
		return nil, false
	}
	return &content.Component{UID: in.UID, Label: in.Label, Fields: mapAll(in.Fields, fieldDTO.toField)}, true
}

func (h *handler) createComponent(c *gin.Context) {
	in, ok := h.bindComponent(c)
	if !ok {
		return
	}
	comp, err := h.schemas.CreateComponent(c, in)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, toComponentDTO(comp))
}

func (h *handler) updateComponent(c *gin.Context) {
	in, ok := h.bindComponent(c)
	if !ok {
		return
	}
	comp, err := h.schemas.UpdateComponent(c, c.Param("uid"), in)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toComponentDTO(comp))
}

func (h *handler) deleteComponent(c *gin.Context) {
	if err := h.schemas.DeleteComponent(c, c.Param("uid")); err != nil {
		fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ---------- entries ----------

// readScope: without ?view = read permission on the content type; with ?view=<key> = read permission on that view, restricted to its statuses.
func (h *handler) readScope(c *gin.Context) (*content.ContentType, []string, bool) {
	ct, err := h.schemas.GetContentType(c, c.Param("uid"))
	if err != nil {
		fail(c, err)
		return nil, nil, false
	}
	key := c.Query("view")
	if key == "" {
		return ct, nil, h.az.Can(c, content.PermKey(ct.UID), common.Read)
	}
	if ct.Workflow == nil {
		fail(c, errs.NotFound("view"))
		return nil, nil, false
	}
	i := slices.IndexFunc(ct.Workflow.Views, func(v content.View) bool { return v.Key == key })
	if i < 0 {
		fail(c, errs.NotFound("view"))
		return nil, nil, false
	}
	return ct, ct.Workflow.Views[i].States, h.az.Can(c, content.ViewPermKey(ct.UID, key), common.Read)
}

func (h *handler) can(c *gin.Context, action string) bool {
	return h.az.Can(c, content.PermKey(c.Param("uid")), action)
}

// actor: user from the token + superadmin status + placement from the user profile (once per request).
// Used for history, per-site record restriction, and site/organization auto fill.
func (h *handler) actor(c *gin.Context) content.Actor {
	if v, ok := c.Get("actor"); ok {
		return v.(content.Actor)
	}
	u := common.CurrentUser(c)
	super := false
	if acc, err := h.az.Access(c); err == nil {
		super = acc.SuperAdmin
	}
	a, err := h.entries.ActorFor(c, u.Subject, u.Name, super)
	if err != nil {
		// ponytail: failing to read the profile = no sites (safe: restricted records are hidden); it does not fail the request.
		a = content.Actor{ID: u.Subject, Name: u.Name, Super: super}
	}
	c.Set("actor", a)
	return a
}

func (h *handler) listEntries(c *gin.Context) {
	ct, states, ok := h.readScope(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	// ?status=a,b: status filter for holders of read permission on the content type (views already restrict their own statuses).
	if states == nil && c.Query("status") != "" {
		states = strings.Split(c.Query("status"), ",")
	}
	res, err := h.entries.List(c, ct.UID, app.Query{Q: c.Query("q"), Page: page, Size: size, Sort: c.Query("sort"), States: states}, h.actor(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"total": res.Total, "page": max(page, 1), "size": min(max(size, 1), 100), "items": mapAll(res.Items, toEntryDTO)})
}

func (h *handler) getEntry(c *gin.Context) {
	ct, states, ok := h.readScope(c)
	if !ok {
		return
	}
	e, err := h.entries.Get(c, ct.UID, c.Param("id"), h.actor(c))
	if err != nil {
		fail(c, err)
		return
	}
	if states != nil && !slices.Contains(states, e.Status) {
		fail(c, errs.Forbidden("this record is not part of that view"))
		return
	}
	c.JSON(http.StatusOK, toEntryDTO(e))
}

// history: audit trail (create/update/delete/transition). Read permission on the content type; still readable after the entry is deleted.
func (h *handler) history(c *gin.Context) {
	if !h.can(c, common.Read) {
		return
	}
	revs, err := h.entries.History(c, c.Param("uid"), c.Param("id"), h.actor(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, mapAll(revs, toRevisionDTO))
}

func (h *handler) createEntry(c *gin.Context) {
	var data map[string]any
	if !h.can(c, common.Create) || !httpx.Bind(c, &data) {
		return
	}
	e, err := h.entries.Create(c, c.Param("uid"), data, h.actor(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, toEntryDTO(e))
}

// autofill: preview auto fill fields for the form (realtime), body {id?, data}.
func (h *handler) autofill(c *gin.Context) {
	var body struct {
		ID   string         `json:"id"`
		Data map[string]any `json:"data"`
	}
	if !httpx.Bind(c, &body) {
		return
	}
	action := common.Create
	if body.ID != "" {
		action = common.Update
	}
	if !h.can(c, action) {
		return
	}
	res, err := h.entries.Preview(c, c.Param("uid"), body.ID, body.Data, h.actor(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handler) updateEntry(c *gin.Context) {
	var data map[string]any
	if !h.can(c, common.Update) || !httpx.Bind(c, &data) {
		return
	}
	e, err := h.entries.Update(c, c.Param("uid"), c.Param("id"), data, h.actor(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toEntryDTO(e))
}

func (h *handler) deleteEntry(c *gin.Context) {
	if !h.can(c, common.Delete) {
		return
	}
	if err := h.entries.Delete(c, c.Param("uid"), c.Param("id"), h.actor(c)); err != nil {
		fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *handler) transition(c *gin.Context) {
	uid, key := c.Param("uid"), c.Param("key")
	if !h.az.Can(c, content.TransitionPermKey(uid, key), common.Update) {
		return
	}
	var in struct {
		Note string         `json:"note"`
		Data map[string]any `json:"data"` // transition form input
	}
	c.ShouldBindJSON(&in) // optional body
	e, err := h.entries.Transition(c, uid, c.Param("id"), key, h.actor(c), strings.TrimSpace(in.Note), in.Data)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toEntryDTO(e))
}

func (h *handler) options(c *gin.Context) {
	if !h.can(c, common.Read) {
		return
	}
	opts, err := h.entries.Options(c, c.Param("uid"), c.Query("q"), c.QueryMap("where"), h.actor(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, mapAll(opts, func(o app.Option) gin.H { return gin.H{"id": o.ID, "label": o.Label} }))
}

// export: all entries as CSV (the "download" action).
func (h *handler) export(c *gin.Context) {
	if !h.can(c, common.Download) {
		return
	}
	ct, all, err := h.entries.All(c, c.Param("uid"), h.actor(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-%s.csv"`, ct.UID, time.Now().Format("20060102-1504")))
	c.Writer.WriteString("\xEF\xBB\xBF") // BOM so Excel reads UTF-8
	w := csv.NewWriter(c.Writer)
	head := append([]string{"id"}, mapAll(ct.Fields, func(f content.Field) string { return f.Label })...)
	if ct.Workflow != nil {
		head = append(head, "workflowStatus")
	}
	w.Write(append(head, "createdAt", "updatedAt"))
	for _, e := range all {
		row := append([]string{e.ID}, mapAll(ct.Fields, func(f content.Field) string { return csvCell(e.Data[f.Name]) })...)
		if ct.Workflow != nil {
			row = append(row, e.Status)
		}
		w.Write(append(row, e.CreatedAt.Format(time.RFC3339), e.UpdatedAt.Format(time.RFC3339)))
	}
	w.Flush()
}

func csvCell(v any) string {
	var s string
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		s = x
	case []any, []string, map[string]any:
		b, _ := json.Marshal(x)
		s = string(b)
	default:
		s = fmt.Sprint(x)
	}
	return httpx.CSVSafe(s)
}

// ---------- user profile ----------

// profile: the profile entry, or an empty object (version 0) if not created yet.
func (h *handler) profile(c *gin.Context, userID string) {
	e, err := h.entries.Get(c, content.ProfileUID, userID, content.Actor{ID: userID, Super: true}) // profiles are not site-restricted; the caller checks permissions
	if errs.KindOf(err) == errs.KindNotFound {
		c.JSON(http.StatusOK, gin.H{"id": userID, "version": 0})
		return
	}
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toEntryDTO(e))
}

// myProfile: anyone may read their own profile.
func (h *handler) myProfile(c *gin.Context) { h.profile(c, common.CurrentUser(c).Subject) }

func (h *handler) getProfile(c *gin.Context) {
	if id := c.Param("userId"); id != common.CurrentUser(c).Subject && !h.az.Can(c, content.PermKey(content.ProfileUID), common.Read) {
		return
	}
	h.profile(c, c.Param("userId"))
}

// putProfile: changing a profile (including your own) needs update permission on ct/user-profile —
// the profile holds the placement (site/organization) that determines access, so users must not change it themselves.
func (h *handler) putProfile(c *gin.Context) {
	var data map[string]any
	if !h.az.Can(c, content.PermKey(content.ProfileUID), common.Update) || !httpx.Bind(c, &data) {
		return
	}
	e, err := h.entries.Upsert(c, content.ProfileUID, c.Param("userId"), data, h.actor(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toEntryDTO(e))
}

// ---------- domain ----------

// classAttributes: attributes of a classification (including inherited ones) for attributes-type form fields.
// Definitions only (code, label, type, unit, domain values) — not entry data — so login is enough.
func (h *handler) classAttributes(c *gin.Context) {
	defs, err := h.entries.AttributesOf(c, c.Param("id"))
	if err != nil {
		fail(c, err)
		return
	}
	out := make([]gin.H, len(defs))
	for i, d := range defs {
		out[i] = gin.H{"id": d.ID, "code": d.Code, "label": d.Label, "dataType": d.DataType, "unit": d.Unit,
			"domain": d.Domain, "length": d.Length, "mandatory": d.Mandatory}
		if d.Domain != "" {
			vals, _ := h.entries.DomainValues(c, d.Domain)
			out[i]["values"] = mapAll(vals, func(v content.DomainValue) gin.H { return gin.H{"valueId": v.ID, "valueDescription": v.Description} })
		}
	}
	c.JSON(http.StatusOK, out)
}

func (h *handler) domains(c *gin.Context) {
	list, err := h.entries.DomainCodes(c)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, mapAll(list, func(d app.DomainCode) gin.H { return gin.H{"code": d.Code, "description": d.Description} }))
}

func (h *handler) domainValues(c *gin.Context) {
	vals, err := h.entries.DomainValues(c, c.Param("code"))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, mapAll(vals, func(v content.DomainValue) gin.H {
		return gin.H{"valueId": v.ID, "valueDescription": v.Description, "isDefault": v.Default}
	}))
}

// ---------- media ----------

func (h *handler) upload(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, media.MaxSize+1<<20)
	fh, err := c.FormFile("file")
	if err != nil {
		fail(c, errs.Invalid("file is required (max %dMB)", media.MaxSize>>20))
		return
	}
	f, err := fh.Open()
	if err != nil {
		fail(c, err)
		return
	}
	defer f.Close()
	m, err := h.media.Upload(c, fh.Filename, fh.Size, f, h.actor(c).ID)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, mediaDTO(m))
}

func mediaDTO(m *media.Media) gin.H {
	return gin.H{"id": m.ID, "name": m.Name, "mime": m.Mime, "size": m.Size, "createdBy": m.CreatedBy, "createdAt": m.CreatedAt}
}

func (h *handler) mediaMeta(c *gin.Context) {
	list, err := h.media.Meta(c, strings.Split(c.Query("ids"), ","))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, mapAll(list, mediaDTO))
}

// ponytail: media files are readable by every logged-in user (not tied to content type permissions). Bind them to entries if files are sensitive.
func (h *handler) serveMedia(c *gin.Context) {
	m, rc, err := h.media.Open(c, c.Param("id"))
	if err != nil {
		fail(c, err)
		return
	}
	defer rc.Close()
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "sandbox")
	c.Header("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, m.Disposition(), m.SafeName()))
	c.DataFromReader(http.StatusOK, m.Size, m.Mime, io.Reader(rc), nil)
}
