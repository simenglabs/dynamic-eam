package http

import (
	"context"
	"slices"
	"testing"
	"time"

	"eam/services/dynamic/internal/app"
	"eam/services/dynamic/internal/domain/content"
	"eam/services/dynamic/internal/infra/memory"
	"eam/services/dynamic/internal/seed"
)

// Seed bawaan harus valid & lengkap: kalau rusak, ketahuan di test, bukan saat service start.
func TestBundledSeedsAreValid(t *testing.T) {
	cases := []struct {
		name, parent string
		uids, views  []string
	}{
		{"master", "master", []string{"master-domain", "master-classification", "master-organization", "master-site", "master-attribute", "master-map-symbol", "master-service-address"}, nil},
		{"profile", "administration", []string{"user-profile"}, nil}, // setelah master: relasi ke Organizations & Sites
		{"assets", "assets", []string{"asset-location", "asset-template", "asset-tnd"}, []string{"Asset Approval (Modify)", "Location Approval (Modify)", "Run Report Assets"}}, // setelah master: relasi ke Sites & Domain
	}
	d := app.Deps{Schemas: memory.NewSchemas(), Entries: memory.NewEntries(), Index: memory.NewIndex(), MediaRepo: memory.NewMedia(), Files: memory.NewFiles(),
		Now: time.Now, NewID: func() string { return "x" }}
	schemas := app.NewSchemas(d)
	for _, c := range cases {
		raw, ok := seed.Files[c.name]
		if !ok {
			t.Fatalf("seed %s tidak ada", c.name)
		}
		comps, cts, entries, err := DecodeSeed(raw)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		created, err := schemas.Ensure(context.Background(), comps, cts)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		for _, s := range entries {
			if _, err := app.NewEntries(d).EnsureEntries(context.Background(), s.UID, s.Key, s.Rows, content.Actor{ID: "system", Super: true}); err != nil {
				t.Fatalf("%s data %s: %v", c.name, s.UID, err)
			}
		}
		views := map[string]string{}
		for _, ct := range cts {
			if ct.Parent != c.parent {
				t.Errorf("%s harus di menu %s", ct.UID, c.parent)
			}
			for _, v := range viewsOf(ct) {
				views[v.Label] = v.Parent
			}
		}
		for _, uid := range c.uids {
			if !slices.Contains(created, uid) {
				t.Errorf("%s: seed tanpa %s (%v)", c.name, uid, created)
			}
		}
		for _, label := range c.views {
			if views[label] != c.parent {
				t.Errorf("view %q tidak ada / bukan di menu %s", label, c.parent)
			}
		}
	}
}

func viewsOf(ct *content.ContentType) []content.View {
	if ct.Workflow == nil {
		return nil
	}
	return ct.Workflow.Views
}

func seedMaster(t *testing.T) []byte {
	t.Helper()
	return seed.Files["master"]
}

func (e *env) ensure(comps []*content.Component, cts []*content.ContentType, entries []SeedEntries) {
	e.t.Helper()
	if _, err := e.schemas.Ensure(context.Background(), comps, cts); err != nil {
		e.t.Fatal(err)
	}
	for _, s := range entries {
		if _, err := e.entries.EnsureEntries(context.Background(), s.UID, s.Key, s.Rows, content.Actor{ID: "system", Name: "System", Super: true}); err != nil {
			e.t.Fatal(err)
		}
	}
}
