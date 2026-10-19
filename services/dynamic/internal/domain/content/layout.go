package content

import (
	"regexp"
	"slices"

	"eam/services/common/errs"
)

// Entry form layout: section -> row -> column order. Set by the superadmin in the builder.
type Layout struct {
	Sections []Section
}

type Section struct {
	ID    string
	Title string
	Rows  []Row
}

// Row: Columns = number of grid columns (fields per row); Fields ordered left to right, "" = empty slot.
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
			return errs.Invalid("section ID %q is invalid", s.ID)
		}
		if ids[s.ID] {
			return errs.Invalid("duplicate section ID %q", s.ID)
		}
		ids[s.ID] = true
		if s.Title == "" {
			return errs.Invalid("section title is required")
		}
		for _, r := range s.Rows {
			if r.Columns < 1 || r.Columns > MaxColumns {
				return errs.Invalid("section %s: number of columns must be 1-%d", s.Title, MaxColumns)
			}
			if len(r.Fields) > r.Columns {
				return errs.Invalid("section %s: fields in a row exceed the number of columns", s.Title)
			}
			for _, name := range r.Fields {
				if name == "" {
					continue
				}
				if !slices.ContainsFunc(fields, func(f Field) bool { return f.Name == name }) {
					return errs.Invalid("layout references a field %q that does not exist", name)
				}
				if placed[name] {
					return errs.Invalid("field %q is placed more than once", name)
				}
				placed[name] = true
			}
		}
	}
	return nil
}

const defaultColumns = 2

// EffectiveLayout: layout used to render the form. No layout = one 2-column "General" section.
// Unplaced fields are appended to the last section; references to removed fields become empty slots.
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
		l.Sections = []Section{{ID: "umum", Title: "General"}}
	}
	last := &l.Sections[len(l.Sections)-1]
	for chunk := range slices.Chunk(rest, defaultColumns) {
		last.Rows = append(last.Rows, Row{Columns: defaultColumns, Fields: chunk})
	}
	return l
}
