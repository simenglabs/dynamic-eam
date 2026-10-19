package content

import (
	"testing"

	"eam/services/common/errs"
)

func withWhere(w ...Condition) ContentType {
	ct := vehicle() // location = relasi ke location
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
		"field kosong":      {{"", "x"}},
		"nilai kosong":      {{"site", ""}},
		"ref tidak ada":     {{"site", "{nope}"}},
		"ref diri sendiri":  {{"site", "{location}"}},
		"nama field asing!": {{"si te", "x"}},
	} {
		ct := withWhere(w...)
		if errs.KindOf(ct.Validate(catalog())) != errs.KindInvalid {
			t.Errorf("%s harus ditolak", name)
		}
	}
	ct := vehicle()
	ct.Fields[0].Where = []Condition{{"a", "b"}} // text
	if errs.KindOf(ct.Validate(catalog())) != errs.KindInvalid {
		t.Error("where hanya untuk relasi")
	}
}

func TestWhereFiltersAndMatches(t *testing.T) {
	f := Field{Name: "loc", Type: TRelation, Where: []Condition{{"site", "{site}"}, {"kind", "GUDANG"}, {"workflowStatus", "APPROVED"}}}
	got := f.Filters(map[string]any{"site": "S1"})
	if len(got) != 3 || got[0].Field != "site" || got[0].Values[0] != "S1" || got[1].Values[0] != "GUDANG" {
		t.Fatalf("filters: %+v", got)
	}
	if got := f.Filters(map[string]any{}); len(got) != 2 {
		t.Errorf("ref kosong = kondisi diabaikan: %+v", got)
	}
	e := &Entry{Data: map[string]any{"site": "S1", "kind": "GUDANG"}, Status: "APPROVED"}
	if !Matches(e, f.Filters(map[string]any{"site": "S1"})) {
		t.Error("harus cocok")
	}
	if Matches(e, f.Filters(map[string]any{"site": "S2"})) {
		t.Error("site beda")
	}
	if !Matches(&Entry{Data: map[string]any{"tags": []any{"A", "B"}}}, []Scope{{"tags", []string{"B"}}}) {
		t.Error("field list: cocok bila salah satu")
	}
	if !Matches(&Entry{Data: map[string]any{"active": true}}, []Scope{{"active", []string{"true"}}}) {
		t.Error("boolean dibanding sebagai teks")
	}
}
