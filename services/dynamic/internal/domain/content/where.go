package content

import (
	"fmt"
	"slices"
	"strings"

	"eam/services/common/errs"
)

// Condition: relation option filter — field on the target entry = Value.
// Literal Value ("GUDANG") or "{field}" = value of another field in this form (cascading dropdown, e.g. location per site).
// Field "workflowStatus" = workflow status of the target entry.
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
		return errs.Invalid("field %q: filter (where) is only for relations", f.Name)
	}
	for _, c := range f.Where {
		if !fieldRe.MatchString(c.Field) && c.Field != StatusField {
			return errs.Invalid("field %q: filter field %q is invalid", f.Name, c.Field)
		}
		if strings.TrimSpace(c.Value) == "" {
			return errs.Invalid("field %q: filter value %s is required", f.Name, c.Field)
		}
		if name, ok := c.ref(); ok && (inComponent || name == f.Name || !slices.ContainsFunc(siblings, func(x Field) bool { return x.Name == name })) {
			return errs.Invalid("field %q: filter references field {%s} that is not in the form", f.Name, name)
		}
	}
	return nil
}

// StatusField: filter field name for the target entry's workflow status.
const StatusField = "workflowStatus"

// Filters: where conditions with values filled from the form data. An empty ref = the condition is ignored.
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

// Matches: the target entry satisfies all filters (a list field matches if any element matches).
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
