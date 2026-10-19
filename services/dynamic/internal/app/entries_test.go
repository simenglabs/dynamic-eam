package app

import (
	"strings"
	"testing"

	"eam/services/common/errs"
)

func TestCreateEntry(t *testing.T) {
	e := newEnv()
	loc := e.seed()
	m, _ := e.media.Upload(e.ctx, "a.png", 8, strings.NewReader("\x89PNG\r\n\x1a\n"), "u1")
	got, err := e.entries.Create(e.ctx, "vehicle", map[string]any{"plate": "L 1", "location": loc, "photos": []any{m.ID}}, u1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Data["status"] != "Aktif" || got.CreatedBy != "u1" || !got.CreatedAt.Equal(t0) {
		t.Errorf("entri: %+v", got)
	}
	page, _ := e.entries.List(e.ctx, "vehicle", Query{Q: "l 1", Page: 1, Size: 10}, u1)
	if page.Total != 1 {
		t.Errorf("entri harus ter-index & bisa dicari: %+v", page)
	}
}

func TestCreateEntryChecksReferences(t *testing.T) {
	e := newEnv()
	e.seed()
	cases := map[string]map[string]any{
		"relation palsu": {"plate": "L 2", "location": "palsu"},
		"media palsu":    {"plate": "L 3", "photos": []any{"palsu"}},
		"select salah":   {"plate": "L 4", "status": "Hilang"},
	}
	for name, data := range cases {
		if _, err := e.entries.Create(e.ctx, "vehicle", data, u1); errs.KindOf(err) != errs.KindInvalid {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := e.entries.Create(e.ctx, "nope", map[string]any{}, u1); errs.KindOf(err) != errs.KindNotFound {
		t.Errorf("content type tidak ada: %v", err)
	}
}

func TestUniqueConflict(t *testing.T) {
	e := newEnv()
	e.seed()
	if _, err := e.entries.Create(e.ctx, "location", map[string]any{"name": "Gudang"}, u1); errs.KindOf(err) != errs.KindConflict {
		t.Errorf("nama unik duplikat: %v", err)
	}
}

func TestUpdateEntryKeepsCreatedMeta(t *testing.T) {
	e := newEnv()
	loc := e.seed()
	got, err := e.entries.Update(e.ctx, "location", loc, map[string]any{"name": "Gudang Baru"}, u2)
	if err != nil {
		t.Fatal(err)
	}
	if got.CreatedBy != "u1" || got.UpdatedBy != "u2" || got.Data["name"] != "Gudang Baru" {
		t.Errorf("update: %+v", got)
	}
	if _, err := e.entries.Update(e.ctx, "location", "nope", map[string]any{"name": "x"}, u2); errs.KindOf(err) != errs.KindNotFound {
		t.Errorf("update tidak ada: %v", err)
	}
}

func TestEntryIndexFailureIsUnavailable(t *testing.T) {
	e := newEnv()
	e.seed()
	e.index.Fail = true
	_, err := e.entries.Create(e.ctx, "location", map[string]any{"name": "Gudang 2"}, u1)
	if errs.KindOf(err) != errs.KindUnavailable {
		t.Errorf("err = %v", err)
	}
}

func TestDeleteEntry(t *testing.T) {
	e := newEnv()
	loc := e.seed()
	if err := e.entries.Delete(e.ctx, "location", loc, u1); err != nil {
		t.Fatal(err)
	}
	if _, err := e.entries.Get(e.ctx, "location", loc, u1); errs.KindOf(err) != errs.KindNotFound {
		t.Errorf("harus terhapus: %v", err)
	}
	if p, _ := e.entries.List(e.ctx, "location", Query{Page: 1, Size: 10}, u1); p.Total != 0 {
		t.Error("harus terhapus dari index")
	}
}

func TestOptionsUseDisplayField(t *testing.T) {
	e := newEnv()
	loc := e.seed()
	opts, err := e.entries.Options(e.ctx, "location", "", nil, u1)
	if err != nil || len(opts) != 1 || opts[0].ID != loc || opts[0].Label != "Gudang" {
		t.Errorf("options: %+v %v", opts, err)
	}
}

func TestAllForExport(t *testing.T) {
	e := newEnv()
	e.seed()
	ct, all, err := e.entries.All(e.ctx, "location", u1)
	if err != nil || ct.UID != "location" || len(all) != 1 {
		t.Errorf("all: %v %d", err, len(all))
	}
}

func TestOptionsUseDisplayTemplate(t *testing.T) {
	e := newEnv()
	loc := e.seed()
	ct, _ := e.schemas.GetContentType(e.ctx, "location")
	ct.DisplayField = "Lokasi: {name}"
	if _, err := e.schemas.UpdateContentType(e.ctx, "location", ct); err != nil {
		t.Fatal(err)
	}
	opts, _ := e.entries.Options(e.ctx, "location", "", nil, u1)
	if len(opts) != 1 || opts[0].ID != loc || opts[0].Label != "Lokasi: Gudang" {
		t.Errorf("options: %+v", opts)
	}
}
