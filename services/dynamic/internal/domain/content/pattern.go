package content

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"eam/services/common/errs"
)

// AutoPattern: kode dibuat dari pola, mis. "AST-{seq:5}", "{classification.classificationCode}-{YYYY}-{seq:4}".
// Token: {seq} / {seq:N} nomor urut (N digit, nol di depan), {YYYY} {YY} {MM} {DD}, {field}, {relasi.field}.
// Nomor urut terpisah per hasil pola selain {seq} (tiap kode classification punya urutan sendiri).
const AutoPattern = "pattern"

var patternToken = regexp.MustCompile(`\{([^{}]*)\}`)

func (ct *ContentType) validatePattern(f Field) error {
	p := f.AutoFill.Pattern
	if f.Type != TText {
		return errs.Invalid("field %q: kode otomatis hanya untuk field text", f.Name)
	}
	if strings.TrimSpace(p) == "" {
		return errs.Invalid("field %q: isi pola kode", f.Name)
	}
	seqs := 0
	for _, m := range patternToken.FindAllStringSubmatch(p, -1) {
		tok := m[1]
		switch {
		case tok == "seq" || strings.HasPrefix(tok, "seq:"):
			if n, err := strconv.Atoi(strings.TrimPrefix(tok, "seq:")); tok != "seq" && (err != nil || n < 1 || n > 12) {
				return errs.Invalid("field %q: {seq:N} butuh N 1-12", f.Name)
			}
			seqs++
		case tok == "YYYY" || tok == "YY" || tok == "MM" || tok == "DD":
		default:
			name, sub, nested := strings.Cut(tok, ".")
			i := slices.IndexFunc(ct.Fields, func(x Field) bool { return x.Name == name })
			if i < 0 || name == f.Name {
				return errs.Invalid("field %q: token {%s} tidak dikenal", f.Name, tok)
			}
			if nested && (ct.Fields[i].Type != TRelation || ct.Fields[i].Multiple || sub == "") {
				return errs.Invalid("field %q: {%s} harus dari field relasi tunggal", f.Name, tok)
			}
		}
	}
	if seqs > 1 {
		return errs.Invalid("field %q: {seq} hanya boleh sekali", f.Name)
	}
	return nil
}

// ApplyPatterns mengisi field kode berpola (input tidak dimutasi). Terkunci: kode lama dipertahankan, nilai user diabaikan;
// tidak terkunci: hanya dibuat bila kosong. next memberi nomor urut berikutnya untuk sebuah kunci counter.
// ponytail: nomor urut diambil sebelum simpan, jadi simpan yang gagal meninggalkan nomor loncat (seperti sequence DB).
func (ct *ContentType) ApplyPatterns(data, old map[string]any, src AutoFillSources, next func(key string) (int64, error)) (map[string]any, error) {
	out := maps.Clone(data)
	if out == nil {
		out = map[string]any{}
	}
	for _, f := range ct.Fields {
		af := f.AutoFill
		if af == nil || af.Source != AutoPattern {
			continue
		}
		if af.Lock {
			out[f.Name] = old[f.Name]
		}
		if !isEmpty(out[f.Name]) {
			continue
		}
		var seq func() (string, error)
		render := func(withSeq bool) (string, error) {
			var err error
			s := patternToken.ReplaceAllStringFunc(af.Pattern, func(m string) string {
				tok := m[1 : len(m)-1]
				switch {
				case tok == "seq" || strings.HasPrefix(tok, "seq:"):
					if !withSeq {
						return "{seq}"
					}
					var n string
					n, err = seq()
					if w, e := strconv.Atoi(strings.TrimPrefix(tok, "seq:")); e == nil && len(n) < w {
						n = strings.Repeat("0", w-len(n)) + n
					}
					return n
				case tok == "YYYY":
					return src.Now.Format("2006")
				case tok == "YY":
					return src.Now.Format("06")
				case tok == "MM":
					return src.Now.Format("01")
				case tok == "DD":
					return src.Now.Format("02")
				}
				name, sub, nested := strings.Cut(tok, ".")
				v := out[name]
				if nested {
					v = nil
					if id, _ := out[name].(string); id != "" && src.Related != nil {
						v = src.Related(name, id)[sub]
					}
				}
				if v == nil {
					return ""
				}
				return fmt.Sprint(v)
			})
			return s, err
		}
		scope, _ := render(false)
		seq = func() (string, error) {
			n, err := next(ct.UID + "/" + f.Name + "/" + scope)
			return strconv.FormatInt(n, 10), err
		}
		v, err := render(true)
		if err != nil {
			return nil, err
		}
		out[f.Name] = v
	}
	return out, nil
}
