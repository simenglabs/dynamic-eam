// Package content: dynamic menu bounded context — ContentType, Component (schema) and Entry (record).
package content

import (
	"regexp"
	"slices"
	"strings"
	"time"

	"eam/services/common/errs"
)

// Field types (equivalent to Strapi's Content-Type Builder).
const (
	TText       = "text"     // short string, max 255
	TLongText   = "longtext" // long string, max 100k
	TNumber     = "number"
	TBoolean    = "boolean"
	TDate       = "date"       // YYYY-MM-DD
	TSelect     = "select"     // enum, Options
	TRelation   = "relation"   // entry id of the Target content type; Multiple = many
	TMedia      = "media"      // file id; Multiple = many files
	TComponent  = "component"  // object matching Component; Repeatable = list
	TDomain     = "domain"     // valueId from the Domain master (code in Field.Domain); options & default come from the domain
	TAttributes = "attributes" // attributes of the classification in the From relation field (see attributes.go)
)

var fieldTypes = []string{TText, TLongText, TNumber, TBoolean, TDate, TSelect, TRelation, TMedia, TComponent, TDomain, TAttributes}

type Field struct {
	Name       string
	Label      string
	Type       string
	Required   bool
	Unique     bool
	Default    any
	Options    []string
	Target     string
	Multiple   bool
	Component  string
	Repeatable bool
	UniqueBy   string      // repeatable component: component field whose value must be unique within the list
	AutoFill   *AutoFill   // auto fill (relation/profile/today/user)
	Domain     string      // TDomain: domain code (e.g. "PRIORITY")
	Where      []Condition // TRelation: option filter (see where.go)
	From       string      // TAttributes: relation field to classification
}

type ContentType struct {
	UID          string
	Label        string
	Parent       string // zone where the menu appears in the sidebar
	Hidden       bool   // true = hidden from the sidebar (e.g. user profile), permissions still set in Role
	ScopeField   string // relation field to Sites: records restricted to the user's accessible sites (empty = unrestricted)
	DisplayField string
	Fields       []Field
	Layout       *Layout   // nil = default layout
	Workflow     *Workflow // nil = no status
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Component struct {
	UID       string
	Label     string
	Fields    []Field
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Catalog: other existing schemas, for relation/component validation.
type Catalog struct {
	ContentTypes map[string]bool
	Components   map[string]Component
}

// PermKey: RBAC key of a dynamic menu (set in Role Management).
func PermKey(uid string) string { return "ct/" + uid }

var (
	uidRe          = regexp.MustCompile(`^[a-z][a-z0-9-]{1,47}$`)
	fieldRe        = regexp.MustCompile(`^[a-z][a-zA-Z0-9]{0,47}$`)
	reservedFields = []string{"id", "createdAt", "updatedAt", "createdBy", "updatedBy", "workflowStatus", "workflowHistory"}
)

func validateHeader(uid, label string) error {
	if !uidRe.MatchString(uid) {
		return errs.Invalid("UID %q is invalid (lowercase letters, digits, '-', 2-48 characters)", uid)
	}
	if label == "" {
		return errs.Invalid("label is required")
	}
	return nil
}

func (ct *ContentType) Validate(cat Catalog) error {
	if err := validateHeader(ct.UID, ct.Label); err != nil {
		return err
	}
	if ct.Parent == "" {
		return errs.Invalid("parent menu is required")
	}
	types := map[string]bool{ct.UID: true} // self relation is allowed
	for k, v := range cat.ContentTypes {
		types[k] = v
	}
	if err := validateFields(ct.Fields, Catalog{types, cat.Components}, false); err != nil {
		return err
	}
	if err := ct.validateDisplay(); err != nil {
		return err
	}
	if err := ct.validateScope(); err != nil {
		return err
	}
	if err := ct.validateAutoFill(); err != nil {
		return err
	}
	if ct.Layout != nil {
		if err := ct.Layout.validate(ct.Fields); err != nil {
			return err
		}
	}
	if ct.Workflow != nil {
		return ct.Workflow.validate(Catalog{types, cat.Components})
	}
	return nil
}

func (c *Component) Validate(cat Catalog) error {
	if err := validateHeader(c.UID, c.Label); err != nil {
		return err
	}
	comps := map[string]Component{c.UID: *c}
	for k, v := range cat.Components {
		if k != c.UID {
			comps[k] = v
		}
	}
	if err := validateFields(c.Fields, Catalog{cat.ContentTypes, comps}, true); err != nil {
		return err
	}
	// Prevent cycles A -> B -> A (entry validation would recurse forever).
	var visit func(uid string, path []string) error
	visit = func(uid string, path []string) error {
		if slices.Contains(path, uid) {
			return errs.Invalid("cyclic component: %v", append(path, uid))
		}
		for _, f := range comps[uid].Fields {
			if f.Type == TComponent {
				if err := visit(f.Component, append(path, uid)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return visit(c.UID, nil)
}

func validateFields(fields []Field, cat Catalog, inComponent bool) error {
	if len(fields) == 0 {
		return errs.Invalid("at least one field is required")
	}
	seen := map[string]bool{}
	for _, f := range fields {
		if !fieldRe.MatchString(f.Name) || slices.Contains(reservedFields, f.Name) {
			return errs.Invalid("field name %q is invalid (camelCase, not %v)", f.Name, reservedFields)
		}
		if seen[f.Name] {
			return errs.Invalid("duplicate field name %q", f.Name)
		}
		seen[f.Name] = true
		if f.Label == "" {
			return errs.Invalid("field label %q is required", f.Name)
		}
		if !slices.Contains(fieldTypes, f.Type) {
			return errs.Invalid("field %q has unknown type: %q", f.Name, f.Type)
		}
		if f.Unique && (inComponent || (f.Type != TText && f.Type != TNumber)) {
			return errs.Invalid("field %q: unique is only for top-level text/number", f.Name)
		}
		if err := validateWhere(f, fields, inComponent); err != nil {
			return err
		}
		switch f.Type {
		case TAttributes:
			if err := validateAttributesField(f, fields, inComponent); err != nil {
				return err
			}
		case TSelect:
			if len(f.Options) == 0 {
				return errs.Invalid("field %q: select needs at least one option", f.Name)
			}
		case TRelation:
			if !cat.ContentTypes[f.Target] {
				return errs.Invalid("field %q: relation target %q does not exist", f.Name, f.Target)
			}
		case TComponent:
			comp, ok := cat.Components[f.Component]
			if !ok {
				return errs.Invalid("field %q: component %q does not exist", f.Name, f.Component)
			}
			if f.UniqueBy != "" && (!f.Repeatable || !slices.ContainsFunc(comp.Fields, func(x Field) bool { return x.Name == f.UniqueBy })) {
				return errs.Invalid("field %q: unique-by is only for repeatable components and must be a field of the component", f.Name)
			}
		case TDomain:
			if strings.TrimSpace(f.Domain) == "" {
				return errs.Invalid("field %q: domain code is required", f.Name)
			}
		}
		if f.Default != nil {
			if _, err := coerce(f, f.Default, nil); err != nil {
				return errs.Invalid("field %q: default is invalid: %v", f.Name, err)
			}
		}
	}
	return nil
}

func (ct *ContentType) UniqueFields() []string {
	var out []string
	for _, f := range ct.Fields {
		if f.Unique {
			out = append(out, f.Name)
		}
	}
	return out
}

// References: does this content type have a relation to target (other than itself)?
func (ct *ContentType) References(target string) bool {
	return target != ct.UID && slices.ContainsFunc(ct.Fields, func(f Field) bool { return f.Type == TRelation && f.Target == target })
}

func usesComponent(fields []Field, uid string) bool {
	return slices.ContainsFunc(fields, func(f Field) bool { return f.Type == TComponent && f.Component == uid })
}

func (ct *ContentType) UsesComponent(uid string) bool { return usesComponent(ct.Fields, uid) }
func (c *Component) UsesComponent(uid string) bool    { return usesComponent(c.Fields, uid) }
