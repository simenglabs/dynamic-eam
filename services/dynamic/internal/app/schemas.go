package app

import (
	"context"
	"fmt"

	"eam/services/common/errs"
	"eam/services/dynamic/internal/domain/content"
)

type Schemas struct{ d Deps }

func NewSchemas(d Deps) *Schemas { return &Schemas{d} }

func (s *Schemas) ListContentTypes(ctx context.Context) ([]*content.ContentType, error) {
	return s.d.Schemas.ListContentTypes(ctx)
}

func (s *Schemas) GetContentType(ctx context.Context, uid string) (*content.ContentType, error) {
	return s.d.Schemas.GetContentType(ctx, uid)
}

func (s *Schemas) CreateContentType(ctx context.Context, ct *content.ContentType) (*content.ContentType, error) {
	if _, err := s.d.Schemas.GetContentType(ctx, ct.UID); err == nil {
		return nil, errs.Conflict("UID %q sudah dipakai", ct.UID)
	}
	ct.CreatedAt = s.d.Now()
	return ct, s.saveContentType(ctx, ct)
}

// UpdateContentType: UID & CreatedAt tidak bisa diubah.
func (s *Schemas) UpdateContentType(ctx context.Context, uid string, ct *content.ContentType) (*content.ContentType, error) {
	old, err := s.d.Schemas.GetContentType(ctx, uid)
	if err != nil {
		return nil, err
	}
	ct.UID, ct.CreatedAt = uid, old.CreatedAt
	return ct, s.saveContentType(ctx, ct)
}

func (s *Schemas) saveContentType(ctx context.Context, ct *content.ContentType) error {
	cat, err := catalog(ctx, s.d)
	if err != nil {
		return err
	}
	if err := ct.Validate(cat); err != nil {
		return err
	}
	if err := s.d.Entries.SyncUnique(ctx, ct.UID, ct.UniqueFields()); err != nil {
		return err
	}
	ct.UpdatedAt = s.d.Now()
	if err := s.d.Schemas.SaveContentType(ctx, ct); err != nil {
		return err
	}
	// ponytail: rebuild penuh tiap schema berubah; cukup untuk ribuan entri. Pakai alias + reindex bertahap kalau besar.
	all, err := s.d.Entries.All(ctx, ct.UID)
	if err != nil {
		return err
	}
	if err := s.d.Index.Rebuild(ctx, ct, all); err != nil {
		return indexFailed(err)
	}
	return nil
}

func (s *Schemas) DeleteContentType(ctx context.Context, uid string) error {
	if _, err := s.d.Schemas.GetContentType(ctx, uid); err != nil {
		return err
	}
	cts, err := s.d.Schemas.ListContentTypes(ctx)
	if err != nil {
		return err
	}
	for _, other := range cts {
		if other.References(uid) {
			return errs.Conflict("masih dipakai sebagai target relation oleh %s", other.Label)
		}
	}
	if err := s.d.Schemas.DeleteContentType(ctx, uid); err != nil {
		return err
	}
	if err := s.d.Entries.Drop(ctx, uid); err != nil {
		return err
	}
	if err := s.d.Index.Drop(ctx, uid); err != nil {
		return indexFailed(err)
	}
	return nil
}

func (s *Schemas) ListComponents(ctx context.Context) ([]*content.Component, error) {
	return s.d.Schemas.ListComponents(ctx)
}

func (s *Schemas) GetComponent(ctx context.Context, uid string) (*content.Component, error) {
	return s.d.Schemas.GetComponent(ctx, uid)
}

func (s *Schemas) CreateComponent(ctx context.Context, c *content.Component) (*content.Component, error) {
	if _, err := s.d.Schemas.GetComponent(ctx, c.UID); err == nil {
		return nil, errs.Conflict("UID %q sudah dipakai", c.UID)
	}
	c.CreatedAt = s.d.Now()
	return c, s.saveComponent(ctx, c)
}

func (s *Schemas) UpdateComponent(ctx context.Context, uid string, c *content.Component) (*content.Component, error) {
	old, err := s.d.Schemas.GetComponent(ctx, uid)
	if err != nil {
		return nil, err
	}
	c.UID, c.CreatedAt = uid, old.CreatedAt
	return c, s.saveComponent(ctx, c)
}

func (s *Schemas) saveComponent(ctx context.Context, c *content.Component) error {
	cat, err := catalog(ctx, s.d)
	if err != nil {
		return err
	}
	if err := c.Validate(cat); err != nil {
		return err
	}
	c.UpdatedAt = s.d.Now()
	return s.d.Schemas.SaveComponent(ctx, c)
}

func (s *Schemas) DeleteComponent(ctx context.Context, uid string) error {
	if _, err := s.d.Schemas.GetComponent(ctx, uid); err != nil {
		return err
	}
	cts, err := s.d.Schemas.ListContentTypes(ctx)
	if err != nil {
		return err
	}
	for _, ct := range cts {
		if ct.UsesComponent(uid) {
			return errs.Conflict("component masih dipakai oleh %s", ct.Label)
		}
	}
	comps, err := s.d.Schemas.ListComponents(ctx)
	if err != nil {
		return err
	}
	for _, c := range comps {
		if c.UID != uid && c.UsesComponent(uid) {
			return errs.Conflict("component masih dipakai oleh component %s", c.Label)
		}
	}
	return s.d.Schemas.DeleteComponent(ctx, uid)
}

// SyncConstraints menyamakan constraint database (unique) semua content type dengan schema-nya.
// Dipanggil saat start supaya perubahan aturan (mis. unique mengabaikan entri terhapus) berlaku tanpa simpan ulang schema.
func (s *Schemas) SyncConstraints(ctx context.Context) error {
	cts, err := s.d.Schemas.ListContentTypes(ctx)
	if err != nil {
		return err
	}
	for _, ct := range cts {
		if err := s.d.Entries.SyncUnique(ctx, ct.UID, ct.UniqueFields()); err != nil {
			return fmt.Errorf("%s: %w", ct.UID, err)
		}
	}
	return nil
}
