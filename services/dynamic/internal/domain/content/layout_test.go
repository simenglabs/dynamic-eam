package content

import (
	"testing"

	"eam/services/common/errs"
)

func withLayout(l *Layout) ContentType {
	ct := vehicle()
	ct.Layout = l
	return ct
}

func TestValidLayout(t *testing.T) {
	ct := withLayout(&Layout{Sections: []Section{
		{ID: "utama", Title: "Main Record", Rows: []Row{
			{Columns: 3, Fields: []string{"plate", "status", "active"}},
			{Columns: 2, Fields: []string{"since", ""}}, // empty slot is allowed
		}},
		{ID: "relasi", Title: "Relation", Rows: []Row{{Columns: 1, Fields: []string{"location"}}}},
	}})
	if err := ct.Validate(catalog()); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidLayouts(t *testing.T) {
	cases := map[string]*Layout{
		"section without title": {Sections: []Section{{ID: "a", Title: "", Rows: []Row{{Columns: 1, Fields: []string{"plate"}}}}}},
		"duplicate section id":  {Sections: []Section{{ID: "a", Title: "A"}, {ID: "a", Title: "B"}}},
		"invalid section id":    {Sections: []Section{{ID: "Bad ID", Title: "A"}}},
		"zero columns":          {Sections: []Section{{ID: "a", Title: "A", Rows: []Row{{Columns: 0, Fields: []string{"plate"}}}}}},
		"more than 4 columns":   {Sections: []Section{{ID: "a", Title: "A", Rows: []Row{{Columns: 5, Fields: []string{"plate"}}}}}},
		"fields exceed columns": {Sections: []Section{{ID: "a", Title: "A", Rows: []Row{{Columns: 1, Fields: []string{"plate", "status"}}}}}},
		"unknown field":         {Sections: []Section{{ID: "a", Title: "A", Rows: []Row{{Columns: 1, Fields: []string{"nope"}}}}}},
		"field placed twice":    {Sections: []Section{{ID: "a", Title: "A", Rows: []Row{{Columns: 2, Fields: []string{"plate", "plate"}}}}}},
	}
	for name, l := range cases {
		ct := withLayout(l)
		if errs.KindOf(ct.Validate(catalog())) != errs.KindInvalid {
			t.Errorf("%s: must be invalid", name)
		}
	}
}

func TestEffectiveLayoutDefault(t *testing.T) {
	ct := vehicle() // no layout
	l := ct.EffectiveLayout()
	if len(l.Sections) != 1 || l.Sections[0].Title != "General" {
		t.Fatalf("default: %+v", l)
	}
	var placed []string
	for _, r := range l.Sections[0].Rows {
		if r.Columns != 2 {
			t.Errorf("default 2 columns, got %d", r.Columns)
		}
		placed = append(placed, r.Fields...)
	}
	if len(placed) < len(ct.Fields) || placed[0] != "plate" {
		t.Errorf("all fields in order: %v", placed)
	}
}

func TestEffectiveLayoutAppendsUnplacedFields(t *testing.T) {
	ct := withLayout(&Layout{Sections: []Section{
		{ID: "a", Title: "A", Rows: []Row{{Columns: 1, Fields: []string{"status"}}}},
		{ID: "b", Title: "B", Rows: []Row{{Columns: 2, Fields: []string{"plate"}}}},
	}})
	l := ct.EffectiveLayout()
	if l.Sections[0].Rows[0].Fields[0] != "status" {
		t.Error("the configured order must be preserved")
	}
	seen := map[string]int{}
	for _, s := range l.Sections {
		for _, r := range s.Rows {
			for _, f := range r.Fields {
				seen[f]++
			}
		}
	}
	for _, f := range ct.Fields {
		if seen[f.Name] != 1 {
			t.Errorf("field %s appears %d times", f.Name, seen[f.Name])
		}
	}
	last := l.Sections[len(l.Sections)-1]
	if last.ID != "b" || len(last.Rows) < 2 {
		t.Errorf("unplaced fields go to the last section: %+v", last)
	}
}

func TestEffectiveLayoutDropsRemovedFields(t *testing.T) {
	ct := withLayout(&Layout{Sections: []Section{{ID: "a", Title: "A", Rows: []Row{{Columns: 2, Fields: []string{"plate", "removed"}}}}}})
	l := ct.EffectiveLayout() // an old schema may reference removed fields
	if f := l.Sections[0].Rows[0].Fields; f[1] != "" {
		t.Errorf("missing fields become empty slots: %v", f)
	}
}
