package app

import (
	"testing"

	"eam/services/common/errs"
	"eam/services/dynamic/internal/domain/content"
)

func TestEnsureCreatesMissingInDependencyOrder(t *testing.T) {
	e := newEnv()
	addr := &content.Component{UID: "address", Label: "Alamat", Fields: []content.Field{{Name: "street", Label: "Jalan", Type: content.TText}}}
	loc := locationCT()
	veh := vehicleCT() // relation ke location: harus dibuat setelah location walau urutan di seed terbalik
	veh.Fields = append(veh.Fields, content.Field{Name: "addr", Label: "Alamat", Type: content.TComponent, Component: "address"})
	created, err := e.schemas.Ensure(e.ctx, []*content.Component{addr}, []*content.ContentType{veh, loc})
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 3 {
		t.Errorf("created = %v", created)
	}
	if _, err := e.schemas.GetContentType(e.ctx, "vehicle"); err != nil {
		t.Error(err)
	}
}

func TestEnsureSkipsExisting(t *testing.T) {
	e := newEnv()
	e.seed()
	custom, _ := e.schemas.GetContentType(e.ctx, "location")
	custom.Label = "Lokasi (diubah user)"
	e.schemas.UpdateContentType(e.ctx, "location", custom)

	created, err := e.schemas.Ensure(e.ctx, nil, []*content.ContentType{locationCT(), vehicleCT()})
	if err != nil || len(created) != 0 {
		t.Fatalf("created=%v err=%v", created, err)
	}
	if got, _ := e.schemas.GetContentType(e.ctx, "location"); got.Label != "Lokasi (diubah user)" {
		t.Error("seed tidak boleh menimpa perubahan user")
	}
}

func TestEnsureReportsInvalidSeed(t *testing.T) {
	e := newEnv()
	bad := locationCT()
	bad.DisplayField = "{tidakAda}"
	if _, err := e.schemas.Ensure(e.ctx, nil, []*content.ContentType{bad}); errs.KindOf(err) != errs.KindInvalid {
		t.Errorf("seed invalid harus error: %v", err)
	}
}

func TestEnsureUnresolvableRelation(t *testing.T) {
	e := newEnv()
	if _, err := e.schemas.Ensure(e.ctx, nil, []*content.ContentType{vehicleCT()}); errs.KindOf(err) != errs.KindInvalid {
		t.Errorf("relation ke content type yang tidak ada di seed/DB: %v", err)
	}
}

// Constraint DB turunan schema (unique) disamakan saat start, bukan hanya saat schema disimpan.
func TestSyncConstraintsRestoresUnique(t *testing.T) {
	e := newEnv()
	e.seed()                                               // location.name unique, berisi "Gudang"
	e.schemas.d.Entries.SyncUnique(e.ctx, "location", nil) // simulasi constraint lama/hilang
	if err := e.schemas.SyncConstraints(e.ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := e.entries.Create(e.ctx, "location", map[string]any{"name": "Gudang"}, u1); errs.KindOf(err) != errs.KindConflict {
		t.Errorf("unique harus aktif lagi setelah sync: %v", err)
	}
}

// Component boleh merujuk content type di seed yang sama (mis. atribut classification -> master attribute).
func TestEnsureInterleavesComponentAndContentTypeDeps(t *testing.T) {
	e := newEnv()
	attr := &content.ContentType{UID: "attr", Label: "Attribute", Parent: "master", DisplayField: "code",
		Fields: []content.Field{{Name: "code", Label: "Kode", Type: content.TText, Required: true}}}
	row := &content.Component{UID: "cls-attr", Label: "Baris", Fields: []content.Field{
		{Name: "attribute", Label: "Attribute", Type: content.TRelation, Target: "attr", Required: true}}}
	cls := &content.ContentType{UID: "cls", Label: "Classification", Parent: "master", DisplayField: "code", Fields: []content.Field{
		{Name: "code", Label: "Kode", Type: content.TText, Required: true},
		{Name: "attributes", Label: "Attributes", Type: content.TComponent, Component: "cls-attr", Repeatable: true, UniqueBy: "attribute"}}}
	created, err := e.schemas.Ensure(e.ctx, []*content.Component{row}, []*content.ContentType{cls, attr})
	if err != nil || len(created) != 3 {
		t.Fatalf("created=%v err=%v", created, err)
	}
}
