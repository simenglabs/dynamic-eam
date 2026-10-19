package content

import (
	"bytes"
	"encoding/json"
	"slices"
	"time"
)

// Actions recorded in the entry history.
const (
	ActionCreate     = "create"
	ActionUpdate     = "update"
	ActionDelete     = "delete"
	ActionTransition = "transition"
)

// Actor: the user performing the action (ID for integrity, Name for display) plus their placement
// from the user profile (for per-site record scoping & auto fill).
type Actor struct {
	ID, Name     string
	Super        bool     // superadmin: not restricted by site
	Sites        []string // accessible sites (profile: sites + defaultSite)
	DefaultSite  string
	Organization string
	Profile      map[string]any // user profile data (source of "profile" auto fill)
}

// Change: a change to one field (nil = no value).
type Change struct {
	Field    string
	From, To any
}

// Revision: one audit trail row of an entry. Version = entry version after the action.
type Revision struct {
	EntryID string
	Version int
	Action  string
	By      string
	ByName  string
	At      time.Time
	Changes []Change
	Note    string
	// Workflow transition: transition key + its form input (e.g. attachments).
	Transition string
	Data       map[string]any
}

func NewRevision(e *Entry, action string, by Actor, at time.Time, changes []Change, note string) *Revision {
	return &Revision{EntryID: e.ID, Version: e.Version, Action: action, By: by.ID, ByName: by.Name, At: at, Changes: changes, Note: note}
}

// Diff: per-field changes, ordered by field name. Values are compared via JSON so that
// []string vs []any or int vs float64 with the same content are not treated as changed.
func Diff(before, after map[string]any) []Change {
	keys := map[string]bool{}
	for k := range before {
		keys[k] = true
	}
	for k := range after {
		keys[k] = true
	}
	var out []Change
	for _, k := range slices.Sorted(func(yield func(string) bool) {
		for k := range keys {
			if !yield(k) {
				return
			}
		}
	}) {
		a, b := before[k], after[k]
		ja, _ := json.Marshal(a)
		jb, _ := json.Marshal(b)
		if !bytes.Equal(ja, jb) {
			out = append(out, Change{Field: k, From: a, To: b})
		}
	}
	return out
}
