package app

import (
	"testing"

	"eam/services/common/errs"
)

func TestCreateRecordsMetaAndRevision(t *testing.T) {
	e := newEnv()
	loc := e.seed() // create location by u1
	got, _ := e.entries.Get(e.ctx, "location", loc, u1)
	if got.Version != 1 || got.CreatedBy != "u1" || got.UpdatedBy != "u1" || got.CreatedAt.IsZero() || got.Deleted {
		t.Errorf("meta create: %+v", got)
	}
	hist, err := e.entries.History(e.ctx, "location", loc, u1)
	if err != nil || len(hist) != 1 {
		t.Fatalf("history: %+v %v", hist, err)
	}
	h := hist[0]
	if h.Action != "create" || h.Version != 1 || h.By != "u1" || h.ByName != "Budi" || len(h.Changes) != 1 || h.Changes[0].Field != "name" || h.Changes[0].To != "Gudang" {
		t.Errorf("create revision: %+v", h)
	}
}

func TestUpdateRecordsOnlyChangedFields(t *testing.T) {
	e := newEnv()
	e.seed()
	v, _ := e.entries.Create(e.ctx, "vehicle", map[string]any{"plate": "L 1", "status": "Aktif"}, u1)
	v, err := e.entries.Update(e.ctx, "vehicle", v.ID, map[string]any{"plate": "L 1", "status": "Rusak"}, u2)
	if err != nil || v.Version != 2 || v.UpdatedBy != "u2" || v.CreatedBy != "u1" {
		t.Fatalf("update: %+v %v", v, err)
	}
	hist, _ := e.entries.History(e.ctx, "vehicle", v.ID, u1)
	last := hist[len(hist)-1]
	if last.Action != "update" || last.ByName != "Ani" || len(last.Changes) != 1 || last.Changes[0].From != "Aktif" || last.Changes[0].To != "Rusak" {
		t.Errorf("update revision: %+v", last)
	}
	// Saving without changes: adds neither a version nor a revision.
	v, _ = e.entries.Update(e.ctx, "vehicle", v.ID, map[string]any{"plate": "L 1", "status": "Rusak"}, u2)
	if hist2, _ := e.entries.History(e.ctx, "vehicle", v.ID, u1); v.Version != 2 || len(hist2) != len(hist) {
		t.Errorf("no-op update added a revision: v%d %d revisions", v.Version, len(hist2))
	}
}

func TestSoftDelete(t *testing.T) {
	e := newEnv()
	loc := e.seed()
	if err := e.entries.Delete(e.ctx, "location", loc, u2); err != nil {
		t.Fatal(err)
	}
	if _, err := e.entries.Get(e.ctx, "location", loc, u1); errs.KindOf(err) != errs.KindNotFound {
		t.Errorf("deleted entry must not be openable: %v", err)
	}
	if p, _ := e.entries.List(e.ctx, "location", Query{Page: 1, Size: 10}, u1); p.Total != 0 {
		t.Error("deleted entry must not appear in the list")
	}
	if _, all, _ := e.entries.All(e.ctx, "location", u1); len(all) != 0 {
		t.Error("deleted entry must not be exported")
	}
	if _, err := e.entries.Create(e.ctx, "vehicle", map[string]any{"plate": "L 9", "location": loc}, u1); errs.KindOf(err) != errs.KindInvalid {
		t.Errorf("relation to a deleted entry must be rejected: %v", err)
	}
	if _, err := e.entries.Create(e.ctx, "location", map[string]any{"name": "Gudang"}, u1); err != nil {
		t.Errorf("unique value of a deleted entry may be reused: %v", err)
	}
	hist, _ := e.entries.History(e.ctx, "location", loc, u1)
	if last := hist[len(hist)-1]; last.Action != "delete" || last.By != "u2" || last.Version != 2 {
		t.Errorf("delete revision: %+v", last)
	}
	if err := e.entries.Delete(e.ctx, "location", loc, u2); errs.KindOf(err) != errs.KindNotFound {
		t.Errorf("delete twice: %v", err)
	}
}

func TestTransitionRecordsRevision(t *testing.T) {
	e := newEnv()
	e.seed()
	withWorkflow(e)
	v, _ := e.entries.Create(e.ctx, "vehicle", map[string]any{"plate": "L 1"}, u1)
	v, _ = e.entries.Transition(e.ctx, "vehicle", v.ID, "submit", u2, "cek", nil)
	hist, _ := e.entries.History(e.ctx, "vehicle", v.ID, u1)
	last := hist[len(hist)-1]
	if last.Action != "transition" || last.Note != "cek" || last.Version != v.Version || len(last.Changes) != 1 ||
		last.Changes[0].Field != "workflowStatus" || last.Changes[0].From != "draft" || last.Changes[0].To != "submitted" {
		t.Errorf("transition revision: %+v", last)
	}
}
