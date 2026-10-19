package content

import (
	"regexp"
	"slices"
	"time"

	"eam/services/common/errs"
)

// Workflow: status entri, status awal setelah create, transisi antar status, dan view (menu) per status.
type Workflow struct {
	States      []State
	Initial     string
	Transitions []Transition
	Views       []View
}

type State struct {
	Key      string
	Label    string
	Color    string // warna badge: neutral|green|red|yellow|blue
	Editable bool   // false = entri terkunci (tidak bisa diubah/dihapus) di status ini
}

// Transition: aksi (tombol) yang memindahkan entri dari salah satu From ke To. Izin: TransitionPermKey.
type Transition struct {
	Key   string
	Label string
	From  []string
	To    string
	// Fields: isian yang wajib/boleh diisi saat menjalankan transisi (mis. lampiran + jenisnya), tercatat di riwayat.
	Fields []Field
}

// View: menu tambahan berisi entri dengan status tertentu (mis. "Approval"). Izin: ViewPermKey.
type View struct {
	Key    string
	Label  string
	Parent string // zone tempat menu muncul
	States []string
}

// StatusChange: satu baris riwayat status entri.
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
		return errs.Invalid("workflow minimal punya satu status")
	}
	states := map[string]bool{}
	for _, s := range w.States {
		if !wfKeyRe.MatchString(s.Key) {
			return errs.Invalid("key status %q tidak valid", s.Key)
		}
		if states[s.Key] {
			return errs.Invalid("key status %q duplikat", s.Key)
		}
		if s.Label == "" {
			return errs.Invalid("label status %q wajib diisi", s.Key)
		}
		if !slices.Contains(colors, s.Color) {
			return errs.Invalid("warna status %q harus salah satu dari %v", s.Key, colors)
		}
		states[s.Key] = true
	}
	if !states[w.Initial] {
		return errs.Invalid("status awal %q tidak ada di daftar status", w.Initial)
	}
	known := func(list []string) bool {
		return len(list) > 0 && !slices.ContainsFunc(list, func(s string) bool { return !states[s] })
	}
	seen := map[string]bool{}
	for _, t := range w.Transitions {
		if !wfKeyRe.MatchString(t.Key) || seen[t.Key] {
			return errs.Invalid("key transisi %q tidak valid atau duplikat", t.Key)
		}
		seen[t.Key] = true
		if t.Label == "" {
			return errs.Invalid("label transisi %q wajib diisi", t.Key)
		}
		if !known(t.From) || !states[t.To] {
			return errs.Invalid("transisi %q merujuk status yang tidak ada", t.Key)
		}
		if len(t.Fields) > 0 {
			if err := validateFields(t.Fields, cat, false); err != nil {
				return errs.Invalid("isian transisi %s: %v", t.Label, err)
			}
		}
	}
	seen = map[string]bool{}
	for _, v := range w.Views {
		if !wfKeyRe.MatchString(v.Key) || seen[v.Key] {
			return errs.Invalid("key view %q tidak valid atau duplikat", v.Key)
		}
		seen[v.Key] = true
		if v.Label == "" || v.Parent == "" {
			return errs.Invalid("view %q butuh label dan menu induk", v.Key)
		}
		if !known(v.States) {
			return errs.Invalid("view %q merujuk status yang tidak ada", v.Key)
		}
	}
	return nil
}

// Transition mencari transisi key yang sah dari status current.
func (w *Workflow) Transition(key, current string) (*Transition, error) {
	i := slices.IndexFunc(w.Transitions, func(t Transition) bool { return t.Key == key })
	if i < 0 {
		return nil, errs.NotFound("transisi")
	}
	t := w.Transitions[i]
	if !slices.Contains(t.From, current) {
		return nil, errs.Invalid("%s tidak bisa dilakukan dari status %s", t.Label, current)
	}
	return &t, nil
}

// Available: transisi yang bisa dilakukan dari status current (untuk tombol di UI).
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

// Editable: tanpa workflow / status tak dikenal (mis. sudah dihapus dari workflow) = boleh diedit.
func (ct *ContentType) Editable(status string) bool {
	if ct.Workflow == nil {
		return true
	}
	i := slices.IndexFunc(ct.Workflow.States, func(s State) bool { return s.Key == status })
	return i < 0 || ct.Workflow.States[i].Editable
}
