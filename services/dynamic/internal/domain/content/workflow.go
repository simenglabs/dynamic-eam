package content

import (
	"regexp"
	"slices"
	"time"

	"eam/services/common/errs"
)

// Workflow: entry statuses, initial status after create, transitions between statuses, and views (menus) per status.
type Workflow struct {
	States      []State
	Initial     string
	Transitions []Transition
	Views       []View
}

type State struct {
	Key      string
	Label    string
	Color    string // badge color: neutral|green|red|yellow|blue
	Editable bool   // false = entry is locked (cannot be edited/deleted) in this status
}

// Transition: an action (button) that moves an entry from one of From to To. Permission: TransitionPermKey.
type Transition struct {
	Key   string
	Label string
	From  []string
	To    string
	// Fields: required/optional input when running the transition (e.g. attachment + its type), recorded in history.
	Fields []Field
}

// View: an extra menu listing entries with given statuses (e.g. "Approval"). Permission: ViewPermKey.
type View struct {
	Key    string
	Label  string
	Parent string // zone where the menu appears
	States []string
}

// StatusChange: one row of entry status history.
type StatusChange struct {
	From       string
	To         string
	Transition string
	By         string
	At         time.Time
	Note       string
}

var (
	wfKeyRe = regexp.MustCompile(`^[a-z][a-z0-9-]{0,47}$`)
	colors  = []string{"neutral", "green", "red", "yellow", "blue"}
)

func TransitionPermKey(uid, key string) string { return "ct/" + uid + "/t/" + key }
func ViewPermKey(uid, key string) string       { return "ct/" + uid + "/v/" + key }

func (w *Workflow) validate(cat Catalog) error {
	if len(w.States) == 0 {
		return errs.Invalid("workflow must have at least one status")
	}
	states := map[string]bool{}
	for _, s := range w.States {
		if !wfKeyRe.MatchString(s.Key) {
			return errs.Invalid("status key %q is invalid", s.Key)
		}
		if states[s.Key] {
			return errs.Invalid("duplicate status key %q", s.Key)
		}
		if s.Label == "" {
			return errs.Invalid("status label %q is required", s.Key)
		}
		if !slices.Contains(colors, s.Color) {
			return errs.Invalid("status color %q must be one of %v", s.Key, colors)
		}
		states[s.Key] = true
	}
	if !states[w.Initial] {
		return errs.Invalid("initial status %q is not in the status list", w.Initial)
	}
	known := func(list []string) bool {
		return len(list) > 0 && !slices.ContainsFunc(list, func(s string) bool { return !states[s] })
	}
	seen := map[string]bool{}
	for _, t := range w.Transitions {
		if !wfKeyRe.MatchString(t.Key) || seen[t.Key] {
			return errs.Invalid("transition key %q is invalid or duplicate", t.Key)
		}
		seen[t.Key] = true
		if t.Label == "" {
			return errs.Invalid("transition label %q is required", t.Key)
		}
		if !known(t.From) || !states[t.To] {
			return errs.Invalid("transition %q references a status that does not exist", t.Key)
		}
		if len(t.Fields) > 0 {
			if err := validateFields(t.Fields, cat, false); err != nil {
				return errs.Invalid("transition input %s: %v", t.Label, err)
			}
		}
	}
	seen = map[string]bool{}
	for _, v := range w.Views {
		if !wfKeyRe.MatchString(v.Key) || seen[v.Key] {
			return errs.Invalid("view key %q is invalid or duplicate", v.Key)
		}
		seen[v.Key] = true
		if v.Label == "" || v.Parent == "" {
			return errs.Invalid("view %q needs a label and a parent menu", v.Key)
		}
		if !known(v.States) {
			return errs.Invalid("view %q references a status that does not exist", v.Key)
		}
	}
	return nil
}

// Transition finds the transition key that is valid from the current status.
func (w *Workflow) Transition(key, current string) (*Transition, error) {
	i := slices.IndexFunc(w.Transitions, func(t Transition) bool { return t.Key == key })
	if i < 0 {
		return nil, errs.NotFound("transition")
	}
	t := w.Transitions[i]
	if !slices.Contains(t.From, current) {
		return nil, errs.Invalid("%s cannot be performed from status %s", t.Label, current)
	}
	return &t, nil
}

// Available: transitions possible from the current status (for UI buttons).
func (w *Workflow) Available(current string) []Transition {
	var out []Transition
	for _, t := range w.Transitions {
		if slices.Contains(t.From, current) {
			out = append(out, t)
		}
	}
	return out
}

func (ct *ContentType) InitialStatus() string {
	if ct.Workflow == nil {
		return ""
	}
	return ct.Workflow.Initial
}

// Editable: no workflow / unknown status (e.g. removed from the workflow) = editable.
func (ct *ContentType) Editable(status string) bool {
	if ct.Workflow == nil {
		return true
	}
	i := slices.IndexFunc(ct.Workflow.States, func(s State) bool { return s.Key == status })
	return i < 0 || ct.Workflow.States[i].Editable
}
