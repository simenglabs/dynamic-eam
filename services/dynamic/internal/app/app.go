// Package app: use case menu dinamis (schema, entri, media). Izin RBAC dicek di layer http.
package app

import (
	"context"
	"log"
	"time"

	"eam/services/common/errs"
	"eam/services/dynamic/internal/domain/content"
	"eam/services/dynamic/internal/domain/media"
)

type Deps struct {
	Schemas   content.SchemaRepository
	Entries   content.EntryRepository
	Index     content.SearchIndex
	MediaRepo media.Repository
	Files     media.FileStore
	Now       func() time.Time
	NewID     func() string
}

func catalog(ctx context.Context, d Deps) (content.Catalog, error) {
	cat := content.Catalog{ContentTypes: map[string]bool{}, Components: map[string]content.Component{}}
	cts, err := d.Schemas.ListContentTypes(ctx)
	if err != nil {
		return cat, err
	}
	for _, ct := range cts {
		cat.ContentTypes[ct.UID] = true
	}
	comps, err := d.Schemas.ListComponents(ctx)
	for _, c := range comps {
		cat.Components[c.UID] = *c
	}
	return cat, err
}

// indexFailed: data sudah tersimpan di Mongo tapi Elasticsearch gagal (dual-write).
// ponytail: tidak ada retry otomatis; simpan ulang schema = rebuild index. Pakai outbox/change stream kalau perlu otomatis.
func indexFailed(err error) error {
	log.Printf("search index: %v", err)
	return errs.Unavailable("data tersimpan, tapi index pencarian gagal diperbarui; coba simpan ulang schema untuk rebuild")
}
