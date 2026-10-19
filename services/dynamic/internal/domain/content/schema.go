// Package content: bounded context menu dinamis — ContentType, Component (schema) dan Entry (data).
package content

import (
	"regexp"
	"slices"
	"strings"
	"time"

	"eam/services/common/errs"
)

// Tipe field (setara Content-Type Builder Strapi).
const (
	TText       = "text"     // string pendek, max 255
	TLongText   = "longtext" // string panjang, max 100k
	TNumber     = "number"
	TBoolean    = "boolean"
	TDate       = "date"       // YYYY-MM-DD
	TSelect     = "select"     // enum, Options
	TRelation   = "relation"   // id entri content type Target; Multiple = many
	TMedia      = "media"      // id file; Multiple = banyak file
	TComponent  = "component"  // objek sesuai Component; Repeatable = list
	TDomain     = "domain"     // valueId dari master Domain (kode di Field.Domain); opsi & default dari domain
	TAttributes = "attributes" // atribut sesuai classification di field relasi From (lihat attributes.go)
)

var fieldTypes = []string{TText, TLongText, TNumber, TBoolean, TDate, TSelect, TRelation, TMedia, TComponent, TDomain, TAttributes}

type Field struct {
	Name       string
	Label      string
	Type       string
	Required   bool
	Unique     bool
	Default    any
	Options    []string
	Target     string
	Multiple   bool
	Component  string
	Repeatable bool
	UniqueBy   string      // repeatable component: field di component yang nilainya harus unik dalam list
	AutoFill   *AutoFill   // isi otomatis (relasi/profil/hari ini/user)
	Domain     string      // TDomain: kode domain (mis. "PRIORITY")
	Where      []Condition // TRelation: filter pilihan (lihat where.go)
	From       string      // TAttributes: field relasi ke classification
}

type ContentType struct {
	UID          string
	Label        string
	Parent       string // zone tempat menu muncul di sidebar
	Hidden       bool   // true = tidak tampil di sidebar (mis. profil user), izin tetap diatur di Role
	ScopeField   string // field relasi ke Sites: data dibatasi per site akses user (kosong = tidak dibatasi)
	DisplayField string
	Fields       []Field
	Layout       *Layout   // nil = layout default
	Workflow     *Workflow // nil = tanpa status
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Component struct {
	UID       string
	Label     string
	Fields    []Field
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Catalog: schema lain yang sudah ada, untuk validasi relation/component.
type Catalog struct {
	ContentTypes map[string]bool
	Components   map[string]Component
}

// PermKey: kunci RBAC menu dinamis (diatur di Role Management).
func PermKey(uid string) string { return "ct/" + uid }

var (
	uidRe          = regexp.MustCompile(`^[a-z][a-z0-9-]{1,47}$`)
	fieldRe        = regexp.MustCompile(`^[a-z][a-zA-Z0-9]{0,47}$`)
	reservedFields = []string{"id", "createdAt", "updatedAt", "createdBy", "updatedBy", "workflowStatus", "workflowHistory"}
)

func validateHeader(uid, label string) error {
	if !uidRe.MatchString(uid) {
		return errs.Invalid("UID %q tidak valid (huruf kecil, angka, '-', 2-48 karakter)", uid)
	}
	if label == "" {
		return errs.Invalid("label wajib diisi")
	}
	return nil
}

func (ct *ContentType) Validate(cat Catalog) error {
	if err := validateHeader(ct.UID, ct.Label); err != nil {
		return err
	}
	if ct.Parent == "" {
		return errs.Invalid("menu induk wajib dipilih")
	}
	types := map[string]bool{ct.UID: true} // relation ke diri sendiri boleh
	for k, v := range cat.ContentTypes {
		types[k] = v
	}
	if err := validateFields(ct.Fields, Catalog{types, cat.Components}, false); err != nil {
		return err
	}
	if err := ct.validateDisplay(); err != nil {
		return err
	}
	if err := ct.validateScope(); err != nil {
		return err
	}
	if err := ct.validateAutoFill(); err != nil {
		return err
	}
	if ct.Layout != nil {
		if err := ct.Layout.validate(ct.Fields); err != nil {
			return err
		}
	}
	if ct.Workflow != nil {
		return ct.Workflow.validate(Catalog{types, cat.Components})
	}
	return nil
}

func (c *Component) Validate(cat Catalog) error {
	if err := validateHeader(c.UID, c.Label); err != nil {
		return err
	}
	comps := map[string]Component{c.UID: *c}
	for k, v := range cat.Components {
		if k != c.UID {
			comps[k] = v
		}
	}
	if err := validateFields(c.Fields, Catalog{cat.ContentTypes, comps}, true); err != nil {
		return err
	}
	// Cegah siklus A -> B -> A (validasi entri akan rekursi tanpa akhir).
	var visit func(uid string, path []string) error
	visit = func(uid string, path []string) error {
		if slices.Contains(path, uid) {
			return errs.Invalid("component bersiklus: %v", append(path, uid))
		}
		for _, f := range comps[uid].Fields {
			if f.Type == TComponent {
				if err := visit(f.Component, append(path, uid)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return visit(c.UID, nil)
}

func validateFields(fields []Field, cat Catalog, inComponent bool) error {
	if len(fields) == 0 {
		return errs.Invalid("minimal satu field")
	}
	seen := map[string]bool{}
	for _, f := range fields {
		if !fieldRe.MatchString(f.Name) || slices.Contains(reservedFields, f.Name) {
			return errs.Invalid("nama field %q tidak valid (camelCase, bukan %v)", f.Name, reservedFields)
		}
		if seen[f.Name] {
			return errs.Invalid("nama field %q duplikat", f.Name)
		}
		seen[f.Name] = true
		if f.Label == "" {
			return errs.Invalid("label field %q wajib diisi", f.Name)
		}
		if !slices.Contains(fieldTypes, f.Type) {
			return errs.Invalid("tipe field %q tidak dikenal: %q", f.Name, f.Type)
		}
		if f.Unique && (inComponent || (f.Type != TText && f.Type != TNumber)) {
			return errs.Invalid("field %q: unique hanya untuk text/number di level atas", f.Name)
		}
		if err := validateWhere(f, fields, inComponent); err != nil {
			return err
		}
		switch f.Type {
		case TAttributes:
			if err := validateAttributesField(f, fields, inComponent); err != nil {
				return err
			}
		case TSelect:
			if len(f.Options) == 0 {
				return errs.Invalid("field %q: select butuh minimal satu opsi", f.Name)
			}
		case TRelation:
			if !cat.ContentTypes[f.Target] {
				return errs.Invalid("field %q: target relation %q tidak ada", f.Name, f.Target)
			}
		case TComponent:
			comp, ok := cat.Components[f.Component]
			if !ok {
				return errs.Invalid("field %q: component %q tidak ada", f.Name, f.Component)
			}
			if f.UniqueBy != "" && (!f.Repeatable || !slices.ContainsFunc(comp.Fields, func(x Field) bool { return x.Name == f.UniqueBy })) {
				return errs.Invalid("field %q: unik-berdasarkan hanya untuk component repeatable dan harus field milik component", f.Name)
			}
		case TDomain:
			if strings.TrimSpace(f.Domain) == "" {
				return errs.Invalid("field %q: kode domain wajib diisi", f.Name)
			}
		}
		if f.Default != nil {
			if _, err := coerce(f, f.Default, nil); err != nil {
				return errs.Invalid("field %q: default tidak valid: %v", f.Name, err)
			}
		}
	}
	return nil
}

func (ct *ContentType) UniqueFields() []string {
	var out []string
	for _, f := range ct.Fields {
		if f.Unique {
			out = append(out, f.Name)
		}
	}
	return out
}

// References: content type ini punya relation ke target (selain dirinya sendiri)?
func (ct *ContentType) References(target string) bool {
	return target != ct.UID && slices.ContainsFunc(ct.Fields, func(f Field) bool { return f.Type == TRelation && f.Target == target })
}

func usesComponent(fields []Field, uid string) bool {
	return slices.ContainsFunc(fields, func(f Field) bool { return f.Type == TComponent && f.Component == uid })
}

func (ct *ContentType) UsesComponent(uid string) bool { return usesComponent(ct.Fields, uid) }
func (c *Component) UsesComponent(uid string) bool    { return usesComponent(c.Fields, uid) }
