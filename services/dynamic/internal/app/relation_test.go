package app

import (
	"testing"

	"eam/services/common/errs"
	"eam/services/dynamic/internal/domain/content"
)

// Relasi dengan filter: lokasi dipilih sesuai site di form (dropdown bertingkat), dicek juga saat Save.
func TestRelationWhere(t *testing.T) {
	e := newEnv()
	loc := &content.ContentType{UID: "location", Label: "Lokasi", Parent: "master", DisplayField: "name", Fields: []content.Field{
		{Name: "name", Label: "Nama", Type: content.TText}, {Name: "site", Label: "Site", Type: content.TText}, {Name: "kind", Label: "Jenis", Type: content.TText}}}
	pole := &content.ContentType{UID: "pole", Label: "Tiang", Parent: "assets", DisplayField: "code", Fields: []content.Field{
		{Name: "code", Label: "Kode", Type: content.TText}, {Name: "site", Label: "Site", Type: content.TText},
		{Name: "location", Label: "Lokasi", Type: content.TRelation, Target: "location", Where: []content.Condition{{Field: "site", Value: "{site}"}, {Field: "kind", Value: "GARDU"}}}}}
	if _, err := e.schemas.Ensure(e.ctx, nil, []*content.ContentType{loc, pole}); err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{}
	for _, l := range []struct{ name, site, kind string }{{"G1", "S1", "GARDU"}, {"G2", "S2", "GARDU"}, {"K1", "S1", "KANTOR"}} {
		x, _ := e.entries.Create(e.ctx, "location", map[string]any{"name": l.name, "site": l.site, "kind": l.kind}, u1)
		ids[l.name] = x.ID
	}
	opts, err := e.entries.Options(e.ctx, "location", "", map[string]string{"site": "S1", "kind": "GARDU"}, u1)
	if err != nil || len(opts) != 1 || opts[0].ID != ids["G1"] {
		t.Fatalf("options terfilter: %+v %v", opts, err)
	}
	if _, err := e.entries.Create(e.ctx, "pole", map[string]any{"code": "P1", "site": "S1", "location": ids["G1"]}, u1); err != nil {
		t.Errorf("sesuai filter: %v", err)
	}
	for _, bad := range []string{"G2", "K1"} {
		if _, err := e.entries.Create(e.ctx, "pole", map[string]any{"code": "P2", "site": "S1", "location": ids[bad]}, u1); errs.KindOf(err) != errs.KindInvalid {
			t.Errorf("%s di luar filter harus ditolak: %v", bad, err)
		}
	}
}

// Atribut classification: field yang muncul sesuai classification (termasuk warisan induk), divalidasi saat Save.
func TestClassificationAttributes(t *testing.T) {
	e := newEnv()
	attr := &content.ContentType{UID: content.AttributeUID, Label: "Attributes", Parent: "master", DisplayField: "attributeId", Fields: []content.Field{
		{Name: "attributeId", Label: "Attribute", Type: content.TText}, {Name: "description", Label: "Desc", Type: content.TText},
		{Name: "dataType", Label: "Tipe", Type: content.TText}, {Name: "unit", Label: "Satuan", Type: content.TText},
		{Name: "length", Label: "Panjang", Type: content.TNumber}}}
	comp := &content.Component{UID: "classification-attribute", Label: "CA", Fields: []content.Field{
		{Name: "attribute", Label: "Attribute", Type: content.TRelation, Target: content.AttributeUID},
		{Name: "mandatory", Label: "Wajib", Type: content.TBoolean}, {Name: "sortOrder", Label: "Urutan", Type: content.TNumber}}}
	cls := &content.ContentType{UID: content.ClassificationUID, Label: "Classification", Parent: "master", DisplayField: "code", Fields: []content.Field{
		{Name: "code", Label: "Kode", Type: content.TText},
		{Name: "parent", Label: "Induk", Type: content.TRelation, Target: content.ClassificationUID},
		{Name: "attributes", Label: "Atribut", Type: content.TComponent, Component: "classification-attribute", Repeatable: true}}}
	asset := &content.ContentType{UID: "asset", Label: "Asset", Parent: "assets", DisplayField: "code", Fields: []content.Field{
		{Name: "code", Label: "Kode", Type: content.TText},
		{Name: "classification", Label: "Classification", Type: content.TRelation, Target: content.ClassificationUID},
		{Name: "specs", Label: "Spesifikasi", Type: content.TAttributes, From: "classification"}}}
	if _, err := e.schemas.Ensure(e.ctx, []*content.Component{comp}, []*content.ContentType{attr, cls, asset}); err != nil {
		t.Fatal(err)
	}
	mk := func(uid string, d map[string]any) string {
		x, err := e.entries.Create(e.ctx, uid, d, u1)
		if err != nil {
			t.Fatal(err)
		}
		return x.ID
	}
	volt := mk(content.AttributeUID, map[string]any{"attributeId": "TEGANGAN", "description": "Tegangan", "dataType": "NUMERIC", "unit": "kV"})
	merk := mk(content.AttributeUID, map[string]any{"attributeId": "MERK", "description": "Merk", "dataType": "ALN", "length": 5.0})
	tinggi := mk(content.AttributeUID, map[string]any{"attributeId": "TINGGI", "description": "Tinggi", "dataType": "NUMERIC", "unit": "m"})
	root := mk(content.ClassificationUID, map[string]any{"code": "TGD", "attributes": []any{map[string]any{"attribute": merk, "sortOrder": 9.0}}})
	trafo := mk(content.ClassificationUID, map[string]any{"code": "TRF", "parent": root, "attributes": []any{map[string]any{"attribute": volt, "mandatory": true, "sortOrder": 1.0}}})
	tiang := mk(content.ClassificationUID, map[string]any{"code": "TNG", "attributes": []any{map[string]any{"attribute": tinggi}}})

	defs, err := e.entries.AttributesOf(e.ctx, trafo)
	if err != nil || len(defs) != 2 || defs[0].Code != "TEGANGAN" || !defs[0].Mandatory || defs[0].Unit != "kV" || defs[1].Code != "MERK" || defs[1].Length != 5 {
		t.Fatalf("atribut trafo (+ induk): %+v %v", defs, err)
	}
	if _, err := e.entries.Create(e.ctx, "asset", map[string]any{"code": "A1", "classification": trafo, "specs": map[string]any{merk: "ABB"}}, u1); errs.KindOf(err) != errs.KindInvalid {
		t.Errorf("atribut wajib kosong harus ditolak: %v", err)
	}
	a, err := e.entries.Create(e.ctx, "asset", map[string]any{"code": "A1", "classification": trafo, "specs": map[string]any{volt: 20.0, merk: "ABB"}}, u1)
	if err != nil || a.Data["specs"].(map[string]any)[volt] != 20.0 {
		t.Fatalf("create: %+v %v", a, err)
	}
	// classification diganti: atribut lama dibuang
	a, err = e.entries.Update(e.ctx, "asset", a.ID, map[string]any{"code": "A1", "classification": tiang, "specs": map[string]any{volt: 20.0, tinggi: 9.0}}, u1)
	if err != nil || len(a.Data["specs"].(map[string]any)) != 1 {
		t.Errorf("ganti classification: %+v %v", a, err)
	}
	if _, err := e.entries.AttributesOf(e.ctx, "nope"); errs.KindOf(err) != errs.KindNotFound {
		t.Errorf("classification tidak ada: %v", err)
	}
}
