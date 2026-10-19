package content

import (
	"testing"

	"eam/services/common/errs"
)

func approval() *Workflow {
	return &Workflow{
		States: []State{
			{Key: "draft", Label: "Draft", Color: "neutral", Editable: true},
			{Key: "submitted", Label: "Awaiting Approval", Color: "yellow"},
			{Key: "approved", Label: "Approved", Color: "green"},
			{Key: "rejected", Label: "Rejected", Color: "red", Editable: true},
		},
		Initial: "draft",
		Transitions: []Transition{
			{Key: "submit", Label: "Submit", From: []string{"draft", "rejected"}, To: "submitted"},
			{Key: "approve", Label: "Approve", From: []string{"submitted"}, To: "approved"},
			{Key: "reject", Label: "Reject", From: []string{"submitted"}, To: "rejected"},
		},
		Views: []View{{Key: "approval", Label: "Vehicle Approval", Parent: "assets", States: []string{"submitted"}}},
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
		"no status":                func(w *Workflow) { w.States = nil },
		"invalid status key":       func(w *Workflow) { w.States[0].Key = "New Draft" },
		"duplicate status key":     func(w *Workflow) { w.States[1].Key = "draft" },
		"empty status label":       func(w *Workflow) { w.States[0].Label = "" },
		"unknown color":            func(w *Workflow) { w.States[0].Color = "purple" },
		"unknown initial status":   func(w *Workflow) { w.Initial = "nope" },
		"empty transition from":    func(w *Workflow) { w.Transitions[0].From = nil },
		"unknown transition from":  func(w *Workflow) { w.Transitions[0].From = []string{"nope"} },
		"unknown transition to":    func(w *Workflow) { w.Transitions[0].To = "nope" },
		"duplicate transition key": func(w *Workflow) { w.Transitions[1].Key = "submit" },
		"empty transition label":   func(w *Workflow) { w.Transitions[0].Label = "" },
		"view without parent":      func(w *Workflow) { w.Views[0].Parent = "" },
		"view with unknown status": func(w *Workflow) { w.Views[0].States = []string{"nope"} },
		"view without status":      func(w *Workflow) { w.Views[0].States = nil },
		"invalid view key":         func(w *Workflow) { w.Views[0].Key = "Bad Key" },
	}
	for name, mutate := range cases {
		ct := vehicle()
		ct.Workflow = approval()
		mutate(ct.Workflow)
		if errs.KindOf(ct.Validate(catalog())) != errs.KindInvalid {
			t.Errorf("%s: must be invalid", name)
		}
	}
}

func TestInitialStatus(t *testing.T) {
	ct := vehicle()
	if ct.InitialStatus() != "" {
		t.Error("no workflow = no status")
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
		t.Fatalf("submit from rejected: %+v %v", tr, err)
	}
	if _, err := w.Transition("approve", "draft"); errs.KindOf(err) != errs.KindInvalid {
		t.Errorf("approve from draft: %v", err)
	}
	if _, err := w.Transition("nope", "draft"); errs.KindOf(err) != errs.KindNotFound {
		t.Errorf("missing transition: %v", err)
	}
	if got := w.Available("submitted"); len(got) != 2 || got[0].Key != "approve" {
		t.Errorf("Available(submitted) = %+v", got)
	}
}

func TestEditable(t *testing.T) {
	ct := vehicle()
	if !ct.Editable("anything") {
		t.Error("without workflow it is always editable")
	}
	ct.Workflow = approval()
	if !ct.Editable("draft") || ct.Editable("approved") || !ct.Editable("old-removed-status") {
		t.Error("editable follows the state; unknown status stays editable")
	}
}

func TestWorkflowPermKeys(t *testing.T) {
	if TransitionPermKey("vehicle", "approve") != "ct/vehicle/t/approve" || ViewPermKey("vehicle", "approval") != "ct/vehicle/v/approval" {
		t.Error("workflow permission key format changed")
	}
}

// A transition can require input (e.g. Submit requires uploading an official report + choosing the attachment type).
func TestTransitionFieldsValidation(t *testing.T) {
	ct := vehicle()
	ct.Workflow = approval()
	ct.Workflow.Transitions[0].Fields = []Field{
		{Name: "jenis", Label: "Attachment Type", Type: TSelect, Options: []string{"BA", "Photo"}, Required: true},
		{Name: "file", Label: "File", Type: TMedia, Required: true},
		{Name: "lampiran", Label: "Attachment", Type: TComponent, Component: "address", Repeatable: true},
	}
	if err := ct.Validate(catalog()); err != nil {
		t.Fatal(err)
	}
	for name, f := range map[string]Field{
		"unknown type":           {Name: "x", Label: "X", Type: "magic"},
		"select without options": {Name: "x", Label: "X", Type: TSelect},
		"unknown component":      {Name: "x", Label: "X", Type: TComponent, Component: "nope"},
	} {
		ct.Workflow.Transitions[1].Fields = []Field{f}
		if errs.KindOf(ct.Validate(catalog())) != errs.KindInvalid {
			t.Errorf("%s must be rejected", name)
		}
	}
}
