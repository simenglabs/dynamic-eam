package content

import (
	"testing"

	"eam/services/common/errs"
)

func catalog() Catalog {
	return Catalog{
		ContentTypes: map[string]bool{"location": true},
		Components: map[string]Component{"address": {UID: "address", Label: "Alamat", Fields: []Field{
			{Name: "street", Label: "Jalan", Type: TText, Required: true},
			{Name: "zip", Label: "Kode Pos", Type: TNumber},
		}}},
	}
}

func vehicle() ContentType {
	return ContentType{UID: "vehicle", Label: "Kendaraan", Parent: "assets", DisplayField: "plate", Fields: []Field{
		{Name: "plate", Label: "Plat", Type: TText, Required: true, Unique: true},
		{Name: "status", Label: "Status", Type: TSelect, Options: []string{"Aktif", "Rusak"}, Default: "Aktif"},
		{Name: "active", Label: "Aktif", Type: TBoolean},
		{Name: "since", Label: "Sejak", Type: TDate},
		{Name: "location", Label: "Lokasi", Type: TRelation, Target: "location"},
		{Name: "photos", Label: "Foto", Type: TMedia, Multiple: true},
		{Name: "addresses", Label: "Alamat", Type: TComponent, Component: "address", Repeatable: true},
	}}
}

func TestValidContentType(t *testing.T) {
	ct := vehicle()
	if err := ct.Validate(catalog()); err != nil {
		t.Fatal(err)
	}
}

func TestSelfRelationAllowed(t *testing.T) {
	ct := vehicle()
	ct.Fields = append(ct.Fields, Field{Name: "parent", Label: "Induk", Type: TRelation, Target: "vehicle"})
	if err := ct.Validate(catalog()); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidContentTypes(t *testing.T) {
	cases := map[string]func(*ContentType){
		"uid salah":            func(c *ContentType) { c.UID = "Bad UID" },
		"label kosong":         func(c *ContentType) { c.Label = "" },
		"tanpa induk":          func(c *ContentType) { c.Parent = "" },
		"tanpa field":          func(c *ContentType) { c.Fields = nil },
		"nama field reserved":  func(c *ContentType) { c.Fields[0].Name = "id" },
		"nama field duplikat":  func(c *ContentType) { c.Fields[1].Name = "plate" },
		"tipe asing":           func(c *ContentType) { c.Fields[2].Type = "json" },
		"select tanpa opsi":    func(c *ContentType) { c.Fields[1].Options = nil; c.Fields[1].Default = nil },
		"relation tak ada":     func(c *ContentType) { c.Fields[4].Target = "nope" },
		"component tak ada":    func(c *ContentType) { c.Fields[6].Component = "nope" },
		"default di luar opsi": func(c *ContentType) { c.Fields[1].Default = "Hilang" },
		"unique di boolean":    func(c *ContentType) { c.Fields[2].Unique = true },
		"display field media":  func(c *ContentType) { c.DisplayField = "photos" },
		"display field asing":  func(c *ContentType) { c.DisplayField = "nope" },
	}
	for name, mutate := range cases {
		ct := vehicle()
		ct.Fields = append([]Field(nil), ct.Fields...)
		mutate(&ct)
		if errs.KindOf(ct.Validate(catalog())) != errs.KindInvalid {
			t.Errorf("%s: harus invalid", name)
		}
	}
}

func TestComponentValidation(t *testing.T) {
	cat := catalog()
	ok := Component{UID: "contact", Label: "Kontak", Fields: []Field{{Name: "phone", Label: "Telp", Type: TText}, {Name: "addr", Label: "Alamat", Type: TComponent, Component: "address"}}}
	if err := ok.Validate(cat); err != nil {
		t.Fatal(err)
	}
	loop := Component{UID: "loop", Label: "Loop", Fields: []Field{{Name: "self", Label: "Self", Type: TComponent, Component: "loop"}}}
	if errs.KindOf(loop.Validate(cat)) != errs.KindInvalid {
		t.Error("component bersiklus harus invalid")
	}
	uniq := Component{UID: "u", Label: "U", Fields: []Field{{Name: "code", Label: "Kode", Type: TText, Unique: true}}}
	uniq.UID = "uq"
	if errs.KindOf(uniq.Validate(cat)) != errs.KindInvalid {
		t.Error("unique di dalam component harus invalid")
	}
}

func TestUniqueAndTextFields(t *testing.T) {
	ct := vehicle()
	if u := ct.UniqueFields(); len(u) != 1 || u[0] != "plate" {
		t.Errorf("UniqueFields = %v", u)
	}
	if !ct.References("location") || ct.References("vehicle") {
		t.Error("References salah")
	}
	if !ct.UsesComponent("address") {
		t.Error("UsesComponent salah")
	}
}

func TestPermKey(t *testing.T) {
	if PermKey("vehicle") != "ct/vehicle" {
		t.Error(PermKey("vehicle"))
	}
}
