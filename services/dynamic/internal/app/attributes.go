package app

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"

	"eam/services/common/errs"
	"eam/services/dynamic/internal/domain/content"
)

// AttributesOf: atribut sebuah classification + warisan induknya (atribut di anak menimpa induk), urut sortOrder.
func (s *Entries) AttributesOf(ctx context.Context, classID string) ([]content.AttributeDef, error) {
	var defs []content.AttributeDef
	seen := map[string]bool{}
	for id, depth := classID, 0; id != "" && !seen["class:"+id]; depth++ {
		seen["class:"+id] = true
		c, err := s.live(ctx, content.ClassificationUID, id)
		if err != nil {
			if depth == 0 {
				return nil, err
			}
			break // induk sudah dihapus: cukup sampai sini
		}
		list, _ := c.Data[content.ClassAttributesField].([]any)
		for _, x := range list {
			m, _ := x.(map[string]any)
			aid, _ := m["attribute"].(string)
			if aid == "" || seen[aid] {
				continue
			}
			seen[aid] = true
			a, err := s.live(ctx, content.AttributeUID, aid)
			if err != nil {
				continue // atribut dihapus
			}
			d := content.AttributeDef{ID: aid, Code: str(a.Data["attributeId"]), Label: cmp.Or(str(a.Data["description"]), str(a.Data["attributeId"])),
				DataType: cmp.Or(str(a.Data["dataType"]), content.AttrALN), Unit: str(a.Data["unit"]), Order: 1e9}
			d.Mandatory, _ = m["mandatory"].(bool)
			if o, ok := m["sortOrder"].(float64); ok {
				d.Order = o
			}
			if n, ok := a.Data["length"].(float64); ok {
				d.Length = int(n)
			}
			if dom := str(a.Data["domain"]); dom != "" {
				if de, err := s.live(ctx, content.DomainUID, dom); err == nil {
					d.Domain = str(de.Data[content.DomainCodeField])
				}
			}
			defs = append(defs, d)
		}
		id, _ = c.Data[content.ClassParentField].(string)
	}
	slices.SortStableFunc(defs, func(a, b content.AttributeDef) int { return cmp.Compare(a.Order, b.Order) })
	return defs, nil
}

func str(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

// applyAttributes: nilai field atribut divalidasi terhadap classification yang dipilih.
func (s *Entries) applyAttributes(ctx context.Context, ct *content.ContentType, data map[string]any) error {
	domains := map[string][]content.DomainValue{}
	domainOK := func(code, v string) bool {
		if _, ok := domains[code]; !ok {
			domains[code], _ = s.DomainValues(ctx, code)
		}
		return slices.ContainsFunc(domains[code], func(x content.DomainValue) bool { return x.ID == v })
	}
	for _, f := range ct.Fields {
		if f.Type != content.TAttributes {
			continue
		}
		var defs []content.AttributeDef
		if id, _ := data[f.From].(string); id != "" {
			var err error
			if defs, err = s.AttributesOf(ctx, id); err != nil && errs.KindOf(err) != errs.KindNotFound {
				return err
			}
		}
		vals, err := content.ValidateAttributes(defs, data[f.Name], domainOK)
		if err != nil {
			return err
		}
		if len(vals) == 0 {
			delete(data, f.Name)
		} else {
			data[f.Name] = vals
		}
	}
	return nil
}

// checkWhere: pilihan relasi harus memenuhi filter field-nya (dropdown hanya membantu, server yang menegakkan).
func (s *Entries) checkWhere(ctx context.Context, ct *content.ContentType, data map[string]any) error {
	for _, f := range ct.Fields {
		filters := f.Filters(data)
		if f.Type != content.TRelation || len(filters) == 0 || data[f.Name] == nil {
			continue
		}
		ids, _ := data[f.Name].([]string)
		if id, ok := data[f.Name].(string); ok {
			ids = []string{id}
		}
		for _, id := range ids {
			if e, err := s.live(ctx, f.Target, id); err == nil && !content.Matches(e, filters) {
				return errs.Invalid("%s: pilihan tidak sesuai filter", f.Label)
			}
		}
	}
	return nil
}

// ensureProfile: field "isi dari user" ke relasi profil user butuh entri profil user login. Dibuat otomatis bila belum ada,
// dan nama user disimpan di profil (field "name") supaya label relasinya nama, bukan ID.
func (s *Entries) ensureProfile(ctx context.Context, ct *content.ContentType, by content.Actor) error {
	need := slices.ContainsFunc(ct.Fields, func(f content.Field) bool {
		return f.Type == content.TRelation && f.AutoFill != nil && f.AutoFill.Source == content.AutoUser
	})
	if !need || by.ID == "" || ct.UID == content.ProfileUID {
		return nil
	}
	pct, err := s.d.Schemas.GetContentType(ctx, content.ProfileUID)
	if err != nil {
		return err
	}
	hasName := slices.ContainsFunc(pct.Fields, func(f content.Field) bool { return f.Name == content.ProfileName })
	e, err := s.live(ctx, content.ProfileUID, by.ID)
	data := map[string]any{}
	switch {
	case err == nil:
		if !hasName || e.Data[content.ProfileName] == by.Name {
			return nil
		}
		data = maps.Clone(e.Data)
	case errs.KindOf(err) != errs.KindNotFound:
		return err
	}
	if hasName {
		data[content.ProfileName] = by.Name
	}
	_, err = s.Upsert(ctx, content.ProfileUID, by.ID, data, by)
	return err
}
