package content

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"eam/services/common/errs"
)

// DisplayField: entry label (title, relation options). Set manually: a field name ("number")
// or a template with placeholders ("{number} - {name}").
var placeholderRe = regexp.MustCompile(`\{([a-zA-Z][a-zA-Z0-9]*)\}`)

func (ct *ContentType) displayRefs() []string {
	if !strings.ContainsAny(ct.DisplayField, "{}") {
		return []string{strings.TrimSpace(ct.DisplayField)}
	}
	var out []string
	for _, m := range placeholderRe.FindAllStringSubmatch(ct.DisplayField, -1) {
		out = append(out, m[1])
	}
	return out
}

func (ct *ContentType) validateDisplay() error {
	d := strings.TrimSpace(ct.DisplayField)
	if d == "" {
		return errs.Invalid("display field is required")
	}
	if strings.ContainsAny(placeholderRe.ReplaceAllString(d, ""), "{}") {
		return errs.Invalid("display field: unbalanced curly braces, e.g. {number} - {name}")
	}
	refs := ct.displayRefs()
	if len(refs) == 0 {
		return errs.Invalid("display field: at least one {field} is required")
	}
	for _, name := range refs {
		i := slices.IndexFunc(ct.Fields, func(f Field) bool { return f.Name == name })
		if i < 0 {
			return errs.Invalid("display field references a field %q that does not exist", name)
		}
		if t := ct.Fields[i].Type; t == TComponent || t == TMedia {
			return errs.Invalid("display field cannot use field %s (type %s)", name, t)
		}
	}
	return nil
}

// DisplayLabel: entry label from the record; empty fields become empty strings.
func (ct *ContentType) DisplayLabel(data map[string]any) string {
	val := func(name string) string {
		if v, ok := data[name]; ok && v != nil {
			return fmt.Sprint(v)
		}
		return ""
	}
	if !strings.ContainsAny(ct.DisplayField, "{}") {
		return val(strings.TrimSpace(ct.DisplayField))
	}
	return placeholderRe.ReplaceAllStringFunc(ct.DisplayField, func(m string) string { return val(m[1 : len(m)-1]) })
}

// DisplaySortField: first field of the display, used to sort relation options.
func (ct *ContentType) DisplaySortField() string {
	if refs := ct.displayRefs(); len(refs) > 0 {
		return refs[0]
	}
	return ""
}
