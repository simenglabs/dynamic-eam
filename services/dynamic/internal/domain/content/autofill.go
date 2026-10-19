package content

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"time"

	"eam/services/common/errs"
)

// Auto fill sources.
const (
	AutoRelation = "relation" // value of Field from the entry selected in the From relation field
	AutoProfile  = "profile"  // value of Field from the user profile (user-profile)
	AutoToday    = "today"    // today's date (YYYY-MM-DD)
	AutoUser     = "user"     // the logged-in user: name (text field) or ID (relation to user profile)
)

// AutoFill: an auto-filled field. Lock = always recomputed by the server (not user-editable);
// without Lock it only fills when empty.
type AutoFill struct {
	Source  string
	From    string // AutoRelation: name of the (single) relation field in this content type
	Field   string // AutoRelation: field on the target entry; AutoProfile: profile field
	Pattern string // AutoPattern: code pattern (see pattern.go)
	Lock    bool
}

// AutoFillSources: external data needed by ApplyAutoFill (prepared by the app layer).
type AutoFillSources struct {
	Related func(relationField, id string) map[string]any // relation target entry data (nil if none)
	Profile map[string]any
	Actor   Actor
	Now     time.Time
}

func (ct *ContentType) validateAutoFill() error {
	for _, f := range ct.Fields {
		af := f.AutoFill
		if af == nil {
			continue
		}
		if f.Type == TComponent || f.Type == TMedia {
			return errs.Invalid("field %q: auto fill is not available for type %s", f.Name, f.Type)
		}
		switch af.Source {
		case AutoRelation:
			i := slices.IndexFunc(ct.Fields, func(x Field) bool { return x.Name == af.From })
			if i < 0 || af.From == f.Name || ct.Fields[i].Type != TRelation || ct.Fields[i].Multiple {
				return errs.Invalid("field %q: auto fill must come from a single relation field", f.Name)
			}
			if af.Field == "" {
				return errs.Invalid("field %q: choose the field to take from %s", f.Name, af.From)
			}
		case AutoProfile:
			if af.Field == "" {
				return errs.Invalid("field %q: choose a profile field", f.Name)
			}
		case AutoPattern:
			if err := ct.validatePattern(f); err != nil {
				return err
			}
		case AutoUser:
			if f.Type == TRelation && f.Target != ProfileUID || f.Type != TRelation && f.Type != TText && f.Type != TLongText {
				return errs.Invalid("field %q: fill from user is only for text or a relation to %s", f.Name, ProfileUID)
			}
		case AutoToday:
		default:
			return errs.Invalid("field %q: unknown auto fill source: %q", f.Name, af.Source)
		}
	}
	return nil
}

// RelatedSources: relation fields used by auto fill (so the app layer only loads the needed targets).
func (ct *ContentType) RelatedSources() []string {
	var out []string
	for _, f := range ct.Fields {
		if f.AutoFill != nil && f.AutoFill.Source == AutoRelation && !slices.Contains(out, f.AutoFill.From) {
			out = append(out, f.AutoFill.From)
		}
	}
	return out
}

// ApplyAutoFill returns a copy of the data with auto fill fields filled in (input is not mutated).
func (ct *ContentType) ApplyAutoFill(data map[string]any, src AutoFillSources) map[string]any {
	out := maps.Clone(data)
	if out == nil {
		out = map[string]any{}
	}
	for _, f := range ct.Fields {
		af := f.AutoFill
		if af == nil || af.Source == AutoPattern || (!af.Lock && !isEmpty(out[f.Name])) {
			continue
		}
		var v any
		switch af.Source {
		case AutoRelation:
			if id, _ := out[af.From].(string); id != "" && src.Related != nil {
				v = src.Related(af.From, id)[af.Field]
			}
		case AutoProfile:
			v = src.Profile[af.Field]
		case AutoToday:
			v = src.Now.Format(time.DateOnly)
		case AutoUser:
			v = src.Actor.Name
			if f.Type == TRelation {
				v = src.Actor.ID
				if f.Multiple {
					v = []any{src.Actor.ID}
				}
			}
		}
		if r := reflect.ValueOf(v); !f.Multiple && r.Kind() == reflect.Slice {
			v = nil // list (e.g. profile "sites") into a single field: take the first
			if r.Len() > 0 {
				v = r.Index(0).Interface()
			}
		}
		if (f.Type == TText || f.Type == TLongText) && v != nil {
			if _, ok := v.(string); !ok {
				v = fmt.Sprint(v)
			}
		}
		if isEmpty(v) {
			delete(out, f.Name) // locked & empty source = left empty
		} else {
			out[f.Name] = v
		}
	}
	return out
}
