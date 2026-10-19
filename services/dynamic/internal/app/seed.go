package app

import (
	"context"
	"slices"

	"eam/services/common/errs"
	"eam/services/dynamic/internal/domain/content"
)

// Ensure creates missing built-in components & content types (existing ones are left untouched,
// so user changes made through the builder are not overwritten). Seed order is free: components and content types
// are created alternately according to dependencies (relation/component), repeated until nothing more can be created.
// Search index failures do not stop the seed (the schema is still saved) and are reported at the end.
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
	// Leftovers that could not be created: report the original error.
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
