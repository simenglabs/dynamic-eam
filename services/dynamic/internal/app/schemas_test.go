package app

import (
	"testing"

	"eam/services/common/errs"
	"eam/services/dynamic/internal/domain/content"
)

func TestCreateContentTypeValidatesAndIndexes(t *testing.T) {
	e := newEnv()
	ct, err := e.schemas.CreateContentType(e.ctx, locationCT())
	if err != nil {
		t.Fatal(err)
	}
	if !ct.CreatedAt.Equal(t0) || !e.index.Exists("location") {
		t.Errorf("timestamp / index belum dibuat: %+v", ct)
	}
	if _, err := e.schemas.CreateContentType(e.ctx, locationCT()); errs.KindOf(err) != errs.KindConflict {
		t.Errorf("UID duplikat: %v", err)
	}
	bad := locationCT()
	bad.UID = "lain"
	bad.DisplayField = "nope"
	if _, err := e.schemas.CreateContentType(e.ctx, bad); errs.KindOf(err) != errs.KindInvalid {
		t.Errorf("schema invalid: %v", err)
	}
}

func TestUpdateContentTypeKeepsUIDAndReindexes(t *testing.T) {
	e := newEnv()
	e.seed()
	upd := vehicleCT()
	upd.UID = "diabaikan"
	upd.Label = "Kendaraan Dinas"
	upd.Layout = &content.Layout{Sections: []content.Section{{ID: "a", Title: "Utama", Rows: []content.Row{{Columns: 2, Fields: []string{"plate", "status"}}}}}}
	ct, err := e.schemas.UpdateContentType(e.ctx, "vehicle", upd)
	if err != nil {
		t.Fatal(err)
	}
	if ct.UID != "vehicle" || ct.Label != "Kendaraan Dinas" || ct.Layout == nil {
		t.Errorf("update: %+v", ct)
	}
	if e.index.Rebuilds("vehicle") != 2 {
		t.Errorf("update harus rebuild index (create + update), got %d", e.index.Rebuilds("vehicle"))
	}
	if _, err := e.schemas.UpdateContentType(e.ctx, "nope", vehicleCT()); errs.KindOf(err) != errs.KindNotFound {
		t.Errorf("update tidak ada: %v", err)
	}
}

func TestIndexFailureIsUnavailableButSaved(t *testing.T) {
	e := newEnv()
	e.index.Fail = true
	_, err := e.schemas.CreateContentType(e.ctx, locationCT())
	if errs.KindOf(err) != errs.KindUnavailable {
		t.Fatalf("err = %v", err)
	}
	if _, err := e.schemas.GetContentType(e.ctx, "location"); err != nil {
		t.Error("schema tetap tersimpan walau index gagal")
	}
}

func TestDeleteContentType(t *testing.T) {
	e := newEnv()
	e.seed()
	if err := e.schemas.DeleteContentType(e.ctx, "location"); errs.KindOf(err) != errs.KindConflict {
		t.Errorf("location dipakai relation vehicle: %v", err)
	}
	if err := e.schemas.DeleteContentType(e.ctx, "vehicle"); err != nil {
		t.Fatal(err)
	}
	if e.index.Exists("vehicle") {
		t.Error("index harus di-drop")
	}
	if err := e.schemas.DeleteContentType(e.ctx, "vehicle"); errs.KindOf(err) != errs.KindNotFound {
		t.Errorf("hapus dua kali: %v", err)
	}
}

func TestComponentLifecycle(t *testing.T) {
	e := newEnv()
	addr := &content.Component{UID: "address", Label: "Alamat", Fields: []content.Field{{Name: "street", Label: "Jalan", Type: content.TText}}}
	if _, err := e.schemas.CreateComponent(e.ctx, addr); err != nil {
		t.Fatal(err)
	}
	ct := locationCT()
	ct.Fields = append(ct.Fields, content.Field{Name: "addr", Label: "Alamat", Type: content.TComponent, Component: "address"})
	if _, err := e.schemas.CreateContentType(e.ctx, ct); err != nil {
		t.Fatal(err)
	}
	if err := e.schemas.DeleteComponent(e.ctx, "address"); errs.KindOf(err) != errs.KindConflict {
		t.Errorf("component dipakai: %v", err)
	}
	upd := &content.Component{Label: "Alamat Baru", Fields: addr.Fields}
	if c, err := e.schemas.UpdateComponent(e.ctx, "address", upd); err != nil || c.UID != "address" || !c.CreatedAt.Equal(t0) {
		t.Errorf("update component: %+v %v", c, err)
	}
}
