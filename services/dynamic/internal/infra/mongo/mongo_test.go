package mongo

import (
	"context"
	"os"
	"testing"
	"time"

	"eam/services/common"
	"eam/services/common/errs"
	"eam/services/dynamic/internal/domain/content"
	"eam/services/dynamic/internal/domain/media"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func setup(t *testing.T) *Store {
	if os.Getenv("MONGO_URI") == "" {
		t.Skip("MONGO_URI kosong")
	}
	t.Setenv("MONGO_DB", "eam_test_dynamic")
	db := common.Mongo()
	ctx := context.Background()
	db.Drop(ctx)
	t.Cleanup(func() { db.Drop(ctx) })
	return New(db)
}

var t0 = time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)

func TestSchemaRoundTrip(t *testing.T) {
	s := setup(t)
	ctx := context.Background()
	ct := &content.ContentType{UID: "vehicle", Label: "Kendaraan", Parent: "assets", Hidden: true, ScopeField: "plate", DisplayField: "plate", CreatedAt: t0, UpdatedAt: t0,
		Fields: []content.Field{
			{Name: "plate", Label: "Plat", Type: content.TText, Required: true, Unique: true},
			{Name: "status", Label: "Status", Type: content.TSelect, Options: []string{"A", "B"}, Default: "A"},
			{Name: "addr", Label: "Alamat", Type: content.TComponent, Component: "address", Repeatable: true, UniqueBy: "street"},
			{Name: "priority", Label: "Priority", Type: content.TDomain, Domain: "PRIORITY"},
			{Name: "desc", Label: "Desc", Type: content.TText, AutoFill: &content.AutoFill{Source: content.AutoRelation, From: "loc", Field: "name", Lock: true}},
			{Name: "code", Label: "Kode", Type: content.TText, AutoFill: &content.AutoFill{Source: content.AutoPattern, Pattern: "A-{seq:3}", Lock: true}},
			{Name: "loc2", Label: "Loc2", Type: content.TRelation, Target: "loc", Where: []content.Condition{{Field: "site", Value: "{code}"}}},
			{Name: "cls", Label: "Cls", Type: content.TRelation, Target: content.ClassificationUID},
			{Name: "specs", Label: "Specs", Type: content.TAttributes, From: "cls"},
		},
		Layout: &content.Layout{Sections: []content.Section{{ID: "a", Title: "Utama", Rows: []content.Row{{Columns: 2, Fields: []string{"plate", ""}}}}}},
		Workflow: &content.Workflow{States: []content.State{{Key: "draft", Label: "Draft", Color: "neutral", Editable: true}}, Initial: "draft",
			Transitions: []content.Transition{{Key: "x", Label: "X", From: []string{"draft"}, To: "draft",
				Fields: []content.Field{{Name: "jenis", Label: "Jenis", Type: content.TSelect, Options: []string{"BA"}, Required: true}}}},
			Views:       []content.View{{Key: "v", Label: "V", Parent: "assets", States: []string{"draft"}}}},
	}
	if err := s.SaveContentType(ctx, ct); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetContentType(ctx, "vehicle")
	if err != nil {
		t.Fatal(err)
	}
	if af := got.Fields[4].AutoFill; af == nil || af.Source != "relation" || af.From != "loc" || af.Field != "name" || !af.Lock {
		t.Errorf("autoFill hilang: %+v", got.Fields[4].AutoFill)
	}
	if af := got.Fields[5].AutoFill; af == nil || af.Pattern != "A-{seq:3}" {
		t.Errorf("pola kode hilang: %+v", af)
	}
	if w := got.Fields[6].Where; len(w) != 1 || w[0].Field != "site" || w[0].Value != "{code}" {
		t.Errorf("where hilang: %+v", w)
	}
	if got.Fields[8].From != "cls" {
		t.Errorf("from hilang: %+v", got.Fields[8])
	}
	if tf := got.Workflow.Transitions[0].Fields; len(tf) != 1 || tf[0].Name != "jenis" || !tf[0].Required || tf[0].Options[0] != "BA" {
		t.Errorf("isian transisi hilang: %+v", tf)
	}
	if !got.Hidden || got.ScopeField != "plate" {
		t.Errorf("hidden/scopeField hilang: %v %q", got.Hidden, got.ScopeField)
	}
	if got.Fields[2].UniqueBy != "street" || got.Fields[3].Domain != "PRIORITY" {
		t.Errorf("uniqueBy/domain hilang: %+v %+v", got.Fields[2], got.Fields[3])
	}
	if got.Fields[1].Default != "A" || !got.Fields[2].Repeatable || got.Layout.Sections[0].Rows[0].Fields[1] != "" ||
		got.Workflow.Views[0].States[0] != "draft" || !got.Workflow.States[0].Editable || !got.CreatedAt.Equal(t0) {
		t.Errorf("round trip: %+v", got)
	}
	if _, err := s.GetContentType(ctx, "nope"); errs.KindOf(err) != errs.KindNotFound {
		t.Errorf("not found: %v", err)
	}
	comp := &content.Component{UID: "address", Label: "Alamat", Fields: []content.Field{{Name: "street", Label: "Jalan", Type: content.TText}}, CreatedAt: t0}
	s.SaveComponent(ctx, comp)
	if list, _ := s.ListComponents(ctx); len(list) != 1 || list[0].Fields[0].Name != "street" {
		t.Errorf("components: %+v", list)
	}
	s.DeleteContentType(ctx, "vehicle")
	if list, _ := s.ListContentTypes(ctx); len(list) != 0 {
		t.Error("harus terhapus")
	}
}

func TestEntriesRoundTripAndUnique(t *testing.T) {
	s := setup(t)
	ctx := context.Background()
	e := &content.Entry{ID: "e1", CreatedAt: t0, UpdatedAt: t0, CreatedBy: "u1", UpdatedBy: "u1", Status: "draft",
		History: []content.StatusChange{{From: "", To: "draft", Transition: "create", By: "u1", At: t0, Note: "n"}},
		Data:    map[string]any{"plate": "L 1", "year": 2021.0, "tags": []string{"a"}, "addr": []any{map[string]any{"street": "Jl. A"}}}}
	if err := s.Save(ctx, "vehicle", e); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(ctx, "vehicle", "e1")
	if err != nil {
		t.Fatal(err)
	}
	addr := got.Data["addr"].([]any)[0].(map[string]any)
	if got.Data["plate"] != "L 1" || got.Data["year"] != 2021.0 || addr["street"] != "Jl. A" || got.Status != "draft" ||
		len(got.History) != 1 || got.History[0].Note != "n" || got.CreatedBy != "u1" || !got.CreatedAt.Equal(t0) {
		t.Errorf("round trip: %+v", got)
	}
	if _, ok := got.Data["_id"]; ok {
		t.Error("metadata tidak boleh bocor ke Data")
	}

	if err := s.SyncUnique(ctx, "vehicle", []string{"plate"}); err != nil {
		t.Fatal(err)
	}
	dup := &content.Entry{ID: "e2", CreatedAt: t0, Data: map[string]any{"plate": "L 1"}}
	if err := s.Save(ctx, "vehicle", dup); errs.KindOf(err) != errs.KindConflict {
		t.Errorf("unique: %v", err)
	}
	if err := s.SyncUnique(ctx, "vehicle", nil); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(ctx, "vehicle", dup); err != nil {
		t.Errorf("unique dicabut: %v", err)
	}
	if err := s.SyncUnique(ctx, "vehicle", []string{"plate"}); errs.KindOf(err) != errs.KindConflict {
		t.Errorf("unique dengan data duplikat: %v", err)
	}
	if n, _ := s.CountExisting(ctx, "vehicle", []string{"e1", "e2", "x"}); n != 2 {
		t.Errorf("CountExisting = %d", n)
	}
	if all, _ := s.All(ctx, "vehicle"); len(all) != 2 {
		t.Errorf("All = %d", len(all))
	}
}

func TestSoftDeleteVersionAndRevisions(t *testing.T) {
	s := setup(t)
	ctx := context.Background()
	if err := s.SyncUnique(ctx, "vehicle", []string{"plate"}); err != nil {
		t.Fatal(err)
	}
	e := &content.Entry{ID: "e1", CreatedAt: t0, UpdatedAt: t0, CreatedBy: "u1", UpdatedBy: "u1", Version: 1, Data: map[string]any{"plate": "L 1"}}
	if err := s.Save(ctx, "vehicle", e); err != nil {
		t.Fatal(err)
	}
	e.Deleted, e.DeletedAt, e.DeletedBy, e.Version = true, t0, "u2", 2
	if err := s.Save(ctx, "vehicle", e); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(ctx, "vehicle", "e1")
	if err != nil || !got.Deleted || got.DeletedBy != "u2" || !got.DeletedAt.Equal(t0) || got.Version != 2 {
		t.Fatalf("soft delete round trip: %+v %v", got, err)
	}
	if all, _ := s.All(ctx, "vehicle"); len(all) != 0 {
		t.Error("All harus mengabaikan entri terhapus")
	}
	if n, _ := s.CountExisting(ctx, "vehicle", []string{"e1"}); n != 0 {
		t.Error("CountExisting harus mengabaikan entri terhapus")
	}
	if err := s.Save(ctx, "vehicle", &content.Entry{ID: "e2", CreatedAt: t0, Data: map[string]any{"plate": "L 1"}}); err != nil {
		t.Errorf("unique hanya antar entri aktif: %v", err)
	}
	if err := s.Save(ctx, "vehicle", &content.Entry{ID: "e3", CreatedAt: t0, Data: map[string]any{"plate": "L 1"}}); errs.KindOf(err) != errs.KindConflict {
		t.Errorf("unique antar entri aktif tetap berlaku: %v", err)
	}

	for v, action := range []string{"create", "delete"} {
		rev := &content.Revision{EntryID: "e1", Version: v + 1, Action: action, By: "u1", ByName: "Budi", At: t0,
			Changes: []content.Change{{Field: "plate", From: nil, To: "L 1"}}, Note: "n",
			Transition: "submit", Data: map[string]any{"jenis": "BA", "files": []any{"m1"}}}
		if err := s.AddRevision(ctx, "vehicle", rev); err != nil {
			t.Fatal(err)
		}
	}
	revs, err := s.Revisions(ctx, "vehicle", "e1")
	if err != nil || len(revs) != 2 || revs[0].Action != "create" || revs[1].Version != 2 || revs[0].ByName != "Budi" ||
		revs[0].Changes[0].To != "L 1" || revs[0].Changes[0].From != nil || !revs[0].At.Equal(t0) {
		t.Errorf("revisions: %+v %v", revs, err)
	}
	if l, _ := revs[0].Data["files"].([]any); revs[0].Transition != "submit" || revs[0].Data["jenis"] != "BA" || len(l) != 1 {
		t.Errorf("isian transisi di revisi: %+v", revs[0])
	}
	if other, _ := s.Revisions(ctx, "vehicle", "e2"); len(other) != 0 {
		t.Error("revisi entri lain tidak boleh ikut")
	}
}

func TestFindOneIgnoresDeleted(t *testing.T) {
	s := setup(t)
	ctx := context.Background()
	s.Save(ctx, "d", &content.Entry{ID: "old", CreatedAt: t0, Deleted: true, Data: map[string]any{"domainCode": "PRIORITY"}})
	s.Save(ctx, "d", &content.Entry{ID: "new", CreatedAt: t0, Data: map[string]any{"domainCode": "PRIORITY"}})
	if e, err := s.FindOne(ctx, "d", "domainCode", "PRIORITY"); err != nil || e.ID != "new" {
		t.Errorf("FindOne: %+v %v", e, err)
	}
	if _, err := s.FindOne(ctx, "d", "domainCode", "NOPE"); errs.KindOf(err) != errs.KindNotFound {
		t.Errorf("tidak ada: %v", err)
	}
}

func TestLegacyUniqueIndexReplaced(t *testing.T) {
	s := setup(t)
	ctx := context.Background()
	col := s.db.Collection("ct_vehicle")
	col.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "plate", Value: 1}}, Options: options.Index().SetName("uniq_plate").SetUnique(true)})
	if err := s.SyncUnique(ctx, "vehicle", []string{"plate"}); err != nil {
		t.Fatal(err)
	}
	specs, _ := col.Indexes().ListSpecifications(ctx)
	for _, sp := range specs {
		if sp.Name == "uniq_plate" {
			t.Error("index unique lama (tanpa filter isDeleted) harus diganti")
		}
	}
}

func TestMediaRepository(t *testing.T) {
	s := setup(t)
	ctx := context.Background()
	m := &media.Media{ID: "m1", Name: "a.png", Mime: "image/png", Size: 3, CreatedBy: "u1", CreatedAt: t0}
	if err := s.Media().Save(ctx, m); err != nil {
		t.Fatal(err)
	}
	if got, err := s.Media().Get(ctx, "m1"); err != nil || got.Mime != "image/png" {
		t.Errorf("get: %+v %v", got, err)
	}
	if n, _ := s.Media().CountExisting(ctx, []string{"m1", "x"}); n != 1 {
		t.Errorf("count = %d", n)
	}
}

func TestNextSequence(t *testing.T) {
	s := setup(t)
	for want := int64(1); want <= 3; want++ {
		if n, err := s.NextSequence(t.Context(), "asset/code/TGD-"); err != nil || n != want {
			t.Fatalf("seq = %d %v, mau %d", n, err, want)
		}
	}
	if n, _ := s.NextSequence(t.Context(), "asset/code/TRF-"); n != 1 {
		t.Errorf("kunci lain mulai 1: %d", n)
	}
}
