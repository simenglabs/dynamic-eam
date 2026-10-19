package content

import (
	"cmp"
	"slices"
	"strconv"
	"time"
	"unicode/utf8"

	"eam/services/common/errs"
)

// Atribut classification (spesifikasi aset): field bertipe TAttributes menampilkan atribut sesuai classification
// yang dipilih di field relasi From. Nilai = objek {id entri master-attribute: nilai}.
const (
	ClassificationUID     = "master-classification"
	AttributeUID          = "master-attribute"
	ClassAttributesField  = "attributes" // repeatable component di classification: {attribute, mandatory, sortOrder}
	ClassParentField      = "parent"     // atribut induk ikut diwarisi
	AttrALN, AttrNumeric  = "ALN", "NUMERIC"
	AttrDate, AttrBoolean = "DATE", "BOOLEAN"
)

// AttributeDef: satu atribut yang berlaku untuk sebuah classification (sudah digabung dengan induknya).
type AttributeDef struct {
	ID, Code, Label string
	DataType        string
	Unit            string
	Domain          string // kode domain; nilai harus dari domain ini
	Length          int    // ALN: panjang maksimal (0 = 255)
	Mandatory       bool
	Order           float64
}

func validateAttributesField(f Field, siblings []Field, inComponent bool) error {
	i := slices.IndexFunc(siblings, func(x Field) bool { return x.Name == f.From })
	if inComponent || i < 0 || siblings[i].Type != TRelation || siblings[i].Multiple || siblings[i].Target != ClassificationUID {
		return errs.Invalid("field %q: atribut harus mengacu field relasi tunggal ke %s", f.Name, ClassificationUID)
	}
	return nil
}

// ValidateAttributes memvalidasi nilai atribut sesuai definisi; atribut di luar defs dibuang (classification diganti).
func ValidateAttributes(defs []AttributeDef, v any, domainOK func(code, value string) bool) (map[string]any, error) {
	in, ok := v.(map[string]any)
	if v != nil && !ok {
		return nil, errs.Invalid("atribut harus objek")
	}
	out := map[string]any{}
	for _, d := range defs {
		x := in[d.ID]
		if isEmpty(x) {
			if d.Mandatory {
				return nil, errs.Invalid("atribut %s wajib diisi", d.Label)
			}
			continue
		}
		bad := func(msg string) error { return errs.Invalid("atribut %s: %s", d.Label, msg) }
		switch d.DataType {
		case AttrNumeric:
			if _, ok := x.(float64); !ok {
				return nil, bad("harus angka")
			}
		case AttrBoolean:
			if _, ok := x.(bool); !ok {
				return nil, bad("harus true/false")
			}
		case AttrDate:
			if s, ok := x.(string); !ok || func() bool { _, err := time.Parse(time.DateOnly, s); return err != nil }() {
				return nil, bad("format tanggal harus YYYY-MM-DD")
			}
		default: // ALN
			s, ok := x.(string)
			if !ok {
				return nil, bad("harus teks")
			}
			if max := cmp.Or(d.Length, 255); utf8.RuneCountInString(s) > max {
				return nil, bad("maksimal " + strconv.Itoa(max) + " karakter")
			}
		}
		if d.Domain != "" {
			if s, _ := x.(string); !domainOK(d.Domain, s) {
				return nil, bad("nilai tidak ada di domain " + d.Domain)
			}
		}
		out[d.ID] = x
	}
	return out, nil
}
