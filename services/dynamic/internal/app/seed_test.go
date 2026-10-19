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
	veh := vehicleCT() // relation to location: must be created after location even though the seed order is reversed
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
		t.Error("seed must not overwrite user changes")
	}
}

func TestEnsureReportsInvalidSeed(t *testing.T) {
	e := newEnv()
	bad := locationCT()
	bad.DisplayField = "{tidakAda}"
	if _, err := e.schemas.Ensure(e.ctx, nil, []*content.ContentType{bad}); errs.KindOf(err) != errs.KindInvalid {
		t.Errorf("invalid seed must error: %v", err)
	}
}

func TestEnsureUnresolvableRelation(t *testing.T) {
	e := newEnv()
	if _, err := e.schemas.Ensure(e.ctx, nil, []*content.ContentType{vehicleCT()}); errs.KindOf(err) != errs.KindInvalid {
		t.Errorf("relation to a content type missing from seed/DB: %v", err)
	}
}

// DB constraints derived from the schema (unique) are synced at startup, not only when the schema is saved.
func TestSyncConstraintsRestoresUnique(t *testing.T) {
	e := newEnv()
	e.seed()                                               // location.name unique, contains "Gudang"
	e.schemas.d.Entries.SyncUnique(e.ctx, "location", nil) // simulate an old/missing constraint
	if err := e.schemas.SyncConstraints(e.ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := e.entries.Create(e.ctx, "location", map[string]any{"name": "Gudang"}, u1); errs.KindOf(err) != errs.KindConflict {
		t.Errorf("unique must be active again after sync: %v", err)
	}
}

// A component may reference a content type in the same seed (e.g. classification attributes -> attribute master).
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
