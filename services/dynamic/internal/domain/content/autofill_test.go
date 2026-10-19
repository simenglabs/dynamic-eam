package content

import (
	"testing"
	"time"

	"eam/services/common/errs"
)

func withAutoFill(af *AutoFill) ContentType {
	ct := vehicle() // punya relation "location" -> location
	ct.Fields = append(ct.Fields, Field{Name: "locDesc", Label: "Deskripsi Lokasi", Type: TText, AutoFill: af})
	return ct
}

func TestAutoFillValidation(t *testing.T) {
	ok := []*AutoFill{
		{Source: AutoRelation, From: "location", Field: "description", Lock: true},
		{Source: AutoProfile, Field: "jobTitle"},
		{Source: AutoToday},
		{Source: AutoUser},
	}
	for _, af := range ok {
		ct := withAutoFill(af)
		if err := ct.Validate(catalog()); err != nil {
			t.Errorf("%+v: %v", af, err)
		}
	}
	bad := map[string]*AutoFill{
		"sumber asing":              {Source: "magic"},
		"relation tanpa from":       {Source: AutoRelation, Field: "description"},
		"from bukan relation":       {Source: AutoRelation, From: "plate", Field: "x"},
		"from relation multiple":    {Source: AutoRelation, From: "photos", Field: "x"},
		"relation tanpa field":      {Source: AutoRelation, From: "location"},
		"profile tanpa field":       {Source: AutoProfile},
		"from mengacu diri sendiri": {Source: AutoRelation, From: "locDesc", Field: "x"},
	}
	for name, af := range bad {
		ct := withAutoFill(af)
		if errs.KindOf(ct.Validate(catalog())) != errs.KindInvalid {
			t.Errorf("%s: harus invalid", name)
		}
	}
	comp := withAutoFill(nil)
	comp.Fields[len(comp.Fields)-1].Type = TComponent
	comp.Fields[len(comp.Fields)-1].Component = "address"
	comp.Fields[len(comp.Fields)-1].AutoFill = &AutoFill{Source: AutoToday}
	if errs.KindOf(comp.Validate(catalog())) != errs.KindInvalid {
		t.Error("auto-fill tidak untuk component/media")
	}
}

func TestApplyAutoFill(t *testing.T) {
	ct := withAutoFill(&AutoFill{Source: AutoRelation, From: "location", Field: "description", Lock: true})
	ct.Fields = append(ct.Fields,
		Field{Name: "job", Label: "Jabatan", Type: TText, AutoFill: &AutoFill{Source: AutoProfile, Field: "jobTitle"}},
		Field{Name: "tgl", Label: "Tanggal", Type: TDate, AutoFill: &AutoFill{Source: AutoToday}},
		Field{Name: "oleh", Label: "Oleh", Type: TText, AutoFill: &AutoFill{Source: AutoUser, Lock: true}},
	)
	src := AutoFillSources{
		Related: func(field, id string) map[string]any {
			if field == "location" && id == "L1" {
				return map[string]any{"description": "Gudang Surabaya"}
			}
			return nil
		},
		Profile: map[string]any{"jobTitle": "Teknisi"},
		Actor:   Actor{Name: "Budi"},
		Now:     time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC),
	}
	data := map[string]any{"plate": "L 1", "location": "L1", "locDesc": "diketik user", "job": "Manager"}
	out := ct.ApplyAutoFill(data, src)
	if out["locDesc"] != "Gudang Surabaya" {
		t.Errorf("terkunci: selalu dari relasi, got %v", out["locDesc"])
	}
	if out["job"] != "Manager" {
		t.Errorf("tidak terkunci: isian user dipertahankan, got %v", out["job"])
	}
	if out["tgl"] != "2026-10-19" || out["oleh"] != "Budi" {
		t.Errorf("today/user: %v %v", out["tgl"], out["oleh"])
	}
	if data["locDesc"] != "diketik user" {
		t.Error("input tidak boleh dimutasi")
	}
	cleared := ct.ApplyAutoFill(map[string]any{"plate": "L 1", "locDesc": "lama"}, src)
	if _, ok := cleared["locDesc"]; ok {
		t.Error("relasi dikosongkan: field terkunci ikut kosong")
	}
	empty := ct.ApplyAutoFill(map[string]any{"plate": "L 1"}, src)
	if empty["job"] != "Teknisi" {
		t.Errorf("tidak terkunci & kosong: diisi profil, got %v", empty["job"])
	}
}

// Profil "sites" (list) ke relasi tunggal: ambil yang pertama; ke field multiple: list utuh.
func TestApplyAutoFillListToSingle(t *testing.T) {
	ct := vehicle()
	ct.Fields = append(ct.Fields,
		Field{Name: "site", Type: TRelation, Target: "location", AutoFill: &AutoFill{Source: AutoProfile, Field: "sites", Lock: true}},
		Field{Name: "allSites", Type: TRelation, Target: "location", Multiple: true, AutoFill: &AutoFill{Source: AutoProfile, Field: "sites", Lock: true}})
	out := ct.ApplyAutoFill(nil, AutoFillSources{Profile: map[string]any{"sites": []any{"S1", "S2"}}})
	if out["site"] != "S1" {
		t.Errorf("site = %v, mau S1", out["site"])
	}
	if l, _ := out["allSites"].([]any); len(l) != 2 {
		t.Errorf("allSites = %v", out["allSites"])
	}
}

// Sumber "user" ke relasi profil user = ID user login (pemilik / pembuat), ke teks = namanya.
func TestAutoUserIntoProfileRelation(t *testing.T) {
	ct := vehicle()
	ct.Fields = append(ct.Fields,
		Field{Name: "owner", Label: "Pemilik", Type: TRelation, Target: ProfileUID, AutoFill: &AutoFill{Source: AutoUser, Lock: true}},
		Field{Name: "crew", Label: "Kru", Type: TRelation, Target: ProfileUID, Multiple: true, AutoFill: &AutoFill{Source: AutoUser}})
	cat := catalog()
	cat.ContentTypes[ProfileUID] = true
	if err := ct.Validate(cat); err != nil {
		t.Fatal(err)
	}
	out := ct.ApplyAutoFill(nil, AutoFillSources{Actor: Actor{ID: "U1", Name: "Budi"}})
	if out["owner"] != "U1" {
		t.Errorf("owner = %v", out["owner"])
	}
	if l, _ := out["crew"].([]any); len(l) != 1 || l[0] != "U1" {
		t.Errorf("crew = %v", out["crew"])
	}
	ct.Fields[4].AutoFill = &AutoFill{Source: AutoUser} // location: relasi ke content type lain
	if errs.KindOf(ct.Validate(cat)) != errs.KindInvalid {
		t.Error("user hanya ke teks atau relasi profil user")
	}
}
