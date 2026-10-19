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

// AutoPattern: code generated from a pattern, e.g. "AST-{seq:5}", "{classification.classificationCode}-{YYYY}-{seq:4}".
// Tokens: {seq} / {seq:N} sequence number (N digits, zero-padded), {YYYY} {YY} {MM} {DD}, {field}, {relation.field}.
// Sequence numbers are separate per pattern result excluding {seq} (each classification code has its own sequence).
const AutoPattern = "pattern"

var patternToken = regexp.MustCompile(`\{([^{}]*)\}`)

func (ct *ContentType) validatePattern(f Field) error {
	p := f.AutoFill.Pattern
	if f.Type != TText {
		return errs.Invalid("field %q: auto code is only for text fields", f.Name)
	}
	if strings.TrimSpace(p) == "" {
		return errs.Invalid("field %q: code pattern is required", f.Name)
	}
	seqs := 0
	for _, m := range patternToken.FindAllStringSubmatch(p, -1) {
		tok := m[1]
		switch {
		case tok == "seq" || strings.HasPrefix(tok, "seq:"):
			if n, err := strconv.Atoi(strings.TrimPrefix(tok, "seq:")); tok != "seq" && (err != nil || n < 1 || n > 12) {
				return errs.Invalid("field %q: {seq:N} requires N between 1 and 12", f.Name)
			}
			seqs++
		case tok == "YYYY" || tok == "YY" || tok == "MM" || tok == "DD":
		default:
			name, sub, nested := strings.Cut(tok, ".")
			i := slices.IndexFunc(ct.Fields, func(x Field) bool { return x.Name == name })
			if i < 0 || name == f.Name {
				return errs.Invalid("field %q: unknown token {%s}", f.Name, tok)
			}
			if nested && (ct.Fields[i].Type != TRelation || ct.Fields[i].Multiple || sub == "") {
				return errs.Invalid("field %q: {%s} must come from a single relation field", f.Name, tok)
			}
		}
	}
	if seqs > 1 {
		return errs.Invalid("field %q: {seq} may only appear once", f.Name)
	}
	return nil
}

// ApplyPatterns fills pattern-based code fields (input is not mutated). Locked: the old code is kept, user values are ignored;
// not locked: generated only when empty. next returns the next sequence number for a counter key.
// ponytail: the sequence number is taken before saving, so a failed save leaves a gap (like a DB sequence).
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
