package content

import (
	"testing"

	"eam/services/common/errs"
)

func withWhere(w ...Condition) ContentType {
	ct := vehicle() // location = relation to location
	ct.Fields[4].Where = w
	return ct
}

func TestWhereValidation(t *testing.T) {
	for _, w := range [][]Condition{{{"site", "S1"}}, {{"site", "{status}"}, {"workflowStatus", "APPROVED"}}} {
		ct := withWhere(w...)
		if err := ct.Validate(catalog()); err != nil {
			t.Errorf("%v: %v", w, err)
		}
	}
	for name, w := range map[string][]Condition{
		"empty field":         {{"", "x"}},
		"empty value":         {{"site", ""}},
		"missing ref":         {{"site", "{nope}"}},
		"self ref":            {{"site", "{location}"}},
		"invalid field name!": {{"si te", "x"}},
	} {
		ct := withWhere(w...)
		if errs.KindOf(ct.Validate(catalog())) != errs.KindInvalid {
			t.Errorf("%s must be rejected", name)
		}
	}
	ct := vehicle()
	ct.Fields[0].Where = []Condition{{"a", "b"}} // text
	if errs.KindOf(ct.Validate(catalog())) != errs.KindInvalid {
		t.Error("where is only for relations")
	}
}

func TestWhereFiltersAndMatches(t *testing.T) {
	f := Field{Name: "loc", Type: TRelation, Where: []Condition{{"site", "{site}"}, {"kind", "GUDANG"}, {"workflowStatus", "APPROVED"}}}
	got := f.Filters(map[string]any{"site": "S1"})
	if len(got) != 3 || got[0].Field != "site" || got[0].Values[0] != "S1" || got[1].Values[0] != "GUDANG" {
		t.Fatalf("filters: %+v", got)
	}
	if got := f.Filters(map[string]any{}); len(got) != 2 {
		t.Errorf("empty ref = condition ignored: %+v", got)
	}
	e := &Entry{Data: map[string]any{"site": "S1", "kind": "GUDANG"}, Status: "APPROVED"}
	if !Matches(e, f.Filters(map[string]any{"site": "S1"})) {
		t.Error("must match")
	}
	if Matches(e, f.Filters(map[string]any{"site": "S2"})) {
		t.Error("different site")
	}
	if !Matches(&Entry{Data: map[string]any{"tags": []any{"A", "B"}}}, []Scope{{"tags", []string{"B"}}}) {
		t.Error("list field: matches if any element matches")
	}
	if !Matches(&Entry{Data: map[string]any{"active": true}}, []Scope{{"active", []string{"true"}}}) {
		t.Error("boolean compared as text")
	}
}
