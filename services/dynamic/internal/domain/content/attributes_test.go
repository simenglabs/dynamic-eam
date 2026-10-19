package content

import (
	"strings"
	"testing"

	"eam/services/common/errs"
)

func TestAttributesFieldValidation(t *testing.T) {
	cat := catalog()
	cat.ContentTypes[ClassificationUID] = true
	ct := vehicle()
	ct.Fields = append(ct.Fields,
		Field{Name: "classification", Label: "Classification", Type: TRelation, Target: ClassificationUID},
		Field{Name: "specs", Label: "Attributes", Type: TAttributes, From: "classification"})
	if err := ct.Validate(cat); err != nil {
		t.Fatal(err)
	}
	for _, from := range []string{"", "location", "nope", "specs"} {
		ct.Fields[len(ct.Fields)-1].From = from
		if errs.KindOf(ct.Validate(cat)) != errs.KindInvalid {
			t.Errorf("from %q must be rejected (must be a relation to %s)", from, ClassificationUID)
		}
	}
}

func TestValidateAttributes(t *testing.T) {
	defs := []AttributeDef{
		{ID: "a1", Code: "TEGANGAN", Label: "Voltage", DataType: AttrNumeric, Mandatory: true},
		{ID: "a2", Code: "MERK", Label: "Brand", DataType: AttrALN, Length: 5},
		{ID: "a3", Code: "TGL", Label: "Install Date", DataType: AttrDate},
		{ID: "a4", Code: "GIS", Label: "GIS", DataType: AttrBoolean},
		{ID: "a5", Code: "KONDISI", Label: "Condition", DataType: AttrALN, Domain: "HEALTH_INDEX"},
	}
	domainOK := func(code, v string) bool { return code == "HEALTH_INDEX" && v == "GOOD" }
	got, err := ValidateAttributes(defs, map[string]any{"a1": 20.0, "a2": "ABB", "a3": "2026-01-02", "a4": true, "a5": "GOOD", "old": "dropped"}, domainOK)
	if err != nil || len(got) != 5 || got["a1"] != 20.0 {
		t.Fatalf("valid: %v %v", got, err)
	}
	if _, ok := got["old"]; ok {
		t.Error("attributes outside the classification are dropped (e.g. classification changed)")
	}
	bad := map[string]map[string]any{
		"required":   {"a2": "ABB"},
		"number":     {"a1": "twenty"},
		"length":     {"a1": 1.0, "a2": "VERY LONG TEXT"},
		"date":       {"a1": 1.0, "a3": "02/01/2026"},
		"boolean":    {"a1": 1.0, "a4": "yes"},
		"domain":     {"a1": 1.0, "a5": "BAD"},
		"not object": nil,
	}
	for name, v := range bad {
		var in any = v
		if name == "not object" {
			in = "x"
		}
		if _, err := ValidateAttributes(defs, in, domainOK); errs.KindOf(err) != errs.KindInvalid {
			t.Errorf("%s must be rejected: %v", name, err)
		} else if name == "required" && !strings.Contains(err.Error(), "Voltage") {
			t.Errorf("required message must name the attribute: %v", err)
		}
	}
	if got, err := ValidateAttributes(nil, nil, domainOK); err != nil || len(got) != 0 {
		t.Errorf("no classification: %v %v", got, err)
	}
}
