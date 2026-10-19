package app

import (
	"context"
	"fmt"
	"time"

	"eam/services/dynamic/internal/domain/content"
	"eam/services/dynamic/internal/infra/memory"
)

var t0 = time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)

type env struct {
	ctx     context.Context
	index   *memory.Index
	files   *memory.Files
	schemas *Schemas
	entries *Entries
	media   *Media
}

func newEnv() *env {
	seq := 0
	d := Deps{
		Schemas: memory.NewSchemas(), Entries: memory.NewEntries(), Index: memory.NewIndex(),
		MediaRepo: memory.NewMedia(), Files: memory.NewFiles(),
		Now:   func() time.Time { return t0 },
		NewID: func() string { seq++; return fmt.Sprintf("id%d", seq) },
	}
	e := &env{ctx: context.Background(), index: d.Index.(*memory.Index), files: d.Files.(*memory.Files)}
	e.schemas, e.entries, e.media = NewSchemas(d), NewEntries(d), NewMedia(d)
	return e
}

func locationCT() *content.ContentType {
	return &content.ContentType{UID: "location", Label: "Lokasi", Parent: "master", DisplayField: "name",
		Fields: []content.Field{{Name: "name", Label: "Nama", Type: content.TText, Required: true, Unique: true}}}
}

func vehicleCT() *content.ContentType {
	return &content.ContentType{UID: "vehicle", Label: "Kendaraan", Parent: "assets", DisplayField: "plate", Fields: []content.Field{
		{Name: "plate", Label: "Plat", Type: content.TText, Required: true, Unique: true},
		{Name: "status", Label: "Status", Type: content.TSelect, Options: []string{"Aktif", "Rusak"}, Default: "Aktif"},
		{Name: "location", Label: "Lokasi", Type: content.TRelation, Target: "location"},
		{Name: "photos", Label: "Foto", Type: content.TMedia, Multiple: true},
	}}
}

// seed: content type location + vehicle dan satu lokasi "Gudang".
func (e *env) seed() (locID string) {
	if _, err := e.schemas.CreateContentType(e.ctx, locationCT()); err != nil {
		panic(err)
	}
	if _, err := e.schemas.CreateContentType(e.ctx, vehicleCT()); err != nil {
		panic(err)
	}
	loc, err := e.entries.Create(e.ctx, "location", map[string]any{"name": "Gudang"}, u1)
	if err != nil {
		panic(err)
	}
	return loc.ID
}

var (
	u1 = content.Actor{ID: "u1", Name: "Budi"}
	u2 = content.Actor{ID: "u2", Name: "Ani"}
)
