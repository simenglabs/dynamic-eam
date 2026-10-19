package content

import (
	"testing"
	"time"

	"eam/services/common/errs"
)

func withCode(pattern string, lock bool) ContentType {
	ct := vehicle()
	ct.Fields = append(ct.Fields, Field{Name: "code", Label: "Code", Type: TText, AutoFill: &AutoFill{Source: AutoPattern, Pattern: pattern, Lock: lock}})
	return ct
}

func TestPatternValidation(t *testing.T) {
	for _, p := range []string{"AST-{seq:5}", "{location.code}-{YYYY}{MM}-{seq}", "{plate}/{YY}/{DD}", "TETAP"} {
		ct := withCode(p, true)
		if err := ct.Validate(catalog()); err != nil {
			t.Errorf("%q: %v", p, err)
		}
	}
	for _, p := range []string{"", "{seq}-{seq}", "{nope}", "{plate.x}", "{photos.x}", "{seq:0}", "{seq:abc}", "{YYYYY}"} {
		ct := withCode(p, true)
		if errs.KindOf(ct.Validate(catalog())) != errs.KindInvalid {
			t.Errorf("%q must be rejected", p)
		}
	}
	ct := vehicle()
	ct.Fields[2].AutoFill = &AutoFill{Source: AutoPattern, Pattern: "X{seq}"} // boolean
	if errs.KindOf(ct.Validate(catalog())) != errs.KindInvalid {
		t.Error("code pattern is only for text fields")
	}
}

type counter map[string]int64

func (c counter) next(key string) (int64, error) { c[key]++; return c[key], nil }

func TestApplyPatterns(t *testing.T) {
	now := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	rel := func(_, id string) map[string]any {
		return map[string]map[string]any{"L1": {"code": "TGD"}, "L2": {"code": "TRF"}}[id]
	}
	src := AutoFillSources{Related: rel, Now: now}
	ct := withCode("{location.code}-{YY}{MM}-{seq:4}", true)
	c := counter{}
	gen := func(data, old map[string]any) any {
		out, err := ct.ApplyPatterns(data, old, src, c.next)
		if err != nil {
			t.Fatal(err)
		}
		return out["code"]
	}
	if v := gen(map[string]any{"location": "L1", "code": "typed"}, nil); v != "TGD-2610-0001" {
		t.Errorf("first = %v", v)
	}
	if v := gen(map[string]any{"location": "L1"}, nil); v != "TGD-2610-0002" {
		t.Errorf("second = %v", v)
	}
	if v := gen(map[string]any{"location": "L2"}, nil); v != "TRF-2610-0001" {
		t.Errorf("another prefix starts at 1 = %v", v)
	}
	// update: a locked code does not change even if the relation changes / the user sends another value
	if v := gen(map[string]any{"location": "L2", "code": "X"}, map[string]any{"code": "TGD-2610-0001"}); v != "TGD-2610-0001" {
		t.Errorf("locked update = %v", v)
	}
	// not locked: the user value is used, empty = generated
	ct = withCode("WO-{seq}", false)
	if v := gen(map[string]any{"code": "MANUAL"}, nil); v != "MANUAL" {
		t.Errorf("manual = %v", v)
	}
	if v := gen(map[string]any{}, nil); v != "WO-1" {
		t.Errorf("no padding = %v", v)
	}
	// without {seq}: no counter is used
	ct = withCode("{plate}/{YYYY}", true)
	n := len(c)
	if v := gen(map[string]any{"plate": "B1"}, nil); v != "B1/2026" || len(c) != n {
		t.Errorf("no seq = %v", v)
	}
}
