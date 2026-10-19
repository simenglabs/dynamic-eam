package content

import (
	"encoding/json"
	"testing"

	"eam/services/common/errs"
)

func parse(s string) map[string]any {
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		panic(err)
	}
	return m
}

func TestValidateEntryNormalizesAndCollectsRefs(t *testing.T) {
	ct := vehicle()
	out, refs, err := ValidateEntry(ct.Fields, parse(`{"plate":"L 1","since":"2026-10-19","location":"L1",
		"photos":["m1","m1","m2"],"addresses":[{"street":"Jl. A","zip":60111}]}`), catalog().Components, true)
	if err != nil {
		t.Fatal(err)
	}
	if out["status"] != "Active" {
		t.Error("default must be filled on create")
	}
	if p := out["photos"].([]string); len(p) != 2 {
		t.Errorf("duplicate media must be dropped: %v", p)
	}
	if refs.Relations["location"][0] != "L1" || len(refs.Media) != 2 {
		t.Errorf("refs: %+v", refs)
	}
	if _, ok := out["active"]; ok {
		t.Error("empty optional fields are not saved")
	}
}

func TestDefaultNotAppliedOnUpdate(t *testing.T) {
	out, _, err := ValidateEntry(vehicle().Fields, parse(`{"plate":"L 1"}`), catalog().Components, false)
	if err != nil || out["status"] != nil {
		t.Errorf("update does not fill defaults: %v %v", out, err)
	}
}

func TestValidateEntryRejects(t *testing.T) {
	for _, s := range []string{
		`{}`,                                                     // plate is required
		`{"plate":""}`,                                           // empty = not filled
		`{"plate":"A","extra":1}`,                                // unknown field
		`{"plate":"A","status":"Lost"}`,                          // select outside options
		`{"plate":"A","since":"19-10-2026"}`,                     // date format
		`{"plate":"A","active":"yes"}`,                           // not a bool
		`{"plate":123}`,                                          // not text
		`{"plate":"A","photos":"m1"}`,                            // multiple must be a list
		`{"plate":"A","location":["L1"]}`,                        // single must be a string
		`{"plate":"A","addresses":[{"zip":1}]}`,                  // street is required in component
		`{"plate":"A","addresses":{"street":"x"}}`,               // repeatable must be a list
		`{"plate":"A","addresses":[{"street":"x","unknown":1}]}`, // unknown field in component
	} {
		if _, _, err := ValidateEntry(vehicle().Fields, parse(s), catalog().Components, true); errs.KindOf(err) != errs.KindInvalid {
			t.Errorf("%s: err = %v, want invalid", s, err)
		}
	}
}

func TestTextLengthLimit(t *testing.T) {
	long := make([]byte, 256)
	for i := range long {
		long[i] = 'x'
	}
	if _, _, err := ValidateEntry(vehicle().Fields, map[string]any{"plate": string(long)}, nil, true); err == nil {
		t.Error("text > 255 characters must be rejected")
	}
}

func TestNumberAcceptsIntegerKinds(t *testing.T) {
	f := []Field{{Name: "n", Label: "N", Type: TNumber}}
	for _, v := range []any{float64(1.5), int(2), int32(3), int64(4)} {
		out, _, err := ValidateEntry(f, map[string]any{"n": v}, nil, true)
		if err != nil {
			t.Errorf("%T: %v", v, err)
		}
		if _, ok := out["n"].(float64); !ok {
			t.Errorf("%T must be normalized to float64", v)
		}
	}
}
