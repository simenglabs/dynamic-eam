package content

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"time"

	"eam/services/common/errs"
)

// Sumber isi otomatis.
const (
	AutoRelation = "relation" // nilai Field dari entri yang dipilih di field relasi From
	AutoProfile  = "profile"  // nilai Field dari profil user (user-profile)
	AutoToday    = "today"    // tanggal hari ini (YYYY-MM-DD)
	AutoUser     = "user"     // user yang sedang login: nama (field teks) atau ID (relasi ke profil user)
)

// AutoFill: field diisi otomatis. Lock = selalu dihitung ulang server (tidak bisa diedit user);
// tanpa Lock hanya mengisi bila kosong.
type AutoFill struct {
	Source  string
	From    string // AutoRelation: nama field relasi (tunggal) di content type ini
	Field   string // AutoRelation: field di entri target; AutoProfile: field profil
	Pattern string // AutoPattern: pola kode (lihat pattern.go)
	Lock    bool
}

// AutoFillSources: data luar yang dibutuhkan ApplyAutoFill (disiapkan app layer).
type AutoFillSources struct {
	Related func(relationField, id string) map[string]any // data entri target relasi (nil bila tidak ada)
	Profile map[string]any
	Actor   Actor
	Now     time.Time
}

func (ct *ContentType) validateAutoFill() error {
	for _, f := range ct.Fields {
		af := f.AutoFill
		if af == nil {
			continue
		}
		if f.Type == TComponent || f.Type == TMedia {
			return errs.Invalid("field %q: isi otomatis tidak untuk tipe %s", f.Name, f.Type)
		}
		switch af.Source {
		case AutoRelation:
			i := slices.IndexFunc(ct.Fields, func(x Field) bool { return x.Name == af.From })
			if i < 0 || af.From == f.Name || ct.Fields[i].Type != TRelation || ct.Fields[i].Multiple {
				return errs.Invalid("field %q: isi otomatis harus dari field relasi tunggal", f.Name)
			}
			if af.Field == "" {
				return errs.Invalid("field %q: pilih field yang diambil dari %s", f.Name, af.From)
			}
		case AutoProfile:
			if af.Field == "" {
				return errs.Invalid("field %q: pilih field profil", f.Name)
			}
		case AutoPattern:
			if err := ct.validatePattern(f); err != nil {
				return err
			}
		case AutoUser:
			if f.Type == TRelation && f.Target != ProfileUID || f.Type != TRelation && f.Type != TText && f.Type != TLongText {
				return errs.Invalid("field %q: isi dari user hanya untuk teks atau relasi ke %s", f.Name, ProfileUID)
			}
		case AutoToday:
		default:
			return errs.Invalid("field %q: sumber isi otomatis tidak dikenal: %q", f.Name, af.Source)
		}
	}
	return nil
}

// RelatedSources: field relasi yang dipakai isi otomatis (supaya app layer hanya memuat target yang perlu).
func (ct *ContentType) RelatedSources() []string {
	var out []string
	for _, f := range ct.Fields {
		if f.AutoFill != nil && f.AutoFill.Source == AutoRelation && !slices.Contains(out, f.AutoFill.From) {
			out = append(out, f.AutoFill.From)
		}
	}
	return out
}

// ApplyAutoFill mengembalikan salinan data dengan field isi otomatis terisi (input tidak dimutasi).
func (ct *ContentType) ApplyAutoFill(data map[string]any, src AutoFillSources) map[string]any {
	out := maps.Clone(data)
	if out == nil {
		out = map[string]any{}
	}
	for _, f := range ct.Fields {
		af := f.AutoFill
		if af == nil || af.Source == AutoPattern || (!af.Lock && !isEmpty(out[f.Name])) {
			continue
		}
		var v any
		switch af.Source {
		case AutoRelation:
			if id, _ := out[af.From].(string); id != "" && src.Related != nil {
				v = src.Related(af.From, id)[af.Field]
			}
		case AutoProfile:
			v = src.Profile[af.Field]
		case AutoToday:
			v = src.Now.Format(time.DateOnly)
		case AutoUser:
			v = src.Actor.Name
			if f.Type == TRelation {
				v = src.Actor.ID
				if f.Multiple {
					v = []any{src.Actor.ID}
				}
			}
		}
		if r := reflect.ValueOf(v); !f.Multiple && r.Kind() == reflect.Slice {
			v = nil // list (mis. profil "sites") ke field tunggal: ambil yang pertama
			if r.Len() > 0 {
				v = r.Index(0).Interface()
			}
		}
		if (f.Type == TText || f.Type == TLongText) && v != nil {
			if _, ok := v.(string); !ok {
				v = fmt.Sprint(v)
			}
		}
		if isEmpty(v) {
			delete(out, f.Name) // terkunci & sumber kosong = ikut kosong
		} else {
			out[f.Name] = v
		}
	}
	return out
}
