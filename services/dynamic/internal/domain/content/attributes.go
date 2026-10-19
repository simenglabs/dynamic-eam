package content

import (
	"cmp"
	"slices"
	"strconv"
	"time"
	"unicode/utf8"

	"eam/services/common/errs"
)

// Classification attributes (asset specs): a TAttributes field shows the attributes of the classification
// selected in the From relation field. Value = object {master-attribute entry id: value}.
const (
	ClassificationUID     = "master-classification"
	AttributeUID          = "master-attribute"
	ClassAttributesField  = "attributes" // repeatable component on classification: {attribute, mandatory, sortOrder}
	ClassParentField      = "parent"     // parent attributes are inherited
	AttrALN, AttrNumeric  = "ALN", "NUMERIC"
	AttrDate, AttrBoolean = "DATE", "BOOLEAN"
)

// AttributeDef: one attribute that applies to a classification (already merged with its parents).
type AttributeDef struct {
	ID, Code, Label string
	DataType        string
	Unit            string
	Domain          string // domain code; the value must come from this domain
	Length          int    // ALN: maximum length (0 = 255)
	Mandatory       bool
	Order           float64
}

func validateAttributesField(f Field, siblings []Field, inComponent bool) error {
	i := slices.IndexFunc(siblings, func(x Field) bool { return x.Name == f.From })
	if inComponent || i < 0 || siblings[i].Type != TRelation || siblings[i].Multiple || siblings[i].Target != ClassificationUID {
		return errs.Invalid("field %q: attributes must reference a single relation field to %s", f.Name, ClassificationUID)
	}
	return nil
}

// ValidateAttributes validates attribute values against their definitions; attributes outside defs are dropped (classification changed).
func ValidateAttributes(defs []AttributeDef, v any, domainOK func(code, value string) bool) (map[string]any, error) {
	in, ok := v.(map[string]any)
	if v != nil && !ok {
		return nil, errs.Invalid("attributes must be an object")
	}
	out := map[string]any{}
	for _, d := range defs {
		x := in[d.ID]
		if isEmpty(x) {
			if d.Mandatory {
				return nil, errs.Invalid("attribute %s is required", d.Label)
			}
			continue
		}
		bad := func(msg string) error { return errs.Invalid("attribute %s: %s", d.Label, msg) }
		switch d.DataType {
		case AttrNumeric:
			if _, ok := x.(float64); !ok {
				return nil, bad("must be a number")
			}
		case AttrBoolean:
			if _, ok := x.(bool); !ok {
				return nil, bad("must be true/false")
			}
		case AttrDate:
			if s, ok := x.(string); !ok || func() bool { _, err := time.Parse(time.DateOnly, s); return err != nil }() {
				return nil, bad("date format must be YYYY-MM-DD")
			}
		default: // ALN
			s, ok := x.(string)
			if !ok {
				return nil, bad("must be text")
			}
			if max := cmp.Or(d.Length, 255); utf8.RuneCountInString(s) > max {
				return nil, bad("maximum " + strconv.Itoa(max) + " characters")
			}
		}
		if d.Domain != "" {
			if s, _ := x.(string); !domainOK(d.Domain, s) {
				return nil, bad("value is not in domain " + d.Domain)
			}
		}
		out[d.ID] = x
	}
	return out, nil
}
