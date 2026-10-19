package app

import (
	"context"
	"slices"

	"eam/services/common/errs"
	"eam/services/dynamic/internal/domain/content"
)

// Ensure membuat component & content type bawaan yang belum ada (yang sudah ada tidak disentuh,
// supaya perubahan user lewat builder tidak tertimpa). Urutan seed bebas: component dan content type dibuat
// berselang-seling sesuai dependensi (relation/component), diulang sampai tidak ada lagi yang bisa dibuat.
// Kegagalan index pencarian tidak menghentikan seed (schema tetap tersimpan) dan dilaporkan di akhir.
func (s *Schemas) Ensure(ctx context.Context, comps []*content.Component, cts []*content.ContentType) (created []string, err error) {
	var unavailable error
	ok := func(e error) bool {
		if errs.KindOf(e) == errs.KindUnavailable {
			unavailable = e
			return true
		}
		return e == nil
	}
	pendingC := slices.DeleteFunc(slices.Clone(comps), func(c *content.Component) bool {
		_, e := s.d.Schemas.GetComponent(ctx, c.UID)
		return e == nil
	})
	pendingT := slices.DeleteFunc(slices.Clone(cts), func(ct *content.ContentType) bool {
		_, e := s.d.Schemas.GetContentType(ctx, ct.UID)
		return e == nil
	})
	for progress := true; progress && len(pendingC)+len(pendingT) > 0; {
		progress = false
		pendingC = slices.DeleteFunc(pendingC, func(c *content.Component) bool {
			if _, e := s.CreateComponent(ctx, c); ok(e) {
				created, progress = append(created, "component:"+c.UID), true
				return true
			}
			return false
		})
		pendingT = slices.DeleteFunc(pendingT, func(ct *content.ContentType) bool {
			if _, e := s.CreateContentType(ctx, ct); ok(e) {
				created, progress = append(created, ct.UID), true
				return true
			}
			return false
		})
	}
	// Sisa yang tidak bisa dibuat: laporkan error aslinya.
	if len(pendingC) > 0 {
		_, e := s.CreateComponent(ctx, pendingC[0])
		return created, e
	}
	if len(pendingT) > 0 {
		_, e := s.CreateContentType(ctx, pendingT[0])
		return created, e
	}
	return created, unavailable
}
