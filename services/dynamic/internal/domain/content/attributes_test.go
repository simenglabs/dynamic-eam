package content

import (
	"strings"
	"testing"

	"eam/services/common/errs"
)

func TestAttributesFieldValidation(t *testing.T) {
	cat := catalog()
	cat.ContentTypes[ClassificationUID] = true
	ct := vehicle()
	ct.Fields = append(ct.Fields,
		Field{Name: "classification", Label: "Classification", Type: TRelation, Target: ClassificationUID},
		Field{Name: "specs", Label: "Atribut", Type: TAttributes, From: "classification"})
	if err := ct.Validate(cat); err != nil {
		t.Fatal(err)
	}
	for _, from := range []string{"", "location", "nope", "specs"} {
		ct.Fields[len(ct.Fields)-1].From = from
		if errs.KindOf(ct.Validate(cat)) != errs.KindInvalid {
			t.Errorf("from %q harus ditolak (harus relasi ke %s)", from, ClassificationUID)
		}
	}
}

func TestValidateAttributes(t *testing.T) {
	defs := []AttributeDef{
		{ID: "a1", Code: "TEGANGAN", Label: "Tegangan", DataType: AttrNumeric, Mandatory: true},
		{ID: "a2", Code: "MERK", Label: "Merk", DataType: AttrALN, Length: 5},
		{ID: "a3", Code: "TGL", Label: "Tgl Pasang", DataType: AttrDate},
		{ID: "a4", Code: "GIS", Label: "GIS", DataType: AttrBoolean},
		{ID: "a5", Code: "KONDISI", Label: "Kondisi", DataType: AttrALN, Domain: "HEALTH_INDEX"},
	}
	domainOK := func(code, v string) bool { return code == "HEALTH_INDEX" && v == "GOOD" }
	got, err := ValidateAttributes(defs, map[string]any{"a1": 20.0, "a2": "ABB", "a3": "2026-01-02", "a4": true, "a5": "GOOD", "lama": "dibuang"}, domainOK)
	if err != nil || len(got) != 5 || got["a1"] != 20.0 {
		t.Fatalf("valid: %v %v", got, err)
	}
	if _, ok := got["lama"]; ok {
		t.Error("atribut di luar classification dibuang (mis. classification diganti)")
	}
	bad := map[string]map[string]any{
		"wajib":       {"a2": "ABB"},
		"angka":       {"a1": "dua puluh"},
		"panjang":     {"a1": 1.0, "a2": "PANJANG SEKALI"},
		"tanggal":     {"a1": 1.0, "a3": "02/01/2026"},
		"boolean":     {"a1": 1.0, "a4": "ya"},
		"domain":      {"a1": 1.0, "a5": "BURUK"},
		"bukan objek": nil,
	}
	for name, v := range bad {
		var in any = v
		if name == "bukan objek" {
			in = "x"
		}
		if _, err := ValidateAttributes(defs, in, domainOK); errs.KindOf(err) != errs.KindInvalid {
			t.Errorf("%s harus ditolak: %v", name, err)
		} else if name == "wajib" && !strings.Contains(err.Error(), "Tegangan") {
			t.Errorf("pesan wajib menyebut atribut: %v", err)
		}
	}
	if got, err := ValidateAttributes(nil, nil, domainOK); err != nil || len(got) != 0 {
		t.Errorf("tanpa classification: %v %v", got, err)
	}
}
