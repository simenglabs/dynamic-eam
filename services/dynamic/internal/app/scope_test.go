package app

import (
	"testing"

	"eam/services/common/errs"
	"eam/services/dynamic/internal/domain/content"
)

var (
	super  = content.Actor{ID: "root", Name: "Root", Super: true}
	userS1 = content.Actor{ID: "s1", Name: "Teknisi S1", Sites: []string{"S1"}, DefaultSite: "S1", Organization: "ORG1"}
	nobody = content.Actor{ID: "n", Name: "Tanpa site"}
)

// withScope: site master (S1, S2), organization (ORG1) and a "pole" content type restricted per site.
func withScope(t *testing.T, e *env) (s2Pole string) {
	t.Helper()
	site := &content.ContentType{UID: content.SiteUID, Label: "Sites", Parent: "master", DisplayField: "code",
		Fields: []content.Field{{Name: "code", Label: "Kode", Type: content.TText, Required: true}}}
	org := &content.ContentType{UID: content.OrganizationUID, Label: "Org", Parent: "master", DisplayField: "code",
		Fields: []content.Field{{Name: "code", Label: "Kode", Type: content.TText, Required: true}}}
	pole := &content.ContentType{UID: "pole", Label: "Tiang", Parent: "assets", DisplayField: "code", ScopeField: "site", Fields: []content.Field{
		{Name: "code", Label: "Kode", Type: content.TText, Required: true},
		{Name: "site", Label: "Site", Type: content.TRelation, Target: content.SiteUID},
		{Name: "organization", Label: "Organisasi", Type: content.TRelation, Target: content.OrganizationUID},
	}}
	if _, err := e.schemas.Ensure(e.ctx, nil, []*content.ContentType{site, org, pole}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"S1", "S2"} {
		if _, err := e.entries.Upsert(e.ctx, content.SiteUID, id, map[string]any{"code": id}, super); err != nil {
			t.Fatal(err)
		}
	}
	e.entries.Upsert(e.ctx, content.OrganizationUID, "ORG1", map[string]any{"code": "ORG1"}, super)
	e.entries.Create(e.ctx, "pole", map[string]any{"code": "P-S1", "site": "S1"}, super)
	p, err := e.entries.Create(e.ctx, "pole", map[string]any{"code": "P-S2", "site": "S2"}, super)
	if err != nil {
		t.Fatal(err)
	}
	return p.ID
}

func TestCreateFillsDefaultsAndEnforcesSite(t *testing.T) {
	e := newEnv()
	withScope(t, e)
	got, err := e.entries.Create(e.ctx, "pole", map[string]any{"code": "BARU"}, userS1)
	if err != nil || got.Data["site"] != "S1" || got.Data["organization"] != "ORG1" {
		t.Fatalf("auto fill site & organization: %+v %v", got, err)
	}
	if _, err := e.entries.Create(e.ctx, "pole", map[string]any{"code": "X", "site": "S2"}, userS1); errs.KindOf(err) != errs.KindForbidden {
		t.Errorf("create in a site outside access: %v", err)
	}
	if _, err := e.entries.Create(e.ctx, "pole", map[string]any{"code": "Y"}, nobody); errs.KindOf(err) != errs.KindInvalid {
		t.Errorf("user without sites must choose a site (and has no access): %v", err)
	}
	if got, err := e.entries.Create(e.ctx, "pole", map[string]any{"code": "Z", "site": "S2"}, super); err != nil || got.Data["site"] != "S2" {
		t.Errorf("superadmin is unrestricted: %v", err)
	}
}

func TestReadIsScoped(t *testing.T) {
	e := newEnv()
	s2 := withScope(t, e)
	count := func(a content.Actor) int {
		p, err := e.entries.List(e.ctx, "pole", Query{Page: 1, Size: 50}, a)
		if err != nil {
			t.Fatal(err)
		}
		return p.Total
	}
	if count(userS1) != 1 || count(nobody) != 0 || count(super) != 2 {
		t.Errorf("list: s1=%d nobody=%d super=%d", count(userS1), count(nobody), count(super))
	}
	if _, err := e.entries.Get(e.ctx, "pole", s2, userS1); errs.KindOf(err) != errs.KindNotFound {
		t.Errorf("detail from another site must look nonexistent: %v", err)
	}
	if _, err := e.entries.History(e.ctx, "pole", s2, userS1); errs.KindOf(err) != errs.KindNotFound {
		t.Errorf("history from another site: %v", err)
	}
	if opts, _ := e.entries.Options(e.ctx, "pole", "", nil, userS1); len(opts) != 1 {
		t.Errorf("relation options restricted by site: %+v", opts)
	}
	if _, all, _ := e.entries.All(e.ctx, "pole", userS1); len(all) != 1 {
		t.Errorf("export restricted by site: %d", len(all))
	}
}

func TestWriteOutsideScopeLooksMissing(t *testing.T) {
	e := newEnv()
	s2 := withScope(t, e)
	if _, err := e.entries.Update(e.ctx, "pole", s2, map[string]any{"code": "H", "site": "S2"}, userS1); errs.KindOf(err) != errs.KindNotFound {
		t.Errorf("update in another site: %v", err)
	}
	if err := e.entries.Delete(e.ctx, "pole", s2, userS1); errs.KindOf(err) != errs.KindNotFound {
		t.Errorf("delete in another site: %v", err)
	}
	mine, _ := e.entries.Create(e.ctx, "pole", map[string]any{"code": "MINE"}, userS1)
	if _, err := e.entries.Update(e.ctx, "pole", mine.ID, map[string]any{"code": "MINE", "site": "S2"}, userS1); errs.KindOf(err) != errs.KindForbidden {
		t.Errorf("moving a record to a site outside access: %v", err)
	}
}
