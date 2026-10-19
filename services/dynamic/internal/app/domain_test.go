package app

import (
	"testing"

	"eam/services/common/errs"
	"eam/services/dynamic/internal/domain/content"
)

// withDomains: Domain master (seed) + PRIORITY domain + content type "task" with a domain field.
func withDomains(t *testing.T, e *env) {
	t.Helper()
	comp := &content.Component{UID: "domain-value", Label: "Value", Fields: []content.Field{
		{Name: "valueId", Label: "Value ID", Type: content.TText, Required: true},
		{Name: "valueDescription", Label: "Description", Type: content.TText},
		{Name: "sortOrder", Label: "Urutan", Type: content.TNumber},
		{Name: "isDefault", Label: "Default", Type: content.TBoolean},
		{Name: "isActive", Label: "Aktif", Type: content.TBoolean},
	}}
	dom := &content.ContentType{UID: content.DomainUID, Label: "Domain", Parent: "master", DisplayField: "domainCode", Fields: []content.Field{
		{Name: "domainCode", Label: "Domain", Type: content.TText, Required: true, Unique: true},
		{Name: "values", Label: "Values", Type: content.TComponent, Component: "domain-value", Repeatable: true, UniqueBy: "valueId"},
	}}
	task := &content.ContentType{UID: "task", Label: "Task", Parent: "renkon", DisplayField: "name", Fields: []content.Field{
		{Name: "name", Label: "Nama", Type: content.TText, Required: true},
		{Name: "priority", Label: "Priority", Type: content.TDomain, Domain: "PRIORITY", Required: true},
	}}
	if _, err := e.schemas.Ensure(e.ctx, []*content.Component{comp}, []*content.ContentType{dom, task}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.entries.Create(e.ctx, content.DomainUID, map[string]any{"domainCode": "PRIORITY", "values": []any{
		map[string]any{"valueId": "LOW", "valueDescription": "Rendah", "sortOrder": 1.0},
		map[string]any{"valueId": "MED", "valueDescription": "Sedang", "sortOrder": 2.0, "isDefault": true},
		map[string]any{"valueId": "OLD", "valueDescription": "Lama", "isActive": false},
	}}, u1); err != nil {
		t.Fatal(err)
	}
}

func TestDomainFieldAcceptsOnlyActiveValues(t *testing.T) {
	e := newEnv()
	withDomains(t, e)
	if _, err := e.entries.Create(e.ctx, "task", map[string]any{"name": "A", "priority": "LOW"}, u1); err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{"OLD", "NOPE"} {
		if _, err := e.entries.Create(e.ctx, "task", map[string]any{"name": "B", "priority": v}, u1); errs.KindOf(err) != errs.KindInvalid {
			t.Errorf("%s must be rejected: %v", v, err)
		}
	}
}

func TestDomainDefaultOnCreate(t *testing.T) {
	e := newEnv()
	withDomains(t, e)
	got, err := e.entries.Create(e.ctx, "task", map[string]any{"name": "A"}, u1)
	if err != nil || got.Data["priority"] != "MED" {
		t.Errorf("default from domain: %+v %v", got, err)
	}
}

func TestDomainValuesUseCase(t *testing.T) {
	e := newEnv()
	withDomains(t, e)
	vals, err := e.entries.DomainValues(e.ctx, "PRIORITY")
	if err != nil || len(vals) != 2 || vals[0].ID != "LOW" || !vals[1].Default {
		t.Errorf("values: %+v %v", vals, err)
	}
	if _, err := e.entries.DomainValues(e.ctx, "NOPE"); errs.KindOf(err) != errs.KindNotFound {
		t.Errorf("missing domain: %v", err)
	}
	codes, _ := e.entries.DomainCodes(e.ctx)
	if len(codes) != 1 || codes[0].Code != "PRIORITY" {
		t.Errorf("codes: %+v", codes)
	}
}

func TestDomainMissingMasterIsClearError(t *testing.T) {
	e := newEnv() // Domain master does not exist yet
	task := &content.ContentType{UID: "task", Label: "Task", Parent: "renkon", DisplayField: "name", Fields: []content.Field{
		{Name: "name", Label: "Nama", Type: content.TText, Required: true},
		{Name: "priority", Label: "Priority", Type: content.TDomain, Domain: "PRIORITY"},
	}}
	e.schemas.CreateContentType(e.ctx, task)
	if _, err := e.entries.Create(e.ctx, "task", map[string]any{"name": "A", "priority": "LOW"}, u1); errs.KindOf(err) != errs.KindInvalid {
		t.Errorf("domain not created yet: %v", err)
	}
}

func TestEnsureEntriesIdempotent(t *testing.T) {
	e := newEnv()
	withDomains(t, e)
	seed := []map[string]any{
		{"domainCode": "PRIORITY", "values": []any{map[string]any{"valueId": "X"}}}, // already exists: not overwritten
		{"domainCode": "HEALTH_INDEX", "values": []any{map[string]any{"valueId": "BAIK"}}},
	}
	created, err := e.entries.EnsureEntries(e.ctx, content.DomainUID, "domainCode", seed, u1)
	if err != nil || len(created) != 1 || created[0] != "HEALTH_INDEX" {
		t.Fatalf("created=%v err=%v", created, err)
	}
	if vals, _ := e.entries.DomainValues(e.ctx, "PRIORITY"); len(vals) != 2 {
		t.Error("an existing domain must not be overwritten by the seed")
	}
}
