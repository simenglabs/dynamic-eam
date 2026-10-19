package content

import (
	"regexp"
	"slices"

	"eam/services/common/errs"
)

// Layout form entri: urutan section -> baris -> kolom. Diatur superadmin di builder.
type Layout struct {
	Sections []Section
}

type Section struct {
	ID    string
	Title string
	Rows  []Row
}

// Row: Columns = jumlah kolom grid (1 baris berapa field); Fields berurutan kiri ke kanan, "" = slot kosong.
type Row struct {
	Columns int
	Fields  []string
}

const MaxColumns = 4

var sectionIDRe = regexp.MustCompile(`^[a-z][a-z0-9-]{0,47}$`)

func (l *Layout) validate(fields []Field) error {
	ids, placed := map[string]bool{}, map[string]bool{}
	for _, s := range l.Sections {
		if !sectionIDRe.MatchString(s.ID) {
			return errs.Invalid("ID section %q tidak valid", s.ID)
		}
		if ids[s.ID] {
			return errs.Invalid("ID section %q duplikat", s.ID)
		}
		ids[s.ID] = true
		if s.Title == "" {
			return errs.Invalid("judul section wajib diisi")
		}
		for _, r := range s.Rows {
			if r.Columns < 1 || r.Columns > MaxColumns {
				return errs.Invalid("section %s: jumlah kolom harus 1-%d", s.Title, MaxColumns)
			}
			if len(r.Fields) > r.Columns {
				return errs.Invalid("section %s: field dalam satu baris melebihi jumlah kolom", s.Title)
			}
			for _, name := range r.Fields {
				if name == "" {
					continue
				}
				if !slices.ContainsFunc(fields, func(f Field) bool { return f.Name == name }) {
					return errs.Invalid("layout merujuk field %q yang tidak ada", name)
				}
				if placed[name] {
					return errs.Invalid("field %q ditempatkan lebih dari sekali", name)
				}
				placed[name] = true
			}
		}
	}
	return nil
}

const defaultColumns = 2

// EffectiveLayout: layout untuk render form. Tanpa layout = satu section "Umum" 2 kolom.
// Field yang belum ditempatkan ditambahkan ke section terakhir; rujukan ke field yang sudah dihapus jadi slot kosong.
func (ct *ContentType) EffectiveLayout() Layout {
	var l Layout
	if ct.Layout != nil {
		l = Layout{Sections: make([]Section, len(ct.Layout.Sections))}
		for i, s := range ct.Layout.Sections {
			rows := make([]Row, len(s.Rows))
			for j, r := range s.Rows {
				rows[j] = Row{Columns: r.Columns, Fields: slices.Clone(r.Fields)}
			}
			l.Sections[i] = Section{ID: s.ID, Title: s.Title, Rows: rows}
		}
	}
	exists := func(n string) bool { return slices.ContainsFunc(ct.Fields, func(f Field) bool { return f.Name == n }) }
	placed := map[string]bool{}
	for _, s := range l.Sections {
		for _, r := range s.Rows {
			for k, n := range r.Fields {
				if n != "" && !exists(n) {
					r.Fields[k] = ""
				}
				placed[r.Fields[k]] = true
			}
		}
	}
	var rest []string
	for _, f := range ct.Fields {
		if !placed[f.Name] {
			rest = append(rest, f.Name)
		}
	}
	if len(rest) == 0 {
		return l
	}
	if len(l.Sections) == 0 {
		l.Sections = []Section{{ID: "umum", Title: "Umum"}}
	}
	last := &l.Sections[len(l.Sections)-1]
	for chunk := range slices.Chunk(rest, defaultColumns) {
		last.Rows = append(last.Rows, Row{Columns: defaultColumns, Fields: chunk})
	}
	return l
}
