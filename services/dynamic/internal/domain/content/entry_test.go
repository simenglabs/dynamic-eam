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
	if out["status"] != "Aktif" {
		t.Error("default harus diisi saat create")
	}
	if p := out["photos"].([]string); len(p) != 2 {
		t.Errorf("media duplikat harus dibuang: %v", p)
	}
	if refs.Relations["location"][0] != "L1" || len(refs.Media) != 2 {
		t.Errorf("refs: %+v", refs)
	}
	if _, ok := out["active"]; ok {
		t.Error("field kosong opsional tidak ikut disimpan")
	}
}

func TestDefaultNotAppliedOnUpdate(t *testing.T) {
	out, _, err := ValidateEntry(vehicle().Fields, parse(`{"plate":"L 1"}`), catalog().Components, false)
	if err != nil || out["status"] != nil {
		t.Errorf("update tidak mengisi default: %v %v", out, err)
	}
}

func TestValidateEntryRejects(t *testing.T) {
	for _, s := range []string{
		`{}`,                                                   // plate wajib
		`{"plate":""}`,                                         // kosong = tidak diisi
		`{"plate":"A","extra":1}`,                              // field asing
		`{"plate":"A","status":"Hilang"}`,                      // select di luar opsi
		`{"plate":"A","since":"19-10-2026"}`,                   // format tanggal
		`{"plate":"A","active":"ya"}`,                          // bukan bool
		`{"plate":123}`,                                        // bukan teks
		`{"plate":"A","photos":"m1"}`,                          // multiple harus list
		`{"plate":"A","location":["L1"]}`,                      // single harus string
		`{"plate":"A","addresses":[{"zip":1}]}`,                // street wajib di component
		`{"plate":"A","addresses":{"street":"x"}}`,             // repeatable harus list
		`{"plate":"A","addresses":[{"street":"x","asing":1}]}`, // field asing di component
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
		t.Error("text > 255 karakter harus ditolak")
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
			t.Errorf("%T harus dinormalkan ke float64", v)
		}
	}
}
