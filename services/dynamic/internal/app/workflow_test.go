package app

import (
	"strings"
	"testing"

	"eam/services/common/errs"
	"eam/services/dynamic/internal/domain/content"
)

func withWorkflow(e *env) {
	ct := vehicleCT()
	ct.Workflow = &content.Workflow{
		States: []content.State{
			{Key: "draft", Label: "Draft", Color: "neutral", Editable: true},
			{Key: "submitted", Label: "Submitted", Color: "yellow"},
			{Key: "approved", Label: "Approved", Color: "green"},
		},
		Initial: "draft",
		Transitions: []content.Transition{
			{Key: "submit", Label: "Submit", From: []string{"draft"}, To: "submitted"},
			{Key: "approve", Label: "Approve", From: []string{"submitted"}, To: "approved"},
		},
		Views: []content.View{{Key: "approval", Label: "Approval", Parent: "assets", States: []string{"submitted"}}},
	}
	if _, err := e.schemas.UpdateContentType(e.ctx, "vehicle", ct); err != nil {
		panic(err)
	}
}

func TestCreateSetsInitialStatus(t *testing.T) {
	e := newEnv()
	e.seed()
	withWorkflow(e)
	v, err := e.entries.Create(e.ctx, "vehicle", map[string]any{"plate": "L 1"}, u1)
	if err != nil || v.Status != "draft" {
		t.Fatalf("status awal: %+v %v", v, err)
	}
}

func TestTransitionFlow(t *testing.T) {
	e := newEnv()
	e.seed()
	withWorkflow(e)
	v, _ := e.entries.Create(e.ctx, "vehicle", map[string]any{"plate": "L 1"}, u1)

	if _, err := e.entries.Transition(e.ctx, "vehicle", v.ID, "approve", u2, "", nil); errs.KindOf(err) != errs.KindInvalid {
		t.Errorf("approve dari draft: %v", err)
	}
	v, err := e.entries.Transition(e.ctx, "vehicle", v.ID, "submit", u1, "mohon dicek", nil)
	if err != nil || v.Status != "submitted" {
		t.Fatalf("submit: %+v %v", v, err)
	}
	if len(v.History) != 1 || v.History[0].From != "draft" || v.History[0].To != "submitted" || v.History[0].By != "u1" || v.History[0].Note != "mohon dicek" {
		t.Errorf("riwayat: %+v", v.History)
	}

	// Status submitted tidak editable: update & delete ditolak.
	if _, err := e.entries.Update(e.ctx, "vehicle", v.ID, map[string]any{"plate": "L 2"}, u1); errs.KindOf(err) != errs.KindForbidden {
		t.Errorf("update saat submitted: %v", err)
	}
	if err := e.entries.Delete(e.ctx, "vehicle", v.ID, u1); errs.KindOf(err) != errs.KindForbidden {
		t.Errorf("delete saat submitted: %v", err)
	}

	// View approval hanya berisi status submitted.
	e.entries.Create(e.ctx, "vehicle", map[string]any{"plate": "L 9"}, u1)
	page, _ := e.entries.List(e.ctx, "vehicle", Query{Page: 1, Size: 10, States: []string{"submitted"}}, u1)
	if page.Total != 1 || page.Items[0].ID != v.ID {
		t.Errorf("filter status: %+v", page)
	}

	v, _ = e.entries.Transition(e.ctx, "vehicle", v.ID, "approve", u2, "", nil)
	if v.Status != "approved" || len(v.History) != 2 {
		t.Errorf("approve: %+v", v)
	}
}

func TestUpdateKeepsStatusAndHistory(t *testing.T) {
	e := newEnv()
	e.seed()
	withWorkflow(e)
	v, _ := e.entries.Create(e.ctx, "vehicle", map[string]any{"plate": "L 1"}, u1)
	v, err := e.entries.Update(e.ctx, "vehicle", v.ID, map[string]any{"plate": "L 1A"}, u1)
	if err != nil || v.Status != "draft" {
		t.Errorf("update tidak boleh mengubah status: %+v %v", v, err)
	}
}

func TestTransitionWithoutWorkflow(t *testing.T) {
	e := newEnv()
	loc := e.seed()
	if _, err := e.entries.Transition(e.ctx, "location", loc, "submit", u1, "", nil); errs.KindOf(err) != errs.KindNotFound {
		t.Errorf("content type tanpa workflow: %v", err)
	}
}

func TestTransitionRequiresInputs(t *testing.T) {
	e := newEnv()
	e.seed()
	ct := vehicleCT()
	ct.Workflow = &content.Workflow{
		States:  []content.State{{Key: "draft", Label: "Draft", Color: "neutral", Editable: true}, {Key: "submitted", Label: "Submitted", Color: "yellow"}},
		Initial: "draft",
		Transitions: []content.Transition{{Key: "submit", Label: "Submit", From: []string{"draft"}, To: "submitted", Fields: []content.Field{
			{Name: "jenis", Label: "Jenis Lampiran", Type: content.TSelect, Options: []string{"BA", "Foto"}, Required: true},
			{Name: "file", Label: "File", Type: content.TMedia, Required: true},
			{Name: "ket", Label: "Keterangan", Type: content.TText},
		}}},
	}
	if _, err := e.schemas.UpdateContentType(e.ctx, "vehicle", ct); err != nil {
		t.Fatal(err)
	}
	m, err := e.media.Upload(e.ctx, "ba.pdf", 4, strings.NewReader("%PDF"), u1.ID)
	if err != nil {
		t.Fatal(err)
	}
	v, _ := e.entries.Create(e.ctx, "vehicle", map[string]any{"plate": "L 1"}, u1)
	bad := map[string]map[string]any{
		"tanpa isian": nil,
		"file kosong": {"jenis": "BA"},
		"opsi asing":  {"jenis": "KTP", "file": m.ID},
		"file asing":  {"jenis": "BA", "file": "nope"},
		"field asing": {"jenis": "BA", "file": m.ID, "x": 1},
	}
	for name, data := range bad {
		if _, err := e.entries.Transition(e.ctx, "vehicle", v.ID, "submit", u1, "", data); errs.KindOf(err) != errs.KindInvalid {
			t.Errorf("%s harus ditolak: %v", name, err)
		}
	}
	v, err = e.entries.Transition(e.ctx, "vehicle", v.ID, "submit", u1, "ok", map[string]any{"jenis": "BA", "file": m.ID, "ket": "lengkap"})
	if err != nil || v.Status != "submitted" {
		t.Fatalf("transisi: %+v %v", v, err)
	}
	revs, _ := e.entries.History(e.ctx, "vehicle", v.ID, u1)
	last := revs[len(revs)-1]
	if last.Transition != "submit" || last.Data["jenis"] != "BA" || last.Data["file"] != m.ID || last.Data["ket"] != "lengkap" {
		t.Errorf("isian tercatat di riwayat: %+v", last)
	}
}
