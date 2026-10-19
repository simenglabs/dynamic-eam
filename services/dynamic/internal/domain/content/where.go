package content

import (
	"fmt"
	"slices"
	"strings"

	"eam/services/common/errs"
)

// Condition: filter pilihan relasi — field di entri target = Value.
// Value literal ("GUDANG") atau "{field}" = nilai field lain di form ini (dropdown bertingkat, mis. lokasi per site).
// Field "workflowStatus" = status workflow entri target.
type Condition struct {
	Field string
	Value string
}

func (c Condition) ref() (string, bool) {
	if strings.HasPrefix(c.Value, "{") && strings.HasSuffix(c.Value, "}") {
		return c.Value[1 : len(c.Value)-1], true
	}
	return "", false
}

func validateWhere(f Field, siblings []Field, inComponent bool) error {
	if len(f.Where) > 0 && f.Type != TRelation {
		return errs.Invalid("field %q: filter (where) hanya untuk relasi", f.Name)
	}
	for _, c := range f.Where {
		if !fieldRe.MatchString(c.Field) && c.Field != StatusField {
			return errs.Invalid("field %q: field filter %q tidak valid", f.Name, c.Field)
		}
		if strings.TrimSpace(c.Value) == "" {
			return errs.Invalid("field %q: nilai filter %s wajib diisi", f.Name, c.Field)
		}
		if name, ok := c.ref(); ok && (inComponent || name == f.Name || !slices.ContainsFunc(siblings, func(x Field) bool { return x.Name == name })) {
			return errs.Invalid("field %q: filter mengacu field {%s} yang tidak ada di form", f.Name, name)
		}
	}
	return nil
}

// StatusField: nama field filter untuk status workflow entri target.
const StatusField = "workflowStatus"

// Filters: kondisi where dengan nilai terisi dari data form. Ref yang kosong = kondisinya diabaikan.
func (f Field) Filters(data map[string]any) []Scope {
	var out []Scope
	for _, c := range f.Where {
		v := c.Value
		if name, ok := c.ref(); ok {
			if isEmpty(data[name]) {
				continue
			}
			v = fmt.Sprint(data[name])
		}
		out = append(out, Scope{Field: c.Field, Values: []string{v}})
	}
	return out
}

// Matches: entri target memenuhi semua filter (field list cocok bila salah satu elemennya cocok).
func Matches(e *Entry, filters []Scope) bool {
	for _, s := range filters {
		var v any = e.Status
		if s.Field != StatusField {
			v = e.Data[s.Field]
		}
		vals := []any{v}
		if l, ok := v.([]any); ok {
			vals = l
		}
		if !slices.ContainsFunc(vals, func(x any) bool { return x != nil && slices.Contains(s.Values, fmt.Sprint(x)) }) {
			return false
		}
	}
	return true
}
