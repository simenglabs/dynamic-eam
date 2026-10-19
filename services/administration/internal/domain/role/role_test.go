package role

import (
	"testing"
	"time"

	"eam/services/common/errs"
)

var now = time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)

func TestNewNormalizesPermissions(t *testing.T) {
	r, err := New("teknisi", " Teknisi ", "Lapangan", Permissions{"assets/tnd": {"read", "download", "read"}, "renkon/prk": {}}, now)
	if err != nil {
		t.Fatal(err)
	}
	if r.Name != "Teknisi" || len(r.Permissions["assets/tnd"]) != 2 {
		t.Errorf("normalisasi salah: %+v", r)
	}
	if _, ok := r.Permissions["renkon/prk"]; ok {
		t.Error("menu tanpa aksi harus dibuang")
	}
}

func TestNewRejectsInvalid(t *testing.T) {
	cases := map[string]func() (*Role, error){
		"id salah":    func() (*Role, error) { return New("Bad ID", "x", "", nil, now) },
		"nama kosong": func() (*Role, error) { return New("ok", " ", "", nil, now) },
		"aksi asing":  func() (*Role, error) { return New("ok", "x", "", Permissions{"assets/tnd": {"approve"}}, now) },
		"kunci asing": func() (*Role, error) { return New("ok", "x", "", Permissions{"../etc": {"read"}}, now) },
	}
	for name, f := range cases {
		if _, err := f(); errs.KindOf(err) != errs.KindInvalid {
			t.Errorf("%s: err = %v, want invalid", name, err)
		}
	}
}

func TestWildcardKeyAllowed(t *testing.T) {
	if _, err := New("ok", "x", "", Permissions{AllMenus: {"read"}}, now); err != nil {
		t.Fatal(err)
	}
}

func TestSystemRoleImmutable(t *testing.T) {
	for _, r := range Defaults(now) {
		if r.ID != SuperAdmin {
			continue
		}
		if err := r.Edit("Hack", "", nil, now); errs.KindOf(err) != errs.KindForbidden {
			t.Errorf("edit role sistem: %v", err)
		}
		if err := r.CanDelete(0); errs.KindOf(err) != errs.KindForbidden {
			t.Errorf("hapus role sistem: %v", err)
		}
		return
	}
	t.Fatal("Defaults harus berisi superadmin")
}

func TestCanDeleteInUse(t *testing.T) {
	r, _ := New("teknisi", "Teknisi", "", nil, now)
	if errs.KindOf(r.CanDelete(2)) != errs.KindConflict {
		t.Error("role dipakai user harus ditolak dihapus")
	}
	if r.CanDelete(0) != nil {
		t.Error("role tidak dipakai boleh dihapus")
	}
}

func TestEditUpdatesFields(t *testing.T) {
	r, _ := New("teknisi", "Teknisi", "", nil, now)
	later := now.Add(time.Hour)
	if err := r.Edit("Teknisi Senior", "baru", Permissions{"assets/tnd": {"read"}}, later); err != nil {
		t.Fatal(err)
	}
	if r.Name != "Teknisi Senior" || r.Description != "baru" || len(r.Permissions) != 1 || !r.UpdatedAt.Equal(later) {
		t.Errorf("edit salah: %+v", r)
	}
}

func TestDefaults(t *testing.T) {
	ids := map[string]bool{}
	for _, r := range Defaults(now) {
		ids[r.ID] = true
	}
	if !ids[SuperAdmin] || !ids["admin"] || !ids["user"] {
		t.Errorf("defaults: %v", ids)
	}
}
