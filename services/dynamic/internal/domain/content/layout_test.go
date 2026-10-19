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
		{ID: "utama", Title: "Data Utama", Rows: []Row{
			{Columns: 3, Fields: []string{"plate", "status", "active"}},
			{Columns: 2, Fields: []string{"since", ""}}, // slot kosong boleh
		}},
		{ID: "relasi", Title: "Relasi", Rows: []Row{{Columns: 1, Fields: []string{"location"}}}},
	}})
	if err := ct.Validate(catalog()); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidLayouts(t *testing.T) {
	cases := map[string]*Layout{
		"section tanpa judul":    {Sections: []Section{{ID: "a", Title: "", Rows: []Row{{Columns: 1, Fields: []string{"plate"}}}}}},
		"id section duplikat":    {Sections: []Section{{ID: "a", Title: "A"}, {ID: "a", Title: "B"}}},
		"id section salah":       {Sections: []Section{{ID: "Bad ID", Title: "A"}}},
		"kolom nol":              {Sections: []Section{{ID: "a", Title: "A", Rows: []Row{{Columns: 0, Fields: []string{"plate"}}}}}},
		"kolom lebih dari 4":     {Sections: []Section{{ID: "a", Title: "A", Rows: []Row{{Columns: 5, Fields: []string{"plate"}}}}}},
		"field melebihi kolom":   {Sections: []Section{{ID: "a", Title: "A", Rows: []Row{{Columns: 1, Fields: []string{"plate", "status"}}}}}},
		"field tidak dikenal":    {Sections: []Section{{ID: "a", Title: "A", Rows: []Row{{Columns: 1, Fields: []string{"nope"}}}}}},
		"field ditaruh dua kali": {Sections: []Section{{ID: "a", Title: "A", Rows: []Row{{Columns: 2, Fields: []string{"plate", "plate"}}}}}},
	}
	for name, l := range cases {
		ct := withLayout(l)
		if errs.KindOf(ct.Validate(catalog())) != errs.KindInvalid {
			t.Errorf("%s: harus invalid", name)
		}
	}
}

func TestEffectiveLayoutDefault(t *testing.T) {
	ct := vehicle() // tanpa layout
	l := ct.EffectiveLayout()
	if len(l.Sections) != 1 || l.Sections[0].Title != "Umum" {
		t.Fatalf("default: %+v", l)
	}
	var placed []string
	for _, r := range l.Sections[0].Rows {
		if r.Columns != 2 {
			t.Errorf("default 2 kolom, got %d", r.Columns)
		}
		placed = append(placed, r.Fields...)
	}
	if len(placed) < len(ct.Fields) || placed[0] != "plate" {
		t.Errorf("semua field berurutan: %v", placed)
	}
}

func TestEffectiveLayoutAppendsUnplacedFields(t *testing.T) {
	ct := withLayout(&Layout{Sections: []Section{
		{ID: "a", Title: "A", Rows: []Row{{Columns: 1, Fields: []string{"status"}}}},
		{ID: "b", Title: "B", Rows: []Row{{Columns: 2, Fields: []string{"plate"}}}},
	}})
	l := ct.EffectiveLayout()
	if l.Sections[0].Rows[0].Fields[0] != "status" {
		t.Error("urutan yang diatur harus dipertahankan")
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
			t.Errorf("field %s muncul %d kali", f.Name, seen[f.Name])
		}
	}
	last := l.Sections[len(l.Sections)-1]
	if last.ID != "b" || len(last.Rows) < 2 {
		t.Errorf("field yang belum ditempatkan masuk section terakhir: %+v", last)
	}
}

func TestEffectiveLayoutDropsRemovedFields(t *testing.T) {
	ct := withLayout(&Layout{Sections: []Section{{ID: "a", Title: "A", Rows: []Row{{Columns: 2, Fields: []string{"plate", "dihapus"}}}}}})
	l := ct.EffectiveLayout() // schema lama bisa merujuk field yang sudah dihapus
	if f := l.Sections[0].Rows[0].Fields; f[1] != "" {
		t.Errorf("field yang tidak ada jadi slot kosong: %v", f)
	}
}
