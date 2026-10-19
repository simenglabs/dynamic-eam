package app

import (
	"slices"
	"testing"

	"eam/services/dynamic/internal/domain/content"
)

func TestAutoFillOnSave(t *testing.T) {
	e := newEnv()
	e.seed() // location (name), vehicle
	locA, _ := e.entries.Create(e.ctx, "location", map[string]any{"name": "Gudang A"}, u1)
	locB, _ := e.entries.Create(e.ctx, "location", map[string]any{"name": "Gudang B"}, u1)
	ct, _ := e.schemas.GetContentType(e.ctx, "vehicle")
	ct.Fields = append(ct.Fields,
		content.Field{Name: "locName", Label: "Nama Lokasi", Type: content.TText, AutoFill: &content.AutoFill{Source: content.AutoRelation, From: "location", Field: "name", Lock: true}},
		content.Field{Name: "createdByName", Label: "Dibuat oleh", Type: content.TText, AutoFill: &content.AutoFill{Source: content.AutoUser}},
	)
	if _, err := e.schemas.UpdateContentType(e.ctx, "vehicle", ct); err != nil {
		t.Fatal(err)
	}
	v, err := e.entries.Create(e.ctx, "vehicle", map[string]any{"plate": "L 1", "location": locA.ID, "locName": "palsu"}, u1)
	if err != nil || v.Data["locName"] != "Gudang A" || v.Data["createdByName"] != "Budi" {
		t.Fatalf("create: %+v %v", v, err)
	}
	v, err = e.entries.Update(e.ctx, "vehicle", v.ID, map[string]any{"plate": "L 1", "location": locB.ID, "locName": "Gudang A", "createdByName": "Budi"}, u2)
	if err != nil || v.Data["locName"] != "Gudang B" || v.Data["createdByName"] != "Budi" {
		t.Errorf("update: relasi berubah -> terkunci ikut berubah; tidak terkunci tetap: %+v %v", v, err)
	}
}

func TestActorForExpandsSiteHierarchyAndProfile(t *testing.T) {
	e := newEnv()
	site := &content.ContentType{UID: content.SiteUID, Label: "Sites", Parent: "master", DisplayField: "code", Fields: []content.Field{
		{Name: "code", Label: "Kode", Type: content.TText}, {Name: "parent", Label: "Induk", Type: content.TRelation, Target: content.SiteUID}}}
	prof := &content.ContentType{UID: content.ProfileUID, Label: "Profil", Parent: "administration", Hidden: true, DisplayField: "jobTitle", Fields: []content.Field{
		{Name: "sites", Label: "Sites", Type: content.TRelation, Target: content.SiteUID, Multiple: true},
		{Name: "defaultSite", Label: "Default", Type: content.TRelation, Target: content.SiteUID},
		{Name: "jobTitle", Label: "Jabatan", Type: content.TText}}}
	if _, err := e.schemas.Ensure(e.ctx, nil, []*content.ContentType{site, prof}); err != nil {
		t.Fatal(err)
	}
	// UID -> UP3 -> {ULP1, ULP2}; LAIN terpisah
	for _, s := range []struct{ id, parent string }{{"UID", ""}, {"UP3", "UID"}, {"ULP1", "UP3"}, {"ULP2", "UP3"}, {"LAIN", ""}} {
		data := map[string]any{"code": s.id}
		if s.parent != "" {
			data["parent"] = s.parent
		}
		if _, err := e.entries.Upsert(e.ctx, content.SiteUID, s.id, data, super); err != nil {
			t.Fatal(err)
		}
	}
	e.entries.Upsert(e.ctx, content.ProfileUID, "mgr", map[string]any{"sites": []any{"UP3"}, "defaultSite": "UP3", "jobTitle": "Manajer UP3"}, super)
	a, err := e.entries.ActorFor(e.ctx, "mgr", "Manajer", false)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(a.Sites)
	if !slices.Equal(a.Sites, []string{"ULP1", "ULP2", "UP3"}) || a.DefaultSite != "UP3" || a.Profile["jobTitle"] != "Manajer UP3" || a.Name != "Manajer" {
		t.Errorf("actor: %+v", a)
	}
	none, _ := e.entries.ActorFor(e.ctx, "baru", "Baru", false)
	if len(none.Sites) != 0 {
		t.Errorf("tanpa profil: %+v", none.Sites)
	}
}

func TestPatternCodeOnSave(t *testing.T) {
	e := newEnv()
	e.seed()
	loc, _ := e.entries.Create(e.ctx, "location", map[string]any{"name": "GDG"}, u1)
	ct, _ := e.schemas.GetContentType(e.ctx, "vehicle")
	ct.Fields = append(ct.Fields, content.Field{Name: "code", Label: "Kode", Type: content.TText, Unique: true,
		AutoFill: &content.AutoFill{Source: content.AutoPattern, Pattern: "{location.name}-{seq:3}", Lock: true}})
	if _, err := e.schemas.UpdateContentType(e.ctx, "vehicle", ct); err != nil {
		t.Fatal(err)
	}
	a, err := e.entries.Create(e.ctx, "vehicle", map[string]any{"plate": "L 1", "location": loc.ID}, u1)
	if err != nil || a.Data["code"] != "GDG-001" {
		t.Fatalf("pertama: %+v %v", a, err)
	}
	b, _ := e.entries.Create(e.ctx, "vehicle", map[string]any{"plate": "L 2", "location": loc.ID, "code": "GDG-001"}, u1)
	if b.Data["code"] != "GDG-002" {
		t.Errorf("kedua: %v", b.Data["code"])
	}
	a, err = e.entries.Update(e.ctx, "vehicle", a.ID, map[string]any{"plate": "L 1b", "location": loc.ID}, u2)
	if err != nil || a.Data["code"] != "GDG-001" {
		t.Errorf("update tidak mengubah kode: %+v %v", a, err)
	}
}

func TestPreviewAutoFill(t *testing.T) {
	e := newEnv()
	e.seed()
	loc, _ := e.entries.Create(e.ctx, "location", map[string]any{"name": "GDG"}, u1)
	ct, _ := e.schemas.GetContentType(e.ctx, "vehicle")
	ct.Fields = append(ct.Fields,
		content.Field{Name: "locName", Label: "Nama Lokasi", Type: content.TText, AutoFill: &content.AutoFill{Source: content.AutoRelation, From: "location", Field: "name", Lock: true}},
		content.Field{Name: "code", Label: "Kode", Type: content.TText, AutoFill: &content.AutoFill{Source: content.AutoPattern, Pattern: "{location.name}-{seq:3}", Lock: true}})
	if _, err := e.schemas.UpdateContentType(e.ctx, "vehicle", ct); err != nil {
		t.Fatal(err)
	}
	for range 2 { // preview tidak memakai nomor urut
		got, err := e.entries.Preview(e.ctx, "vehicle", "", map[string]any{"location": loc.ID}, u1)
		if err != nil || got["locName"] != "GDG" || got["code"] != "GDG-001" {
			t.Fatalf("preview: %v %v", got, err)
		}
		if _, ok := got["plate"]; ok {
			t.Error("preview hanya field otomatis")
		}
	}
	v, _ := e.entries.Create(e.ctx, "vehicle", map[string]any{"plate": "L 1", "location": loc.ID}, u1)
	if v.Data["code"] != "GDG-001" {
		t.Errorf("create setelah preview: %v", v.Data["code"])
	}
	got, _ := e.entries.Preview(e.ctx, "vehicle", v.ID, map[string]any{"location": loc.ID}, u1)
	if got["code"] != "GDG-001" {
		t.Errorf("preview entri lama: kode tetap: %v", got["code"])
	}
	if got, _ := e.entries.Preview(e.ctx, "vehicle", "", map[string]any{}, u1); got["code"] != "-001" || got["locName"] != nil {
		t.Errorf("tanpa lokasi: %v", got)
	}
}

// Pemilik = user login: profil dibuat otomatis (dengan nama) supaya relasi valid & labelnya nama user.
func TestAutoUserOwnerEnsuresProfile(t *testing.T) {
	e := newEnv()
	e.seed()
	prof := &content.ContentType{UID: content.ProfileUID, Label: "Profil", Parent: "administration", Hidden: true, DisplayField: "{name}", Fields: []content.Field{
		{Name: "name", Label: "Nama", Type: content.TText}, {Name: "jobTitle", Label: "Jabatan", Type: content.TText}}}
	if _, err := e.schemas.Ensure(e.ctx, nil, []*content.ContentType{prof}); err != nil {
		t.Fatal(err)
	}
	ct, _ := e.schemas.GetContentType(e.ctx, "vehicle")
	ct.Fields = append(ct.Fields, content.Field{Name: "owner", Label: "Pemilik", Type: content.TRelation, Target: content.ProfileUID, Required: true,
		AutoFill: &content.AutoFill{Source: content.AutoUser, Lock: true}})
	if _, err := e.schemas.UpdateContentType(e.ctx, "vehicle", ct); err != nil {
		t.Fatal(err)
	}
	v, err := e.entries.Create(e.ctx, "vehicle", map[string]any{"plate": "L 1"}, u1)
	if err != nil || v.Data["owner"] != "u1" {
		t.Fatalf("create: %+v %v", v, err)
	}
	p, err := e.entries.Get(e.ctx, content.ProfileUID, "u1", super)
	if err != nil || p.Data["name"] != "Budi" {
		t.Fatalf("profil otomatis: %+v %v", p, err)
	}
	// profil yang sudah ada tidak ditimpa, hanya namanya disamakan
	if _, err := e.entries.Upsert(e.ctx, content.ProfileUID, "u2", map[string]any{"jobTitle": "Teknisi"}, super); err != nil {
		t.Fatal(err)
	}
	if _, err := e.entries.Create(e.ctx, "vehicle", map[string]any{"plate": "L 2"}, u2); err != nil {
		t.Fatal(err)
	}
	if p, _ := e.entries.Get(e.ctx, content.ProfileUID, "u2", super); p.Data["name"] != "Ani" || p.Data["jobTitle"] != "Teknisi" {
		t.Errorf("profil lama: %+v", p.Data)
	}
}
