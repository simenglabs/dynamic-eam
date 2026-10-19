package content

import (
	"slices"
	"time"
	"unicode/utf8"

	"eam/services/common/errs"
)

// Refs: id relation & media yang dirujuk entri; keberadaannya dicek app layer ke repository.
type Refs struct {
	Relations map[string][]string // target uid -> ids
	Media     []string
	Domains   map[string][]string // kode domain -> valueId
}

// MetaFields: metadata entri yang dikelola sistem (bukan input user).
var MetaFields = reservedFields

// ValidateEntry memvalidasi & menormalkan data sesuai fields. Key asing ditolak.
// creating = isi default untuk field yang tidak dikirim.
func ValidateEntry(fields []Field, data map[string]any, comps map[string]Component, creating bool) (map[string]any, Refs, error) {
	r := &refsCollector{comps: comps, refs: Refs{Relations: map[string][]string{}, Domains: map[string][]string{}}}
	out, err := r.validate(fields, data, creating)
	return out, r.refs, err
}

type refsCollector struct {
	comps map[string]Component
	refs  Refs
}

func (r *refsCollector) validate(fields []Field, data map[string]any, creating bool) (map[string]any, error) {
	for k := range data {
		if !slices.ContainsFunc(fields, func(f Field) bool { return f.Name == k }) {
			return nil, errs.Invalid("field %q tidak dikenal", k)
		}
	}
	out := map[string]any{}
	for _, f := range fields {
		v, ok := data[f.Name]
		if (!ok || v == nil) && creating && f.Default != nil {
			v = f.Default
		}
		if isEmpty(v) {
			if f.Required {
				return nil, errs.Invalid("%s wajib diisi", f.Label)
			}
			continue
		}
		cv, err := coerce(f, v, r)
		if err != nil {
			return nil, errs.Invalid("%s: %v", f.Label, err)
		}
		out[f.Name] = cv
	}
	return out, nil
}

func isEmpty(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return x == ""
	case []any:
		return len(x) == 0
	case map[string]any:
		return len(x) == 0
	}
	return false
}

var textMax = map[string]int{TText: 255, TLongText: 100_000}

// coerce mengecek tipe satu nilai. r == nil = validasi nilai default (relation/media/component tidak didukung).
func coerce(f Field, v any, r *refsCollector) (any, error) {
	switch f.Type {
	case TText, TLongText:
		s, ok := v.(string)
		if !ok {
			return nil, errs.Invalid("harus teks")
		}
		if utf8.RuneCountInString(s) > textMax[f.Type] {
			return nil, errs.Invalid("maksimal %d karakter", textMax[f.Type])
		}
		return s, nil
	case TNumber:
		switch n := v.(type) {
		case float64:
			return n, nil
		case int:
			return float64(n), nil
		case int32:
			return float64(n), nil
		case int64:
			return float64(n), nil
		}
		return nil, errs.Invalid("harus angka")
	case TBoolean:
		if b, ok := v.(bool); ok {
			return b, nil
		}
		return nil, errs.Invalid("harus true/false")
	case TDate:
		s, ok := v.(string)
		if _, err := time.Parse(time.DateOnly, s); !ok || err != nil {
			return nil, errs.Invalid("format tanggal harus YYYY-MM-DD")
		}
		return s, nil
	case TSelect:
		if s, ok := v.(string); ok && slices.Contains(f.Options, s) {
			return s, nil
		}
		return nil, errs.Invalid("harus salah satu dari %v", f.Options)
	}

	if r == nil {
		return nil, errs.Invalid("default tidak didukung untuk tipe %s", f.Type)
	}
	switch f.Type {
	case TAttributes: // isi dicek app layer terhadap atribut classification (ValidateAttributes)
		m, ok := v.(map[string]any)
		if !ok {
			return nil, errs.Invalid("harus objek atribut")
		}
		return m, nil
	case TDomain:
		s, ok := v.(string)
		if !ok {
			return nil, errs.Invalid("harus kode nilai domain")
		}
		r.refs.Domains[f.Domain] = append(r.refs.Domains[f.Domain], s)
		return s, nil
	case TRelation, TMedia:
		ids, err := idList(v, f.Multiple)
		if err != nil {
			return nil, err
		}
		if f.Type == TRelation {
			r.refs.Relations[f.Target] = append(r.refs.Relations[f.Target], ids...)
		} else {
			r.refs.Media = append(r.refs.Media, ids...)
		}
		if f.Multiple {
			return ids, nil
		}
		return ids[0], nil
	case TComponent:
		comp := r.comps[f.Component]
		one := func(x any) (map[string]any, error) {
			m, ok := x.(map[string]any)
			if !ok {
				return nil, errs.Invalid("harus objek")
			}
			return r.validate(comp.Fields, m, true)
		}
		if !f.Repeatable {
			return one(v)
		}
		list, ok := v.([]any)
		if !ok {
			return nil, errs.Invalid("harus list")
		}
		out := make([]any, 0, len(list))
		seen := map[any]int{}
		for i, x := range list {
			m, err := one(x)
			if err != nil {
				return nil, errs.Invalid("item %d: %v", i+1, err)
			}
			if k := m[f.UniqueBy]; f.UniqueBy != "" && k != nil {
				if j, dup := seen[k]; dup {
					return nil, errs.Invalid("item %d dan %d: %s %v duplikat", j+1, i+1, f.UniqueBy, k)
				}
				seen[k] = i
			}
			out = append(out, m)
		}
		return out, nil
	}
	return nil, errs.Invalid("tipe %q tidak dikenal", f.Type)
}

func idList(v any, multiple bool) ([]string, error) {
	if s, ok := v.(string); ok && !multiple {
		return []string{s}, nil
	}
	list, ok := v.([]any)
	if !ok || !multiple {
		return nil, errs.Invalid("format id tidak valid")
	}
	ids := make([]string, 0, len(list))
	for _, x := range list {
		s, ok := x.(string)
		if !ok || s == "" {
			return nil, errs.Invalid("format id tidak valid")
		}
		if !slices.Contains(ids, s) {
			ids = append(ids, s)
		}
	}
	return ids, nil
}
