package app

import (
	"testing"

	"eam/services/common/errs"
	"eam/services/dynamic/internal/domain/content"
)

func withProfile(t *testing.T, e *env) string {
	t.Helper()
	loc := e.seed() // location "Gudang"
	p := &content.ContentType{UID: content.ProfileUID, Label: "Profil User", Parent: "administration", Hidden: true, DisplayField: "location",
		Fields: []content.Field{{Name: "location", Label: "Lokasi", Type: content.TRelation, Target: "location"}, {Name: "note", Label: "Catatan", Type: content.TText}}}
	if _, err := e.schemas.CreateContentType(e.ctx, p); err != nil {
		t.Fatal(err)
	}
	return loc
}

func TestUpsertCreatesWithGivenIDThenUpdates(t *testing.T) {
	e := newEnv()
	loc := withProfile(t, e)
	got, err := e.entries.Upsert(e.ctx, content.ProfileUID, "USER1", map[string]any{"location": loc}, u1)
	if err != nil || got.ID != "USER1" || got.Version != 1 {
		t.Fatalf("upsert create: %+v %v", got, err)
	}
	got, err = e.entries.Upsert(e.ctx, content.ProfileUID, "USER1", map[string]any{"location": loc, "note": "x"}, u2)
	if err != nil || got.Version != 2 || got.CreatedBy != "u1" || got.UpdatedBy != "u2" {
		t.Fatalf("upsert update: %+v %v", got, err)
	}
	if _, err := e.entries.Upsert(e.ctx, content.ProfileUID, "USER2", map[string]any{"location": "palsu"}, u1); errs.KindOf(err) != errs.KindInvalid {
		t.Errorf("fake relation must still be validated: %v", err)
	}
	hist, _ := e.entries.History(e.ctx, content.ProfileUID, "USER1", u1)
	if len(hist) != 2 || hist[0].Action != "create" || hist[1].Action != "update" {
		t.Errorf("profile history: %+v", hist)
	}
}

func TestUpsertRejectsBadID(t *testing.T) {
	e := newEnv()
	withProfile(t, e)
	for _, id := range []string{"", "../x", "a b"} {
		if _, err := e.entries.Upsert(e.ctx, content.ProfileUID, id, map[string]any{}, u1); errs.KindOf(err) != errs.KindInvalid {
			t.Errorf("id %q must be rejected: %v", id, err)
		}
	}
}

func TestUpsertRevivesSoftDeleted(t *testing.T) {
	e := newEnv()
	withProfile(t, e)
	e.entries.Upsert(e.ctx, content.ProfileUID, "USER1", map[string]any{"note": "a"}, u1)
	e.entries.Delete(e.ctx, content.ProfileUID, "USER1", u1)
	got, err := e.entries.Upsert(e.ctx, content.ProfileUID, "USER1", map[string]any{"note": "b"}, u1)
	if err != nil || got.Deleted || got.Data["note"] != "b" {
		t.Errorf("deleted profile recreated: %+v %v", got, err)
	}
}
