package content

import (
	"testing"

	"eam/services/common/errs"
)

func approval() *Workflow {
	return &Workflow{
		States: []State{
			{Key: "draft", Label: "Draft", Color: "neutral", Editable: true},
			{Key: "submitted", Label: "Menunggu Approval", Color: "yellow"},
			{Key: "approved", Label: "Disetujui", Color: "green"},
			{Key: "rejected", Label: "Ditolak", Color: "red", Editable: true},
		},
		Initial: "draft",
		Transitions: []Transition{
			{Key: "submit", Label: "Submit", From: []string{"draft", "rejected"}, To: "submitted"},
			{Key: "approve", Label: "Approve", From: []string{"submitted"}, To: "approved"},
			{Key: "reject", Label: "Reject", From: []string{"submitted"}, To: "rejected"},
		},
		Views: []View{{Key: "approval", Label: "Approval Kendaraan", Parent: "assets", States: []string{"submitted"}}},
	}
}

func TestValidWorkflow(t *testing.T) {
	ct := vehicle()
	ct.Workflow = approval()
	if err := ct.Validate(catalog()); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidWorkflows(t *testing.T) {
	cases := map[string]func(*Workflow){
		"tanpa status":          func(w *Workflow) { w.States = nil },
		"key status salah":      func(w *Workflow) { w.States[0].Key = "Draft Baru" },
		"key status duplikat":   func(w *Workflow) { w.States[1].Key = "draft" },
		"label status kosong":   func(w *Workflow) { w.States[0].Label = "" },
		"warna asing":           func(w *Workflow) { w.States[0].Color = "ungu" },
		"status awal asing":     func(w *Workflow) { w.Initial = "nope" },
		"transisi from kosong":  func(w *Workflow) { w.Transitions[0].From = nil },
		"transisi from asing":   func(w *Workflow) { w.Transitions[0].From = []string{"nope"} },
		"transisi to asing":     func(w *Workflow) { w.Transitions[0].To = "nope" },
		"key transisi duplikat": func(w *Workflow) { w.Transitions[1].Key = "submit" },
		"label transisi kosong": func(w *Workflow) { w.Transitions[0].Label = "" },
		"view tanpa induk":      func(w *Workflow) { w.Views[0].Parent = "" },
		"view status asing":     func(w *Workflow) { w.Views[0].States = []string{"nope"} },
		"view tanpa status":     func(w *Workflow) { w.Views[0].States = nil },
		"key view salah":        func(w *Workflow) { w.Views[0].Key = "Bad Key" },
	}
	for name, mutate := range cases {
		ct := vehicle()
		ct.Workflow = approval()
		mutate(ct.Workflow)
		if errs.KindOf(ct.Validate(catalog())) != errs.KindInvalid {
			t.Errorf("%s: harus invalid", name)
		}
	}
}

func TestInitialStatus(t *testing.T) {
	ct := vehicle()
	if ct.InitialStatus() != "" {
		t.Error("tanpa workflow = tanpa status")
	}
	ct.Workflow = approval()
	if ct.InitialStatus() != "draft" {
		t.Error(ct.InitialStatus())
	}
}

func TestTransitionRules(t *testing.T) {
	w := approval()
	tr, err := w.Transition("submit", "rejected")
	if err != nil || tr.To != "submitted" {
		t.Fatalf("submit dari rejected: %+v %v", tr, err)
	}
	if _, err := w.Transition("approve", "draft"); errs.KindOf(err) != errs.KindInvalid {
		t.Errorf("approve dari draft: %v", err)
	}
	if _, err := w.Transition("nope", "draft"); errs.KindOf(err) != errs.KindNotFound {
		t.Errorf("transisi tidak ada: %v", err)
	}
	if got := w.Available("submitted"); len(got) != 2 || got[0].Key != "approve" {
		t.Errorf("Available(submitted) = %+v", got)
	}
}

func TestEditable(t *testing.T) {
	ct := vehicle()
	if !ct.Editable("apa saja") {
		t.Error("tanpa workflow selalu bisa diedit")
	}
	ct.Workflow = approval()
	if !ct.Editable("draft") || ct.Editable("approved") || !ct.Editable("status-lama-yang-dihapus") {
		t.Error("editable mengikuti state; status tak dikenal tetap bisa diedit")
	}
}

func TestWorkflowPermKeys(t *testing.T) {
	if TransitionPermKey("vehicle", "approve") != "ct/vehicle/t/approve" || ViewPermKey("vehicle", "approval") != "ct/vehicle/v/approval" {
		t.Error("format kunci izin workflow berubah")
	}
}

// Transisi bisa minta isian (mis. Submit wajib upload Berita Acara + pilih jenis lampiran).
func TestTransitionFieldsValidation(t *testing.T) {
	ct := vehicle()
	ct.Workflow = approval()
	ct.Workflow.Transitions[0].Fields = []Field{
		{Name: "jenis", Label: "Jenis Lampiran", Type: TSelect, Options: []string{"BA", "Foto"}, Required: true},
		{Name: "file", Label: "File", Type: TMedia, Required: true},
		{Name: "lampiran", Label: "Lampiran", Type: TComponent, Component: "address", Repeatable: true},
	}
	if err := ct.Validate(catalog()); err != nil {
		t.Fatal(err)
	}
	for name, f := range map[string]Field{
		"tipe asing":        {Name: "x", Label: "X", Type: "magic"},
		"select tanpa opsi": {Name: "x", Label: "X", Type: TSelect},
		"component asing":   {Name: "x", Label: "X", Type: TComponent, Component: "nope"},
	} {
		ct.Workflow.Transitions[1].Fields = []Field{f}
		if errs.KindOf(ct.Validate(catalog())) != errs.KindInvalid {
			t.Errorf("%s harus ditolak", name)
		}
	}
}
