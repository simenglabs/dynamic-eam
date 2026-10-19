package content

import (
	"testing"

	"eam/services/common/errs"
)

func catalog() Catalog {
	return Catalog{
		ContentTypes: map[string]bool{"location": true},
		Components: map[string]Component{"address": {UID: "address", Label: "Address", Fields: []Field{
			{Name: "street", Label: "Street", Type: TText, Required: true},
			{Name: "zip", Label: "Postal Code", Type: TNumber},
		}}},
	}
}

func vehicle() ContentType {
	return ContentType{UID: "vehicle", Label: "Vehicle", Parent: "assets", DisplayField: "plate", Fields: []Field{
		{Name: "plate", Label: "Plate", Type: TText, Required: true, Unique: true},
		{Name: "status", Label: "Status", Type: TSelect, Options: []string{"Active", "Broken"}, Default: "Active"},
		{Name: "active", Label: "Active", Type: TBoolean},
		{Name: "since", Label: "Since", Type: TDate},
		{Name: "location", Label: "Location", Type: TRelation, Target: "location"},
		{Name: "photos", Label: "Photos", Type: TMedia, Multiple: true},
		{Name: "addresses", Label: "Addresses", Type: TComponent, Component: "address", Repeatable: true},
	}}
}

func TestValidContentType(t *testing.T) {
	ct := vehicle()
	if err := ct.Validate(catalog()); err != nil {
		t.Fatal(err)
	}
}

func TestSelfRelationAllowed(t *testing.T) {
	ct := vehicle()
	ct.Fields = append(ct.Fields, Field{Name: "parent", Label: "Parent", Type: TRelation, Target: "vehicle"})
	if err := ct.Validate(catalog()); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidContentTypes(t *testing.T) {
	cases := map[string]func(*ContentType){
		"invalid uid":             func(c *ContentType) { c.UID = "Bad UID" },
		"empty label":             func(c *ContentType) { c.Label = "" },
		"no parent":               func(c *ContentType) { c.Parent = "" },
		"no fields":               func(c *ContentType) { c.Fields = nil },
		"reserved field name":     func(c *ContentType) { c.Fields[0].Name = "id" },
		"duplicate field name":    func(c *ContentType) { c.Fields[1].Name = "plate" },
		"unknown type":            func(c *ContentType) { c.Fields[2].Type = "json" },
		"select without options":  func(c *ContentType) { c.Fields[1].Options = nil; c.Fields[1].Default = nil },
		"missing relation":        func(c *ContentType) { c.Fields[4].Target = "nope" },
		"missing component":       func(c *ContentType) { c.Fields[6].Component = "nope" },
		"default outside options": func(c *ContentType) { c.Fields[1].Default = "Lost" },
		"unique on boolean":       func(c *ContentType) { c.Fields[2].Unique = true },
		"display field media":     func(c *ContentType) { c.DisplayField = "photos" },
		"unknown display field":   func(c *ContentType) { c.DisplayField = "nope" },
	}
	for name, mutate := range cases {
		ct := vehicle()
		ct.Fields = append([]Field(nil), ct.Fields...)
		mutate(&ct)
		if errs.KindOf(ct.Validate(catalog())) != errs.KindInvalid {
			t.Errorf("%s: must be invalid", name)
		}
	}
}

func TestComponentValidation(t *testing.T) {
	cat := catalog()
	ok := Component{UID: "contact", Label: "Contact", Fields: []Field{{Name: "phone", Label: "Phone", Type: TText}, {Name: "addr", Label: "Address", Type: TComponent, Component: "address"}}}
	if err := ok.Validate(cat); err != nil {
		t.Fatal(err)
	}
	loop := Component{UID: "loop", Label: "Loop", Fields: []Field{{Name: "self", Label: "Self", Type: TComponent, Component: "loop"}}}
	if errs.KindOf(loop.Validate(cat)) != errs.KindInvalid {
		t.Error("cyclic component must be invalid")
	}
	uniq := Component{UID: "u", Label: "U", Fields: []Field{{Name: "code", Label: "Code", Type: TText, Unique: true}}}
	uniq.UID = "uq"
	if errs.KindOf(uniq.Validate(cat)) != errs.KindInvalid {
		t.Error("unique inside a component must be invalid")
	}
}

func TestUniqueAndTextFields(t *testing.T) {
	ct := vehicle()
	if u := ct.UniqueFields(); len(u) != 1 || u[0] != "plate" {
		t.Errorf("UniqueFields = %v", u)
	}
	if !ct.References("location") || ct.References("vehicle") {
		t.Error("References is wrong")
	}
	if !ct.UsesComponent("address") {
		t.Error("UsesComponent is wrong")
	}
}

func TestPermKey(t *testing.T) {
	if PermKey("vehicle") != "ct/vehicle" {
		t.Error(PermKey("vehicle"))
	}
}
