package content

import (
	"testing"

	"eam/services/common/errs"
)

func withDomainField(f Field) ContentType {
	ct := vehicle()
	ct.Fields = append(ct.Fields, f)
	return ct
}

func TestDomainFieldSchema(t *testing.T) {
	ok := withDomainField(Field{Name: "priority", Label: "Priority", Type: TDomain, Domain: "PRIORITY", Required: true})
	if err := ok.Validate(catalog()); err != nil {
		t.Fatal(err)
	}
	bad := withDomainField(Field{Name: "priority", Label: "Priority", Type: TDomain})
	if errs.KindOf(bad.Validate(catalog())) != errs.KindInvalid {
		t.Error("domain field without a domain code must be invalid")
	}
	def := withDomainField(Field{Name: "priority", Label: "Priority", Type: TDomain, Domain: "PRIORITY", Default: "HIGH"})
	if errs.KindOf(def.Validate(catalog())) != errs.KindInvalid {
		t.Error("static default is not supported for domain fields (use isDefault in the domain)")
	}
}

func TestDomainValueCollectedAsRef(t *testing.T) {
	ct := withDomainField(Field{Name: "priority", Label: "Priority", Type: TDomain, Domain: "PRIORITY"})
	out, refs, err := ValidateEntry(ct.Fields, map[string]any{"plate": "L 1", "priority": "HIGH"}, catalog().Components, true)
	if err != nil {
		t.Fatal(err)
	}
	if out["priority"] != "HIGH" || len(refs.Domains["PRIORITY"]) != 1 || refs.Domains["PRIORITY"][0] != "HIGH" {
		t.Errorf("out=%v refs=%+v", out, refs)
	}
	if _, _, err := ValidateEntry(ct.Fields, map[string]any{"plate": "L 1", "priority": 3}, nil, true); errs.KindOf(err) != errs.KindInvalid {
		t.Error("domain value must be text (valueId)")
	}
}

func TestUniqueByInRepeatableComponent(t *testing.T) {
	cat := catalog()
	cat.Components["domain-value"] = Component{UID: "domain-value", Label: "Value", Fields: []Field{
		{Name: "valueId", Label: "Value ID", Type: TText, Required: true},
		{Name: "valueDescription", Label: "Description", Type: TText},
	}}
	domain := ContentType{UID: "master-domain", Label: "Domain", Parent: "master", DisplayField: "domainCode", Fields: []Field{
		{Name: "domainCode", Label: "Domain", Type: TText, Required: true, Unique: true},
		{Name: "values", Label: "Values", Type: TComponent, Component: "domain-value", Repeatable: true, UniqueBy: "valueId"},
	}}
	if err := domain.Validate(cat); err != nil {
		t.Fatal(err)
	}
	dup := map[string]any{"domainCode": "PRIORITY", "values": []any{map[string]any{"valueId": "LOW"}, map[string]any{"valueId": "LOW"}}}
	if _, _, err := ValidateEntry(domain.Fields, dup, cat.Components, true); errs.KindOf(err) != errs.KindInvalid {
		t.Error("duplicate valueId within one domain must be rejected")
	}
	okData := map[string]any{"domainCode": "PRIORITY", "values": []any{map[string]any{"valueId": "LOW"}, map[string]any{"valueId": "HIGH"}}}
	if _, _, err := ValidateEntry(domain.Fields, okData, cat.Components, true); err != nil {
		t.Error(err)
	}

	for name, f := range map[string]Field{
		"uniqueBy on non-repeatable": {Name: "v", Label: "V", Type: TComponent, Component: "domain-value", UniqueBy: "valueId"},
		"uniqueBy missing field":     {Name: "v", Label: "V", Type: TComponent, Component: "domain-value", Repeatable: true, UniqueBy: "nope"},
	} {
		ct := domain
		ct.Fields = []Field{domain.Fields[0], f}
		if errs.KindOf(ct.Validate(cat)) != errs.KindInvalid {
			t.Errorf("%s: must be invalid", name)
		}
	}
}
