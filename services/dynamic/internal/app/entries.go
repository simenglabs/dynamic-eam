package app

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"eam/services/common/errs"
	"eam/services/dynamic/internal/domain/content"
)

type Entries struct{ d Deps }

func NewEntries(d Deps) *Entries { return &Entries{d} }

type Query = content.Query

func (s *Entries) List(ctx context.Context, uid string, q Query, by content.Actor) (content.Page, error) {
	ct, err := s.d.Schemas.GetContentType(ctx, uid)
	if err != nil {
		return content.Page{}, err
	}
	q.Page, q.Size = max(q.Page, 1), min(max(q.Size, 1), 100)
	if !scope(ct, by, &q) {
		return content.Page{Items: []*content.Entry{}}, nil
	}
	page, err := s.d.Index.Search(ctx, ct, q)
	if err != nil {
		return content.Page{}, indexFailed(err)
	}
	return page, nil
}

// scope: tambahkan filter site ke query bila content type dibatasi untuk actor. false = actor tanpa site akses (hasil kosong).
func scope(ct *content.ContentType, by content.Actor, q *Query) bool {
	if !ct.Scoped(by) {
		return true
	}
	if len(by.Sites) == 0 {
		return false
	}
	q.Scope = &content.Scope{Field: ct.ScopeField, Values: by.Sites}
	return true
}

// Get: entri yang sudah dihapus (soft delete) atau di luar site akses dianggap tidak ada.
func (s *Entries) Get(ctx context.Context, uid, id string, by content.Actor) (*content.Entry, error) {
	ct, err := s.d.Schemas.GetContentType(ctx, uid)
	if err != nil {
		return nil, err
	}
	return s.visible(ctx, ct, id, by)
}

// visible: entri aktif & dalam site akses actor (di luar akses = NotFound, supaya keberadaannya tidak bocor).
func (s *Entries) visible(ctx context.Context, ct *content.ContentType, id string, by content.Actor) (*content.Entry, error) {
	e, err := s.live(ctx, ct.UID, id)
	if err == nil && !ct.InScope(e, by) {
		return nil, errs.NotFound("data")
	}
	return e, err
}

func (s *Entries) live(ctx context.Context, uid, id string) (*content.Entry, error) {
	e, err := s.d.Entries.Get(ctx, uid, id)
	if err == nil && e.Deleted {
		return nil, errs.NotFound("data")
	}
	return e, err
}

// History: audit trail entri (create/update/delete/transisi), urut versi. Tetap bisa dibaca setelah dihapus.
func (s *Entries) History(ctx context.Context, uid, id string, by content.Actor) ([]*content.Revision, error) {
	ct, err := s.d.Schemas.GetContentType(ctx, uid)
	if err != nil {
		return nil, err
	}
	// Entri terhapus tetap boleh dilihat riwayatnya, tapi tetap dibatasi site.
	if e, err := s.d.Entries.Get(ctx, uid, id); err == nil && !ct.InScope(e, by) {
		return nil, errs.NotFound("data")
	}
	return s.d.Entries.Revisions(ctx, uid, id)
}

func (s *Entries) Create(ctx context.Context, uid string, data map[string]any, by content.Actor) (*content.Entry, error) {
	ct, err := s.d.Schemas.GetContentType(ctx, uid)
	if err != nil {
		return nil, err
	}
	e := &content.Entry{ID: s.d.NewID(), CreatedAt: s.d.Now(), CreatedBy: by.ID, Status: ct.InitialStatus()}
	return s.save(ctx, ct, e, data, by, true)
}

// Upsert: simpan entri dengan ID dari luar (mis. profil user = ID user); dibuat bila belum ada / pernah dihapus.
func (s *Entries) Upsert(ctx context.Context, uid, id string, data map[string]any, by content.Actor) (*content.Entry, error) {
	if !content.ExternalIDRe.MatchString(id) {
		return nil, errs.Invalid("id tidak valid")
	}
	ct, err := s.d.Schemas.GetContentType(ctx, uid)
	if err != nil {
		return nil, err
	}
	old, err := s.d.Entries.Get(ctx, uid, id)
	switch {
	case err == nil && !old.Deleted:
		return s.Update(ctx, uid, id, data, by)
	case err != nil && errs.KindOf(err) != errs.KindNotFound:
		return nil, err
	}
	e := &content.Entry{ID: id, CreatedAt: s.d.Now(), CreatedBy: by.ID, Status: ct.InitialStatus()}
	if old != nil {
		e.Version = old.Version // dibuat ulang setelah dihapus: nomor versi berlanjut
	}
	return s.save(ctx, ct, e, data, by, true)
}

// Update hanya mengubah data; status & riwayat hanya berubah lewat Transition.
func (s *Entries) Update(ctx context.Context, uid, id string, data map[string]any, by content.Actor) (*content.Entry, error) {
	ct, e, err := s.editable(ctx, uid, id, by)
	if err != nil {
		return nil, err
	}
	return s.save(ctx, ct, e, data, by, false)
}

// editable memuat entri (dalam site akses) dan menolak bila status saat ini terkunci workflow.
func (s *Entries) editable(ctx context.Context, uid, id string, by content.Actor) (*content.ContentType, *content.Entry, error) {
	ct, err := s.d.Schemas.GetContentType(ctx, uid)
	if err != nil {
		return nil, nil, err
	}
	e, err := s.visible(ctx, ct, id, by)
	if err != nil {
		return nil, nil, err
	}
	if !ct.Editable(e.Status) {
		return nil, nil, errs.Forbidden("data berstatus %s tidak bisa diubah atau dihapus", e.Status)
	}
	return ct, e, nil
}

// Preview: nilai field isi otomatis untuk data form saat ini (sebelum Save), dihitung sama seperti save.
// Nomor urut hanya diintip, tidak dipakai. id kosong = data baru.
func (s *Entries) Preview(ctx context.Context, uid, id string, data map[string]any, by content.Actor) (map[string]any, error) {
	ct, err := s.d.Schemas.GetContentType(ctx, uid)
	if err != nil {
		return nil, err
	}
	var old map[string]any
	if id != "" {
		e, err := s.visible(ctx, ct, id, by)
		if err != nil {
			return nil, err
		}
		old = e.Data
	}
	src := content.AutoFillSources{Related: s.related(ctx, ct), Profile: by.Profile, Actor: by, Now: s.d.Now()}
	out, err := ct.ApplyPatterns(ct.ApplyAutoFill(data, src), old, src, func(key string) (int64, error) { return s.d.Entries.PeekSequence(ctx, key) })
	if err != nil {
		return nil, err
	}
	res := map[string]any{}
	for _, f := range ct.Fields {
		if f.AutoFill != nil {
			res[f.Name] = out[f.Name]
		}
	}
	return res, nil
}

func (s *Entries) save(ctx context.Context, ct *content.ContentType, e *content.Entry, data map[string]any, by content.Actor, creating bool) (*content.Entry, error) {
	cat, err := catalog(ctx, s.d)
	if err != nil {
		return nil, err
	}
	if creating {
		if data, err = s.fillDomainDefaults(ctx, ct, data); err != nil {
			return nil, err
		}
		data = fillActorDefaults(ct, data, by)
	}
	src := content.AutoFillSources{Related: s.related(ctx, ct), Profile: by.Profile, Actor: by, Now: s.d.Now()}
	data = ct.ApplyAutoFill(data, src)
	if data, err = ct.ApplyPatterns(data, e.Data, src, func(key string) (int64, error) { return s.d.Entries.NextSequence(ctx, key) }); err != nil {
		return nil, err
	}
	clean, refs, err := content.ValidateEntry(ct.Fields, data, cat.Components, creating)
	if err != nil {
		return nil, err
	}
	if err := s.applyAttributes(ctx, ct, clean); err != nil {
		return nil, err
	}
	if err := s.checkWhere(ctx, ct, clean); err != nil {
		return nil, err
	}
	if err := ct.CheckScope(clean, by); err != nil {
		return nil, err
	}
	if err := s.ensureProfile(ctx, ct, by); err != nil {
		return nil, err
	}
	if err := s.checkRefs(ctx, refs); err != nil {
		return nil, err
	}
	changes := content.Diff(e.Data, clean)
	if !creating && len(changes) == 0 {
		return e, nil // tidak ada perubahan: versi & riwayat tidak bertambah
	}
	action := content.ActionUpdate
	if creating {
		action = content.ActionCreate
	}
	e.Data, e.UpdatedAt, e.UpdatedBy = clean, s.d.Now(), by.ID
	if err := s.commit(ctx, ct, e, content.NewRevision(e, action, by, e.UpdatedAt, changes, ""), false); err != nil {
		if errs.KindOf(err) == errs.KindUnavailable {
			return e, err
		}
		return nil, err
	}
	return e, nil
}

// commit: naikkan versi, simpan entri + revisi, lalu sinkron ke index (unindex bila terhapus).
func (s *Entries) commit(ctx context.Context, ct *content.ContentType, e *content.Entry, rev *content.Revision, deleted bool) error {
	e.Version++
	rev.Version = e.Version
	if err := s.d.Entries.Save(ctx, ct.UID, e); err != nil {
		e.Version--
		return err
	}
	// ponytail: entri & revisi tidak satu transaksi Mongo; pakai session transaction kalau audit harus 100% konsisten.
	if err := s.d.Entries.AddRevision(ctx, ct.UID, rev); err != nil {
		return err
	}
	var err error
	if deleted {
		err = s.d.Index.Delete(ctx, ct.UID, e.ID)
	} else {
		err = s.d.Index.Put(ctx, ct, e)
	}
	if err != nil {
		return indexFailed(err)
	}
	return nil
}

// checkRefs: semua id relation & media yang dirujuk harus ada.
func (s *Entries) checkRefs(ctx context.Context, r content.Refs) error {
	check := func(ids []string, what string, count func([]string) (int, error)) error {
		if len(ids) == 0 {
			return nil
		}
		ids = slices.Compact(slices.Sorted(slices.Values(ids)))
		n, err := count(ids)
		if err != nil {
			return err
		}
		if n != len(ids) {
			return errs.Invalid("%s yang dipilih tidak ditemukan", what)
		}
		return nil
	}
	for target, ids := range r.Relations {
		err := check(ids, "relasi "+target, func(ids []string) (int, error) { return s.d.Entries.CountExisting(ctx, target, ids) })
		if err != nil {
			return err
		}
	}
	if err := check(r.Media, "file", func(ids []string) (int, error) { return s.d.MediaRepo.CountExisting(ctx, ids) }); err != nil {
		return err
	}
	for code, ids := range r.Domains {
		vals, err := s.DomainValues(ctx, code)
		if errs.KindOf(err) == errs.KindNotFound {
			return errs.Invalid("domain %s belum dibuat di Master › Domain", code)
		}
		if err != nil {
			return err
		}
		for _, id := range ids {
			if !slices.ContainsFunc(vals, func(v content.DomainValue) bool { return v.ID == id }) {
				return errs.Invalid("nilai %q tidak ada atau tidak aktif di domain %s", id, code)
			}
		}
	}
	return nil
}

// related: pemuat data entri target untuk isi otomatis dari relasi (entri terhapus = tidak ada).
func (s *Entries) related(ctx context.Context, ct *content.ContentType) func(field, id string) map[string]any {
	return func(field, id string) map[string]any {
		i := slices.IndexFunc(ct.Fields, func(f content.Field) bool { return f.Name == field })
		if i < 0 {
			return nil
		}
		e, err := s.live(ctx, ct.Fields[i].Target, id)
		if err != nil {
			return nil
		}
		return e.Data
	}
}

// fillActorDefaults: saat create, relasi kosong ke Sites diisi default site user, ke Organizations diisi organisasinya.
func fillActorDefaults(ct *content.ContentType, data map[string]any, by content.Actor) map[string]any {
	out := maps.Clone(data)
	if out == nil {
		out = map[string]any{}
	}
	for _, f := range ct.Fields {
		if f.Type != content.TRelation || f.Multiple || (out[f.Name] != nil && out[f.Name] != "") {
			continue
		}
		switch {
		case f.Target == content.SiteUID && by.DefaultSite != "":
			out[f.Name] = by.DefaultSite
		case f.Target == content.OrganizationUID && by.Organization != "":
			out[f.Name] = by.Organization
		}
	}
	return out
}

// fillDomainDefaults: field domain (level atas) yang tidak diisi saat create memakai value isDefault domain.
func (s *Entries) fillDomainDefaults(ctx context.Context, ct *content.ContentType, data map[string]any) (map[string]any, error) {
	out, copied := data, false
	for _, f := range ct.Fields {
		if f.Type != content.TDomain || (data[f.Name] != nil && data[f.Name] != "") {
			continue
		}
		vals, err := s.DomainValues(ctx, f.Domain)
		if errs.KindOf(err) == errs.KindNotFound {
			continue // dilaporkan saat validasi bila field diisi / required
		}
		if err != nil {
			return nil, err
		}
		if d := content.DefaultDomainValue(vals); d != "" {
			if !copied { // salin sekali supaya input pemanggil tidak berubah
				out, copied = maps.Clone(data), true
				if out == nil {
					out = map[string]any{}
				}
			}
			out[f.Name] = d
		}
	}
	return out, nil
}

// DomainValues: value aktif domain (untuk dropdown & validasi).
func (s *Entries) DomainValues(ctx context.Context, code string) ([]content.DomainValue, error) {
	e, err := s.d.Entries.FindOne(ctx, content.DomainUID, content.DomainCodeField, code)
	if err != nil {
		if errs.KindOf(err) == errs.KindNotFound {
			return nil, errs.NotFound("domain " + code)
		}
		return nil, err
	}
	return content.DomainValuesOf(e), nil
}

type DomainCode struct{ Code, Description string }

// DomainCodes: daftar domain (untuk memilih domain di builder).
func (s *Entries) DomainCodes(ctx context.Context) ([]DomainCode, error) {
	all, err := s.d.Entries.All(ctx, content.DomainUID)
	if err != nil {
		return nil, err
	}
	out := make([]DomainCode, 0, len(all))
	for _, e := range all {
		code, _ := e.Data[content.DomainCodeField].(string)
		desc, _ := e.Data["description"].(string)
		out = append(out, DomainCode{code, desc})
	}
	slices.SortFunc(out, func(a, b DomainCode) int { return strings.Compare(a.Code, b.Code) })
	return out, nil
}

// EnsureEntries: buat data bawaan yang belum ada (dicocokkan lewat keyField); yang sudah ada tidak ditimpa.
func (s *Entries) EnsureEntries(ctx context.Context, uid, keyField string, rows []map[string]any, by content.Actor) (created []string, err error) {
	var unavailable error
	for _, row := range rows {
		key := row[keyField]
		if _, err := s.d.Entries.FindOne(ctx, uid, keyField, key); err == nil {
			continue
		} else if errs.KindOf(err) != errs.KindNotFound {
			return created, err
		}
		if _, err := s.Create(ctx, uid, row, by); err != nil {
			if errs.KindOf(err) != errs.KindUnavailable {
				return created, err
			}
			unavailable = err
		}
		created = append(created, fmt.Sprint(key))
	}
	return created, unavailable
}

// Delete: soft delete (isDeleted), data tetap tersimpan & tercatat di riwayat.
func (s *Entries) Delete(ctx context.Context, uid, id string, by content.Actor) error {
	ct, e, err := s.editable(ctx, uid, id, by)
	if err != nil {
		return err
	}
	now := s.d.Now()
	e.Deleted, e.DeletedAt, e.DeletedBy, e.UpdatedAt, e.UpdatedBy = true, now, by.ID, now, by.ID
	return s.commit(ctx, ct, e, content.NewRevision(e, content.ActionDelete, by, now, nil, ""), true)
}

type Option struct{ ID, Label string }

// Options: pilihan untuk field relation (label = display field). Maks 100.
// where: filter field = nilai (filter relasi di form).
func (s *Entries) Options(ctx context.Context, uid, q string, where map[string]string, by content.Actor) ([]Option, error) {
	ct, err := s.d.Schemas.GetContentType(ctx, uid)
	if err != nil {
		return nil, err
	}
	query := content.Query{Q: q, Page: 1, Size: 100, Sort: ct.DisplaySortField() + ":asc"}
	for _, k := range slices.Sorted(maps.Keys(where)) {
		query.Filters = append(query.Filters, content.Scope{Field: k, Values: []string{where[k]}})
	}
	if !scope(ct, by, &query) {
		return []Option{}, nil
	}
	page, err := s.d.Index.Search(ctx, ct, query)
	if err != nil {
		return nil, indexFailed(err)
	}
	out := make([]Option, len(page.Items))
	for i, e := range page.Items {
		out[i] = Option{ID: e.ID, Label: ct.DisplayLabel(e.Data)}
	}
	return out, nil
}

// All: schema + semua entri (untuk export CSV).
func (s *Entries) All(ctx context.Context, uid string, by content.Actor) (*content.ContentType, []*content.Entry, error) {
	ct, err := s.d.Schemas.GetContentType(ctx, uid)
	if err != nil {
		return nil, nil, err
	}
	all, err := s.d.Entries.All(ctx, uid)
	return ct, slices.DeleteFunc(all, func(e *content.Entry) bool { return !ct.InScope(e, by) }), err
}

// Transition menjalankan aksi workflow (mis. submit/approve) dan mencatat riwayat. Izin dicek di layer http.
// data = isian form transisi (Transition.Fields), divalidasi seperti field entri dan dicatat di riwayat.
func (s *Entries) Transition(ctx context.Context, uid, id, key string, by content.Actor, note string, data map[string]any) (*content.Entry, error) {
	ct, err := s.d.Schemas.GetContentType(ctx, uid)
	if err != nil {
		return nil, err
	}
	if ct.Workflow == nil {
		return nil, errs.NotFound("transisi")
	}
	e, err := s.visible(ctx, ct, id, by)
	if err != nil {
		return nil, err
	}
	t, err := ct.Workflow.Transition(key, e.Status)
	if err != nil {
		return nil, err
	}
	cat, err := catalog(ctx, s.d)
	if err != nil {
		return nil, err
	}
	inputs, refs, err := content.ValidateEntry(t.Fields, data, cat.Components, true)
	if err != nil {
		return nil, err
	}
	if err := s.checkRefs(ctx, refs); err != nil {
		return nil, err
	}
	now := s.d.Now()
	change := content.Change{Field: "workflowStatus", From: e.Status, To: t.To}
	e.History = append(e.History, content.StatusChange{From: e.Status, To: t.To, Transition: t.Key, By: by.ID, At: now, Note: note})
	e.Status, e.UpdatedAt, e.UpdatedBy = t.To, now, by.ID
	rev := content.NewRevision(e, content.ActionTransition, by, now, []content.Change{change}, note)
	rev.Transition = t.Key
	if len(inputs) > 0 {
		rev.Data = inputs
	}
	if err := s.commit(ctx, ct, e, rev, false); err != nil {
		if errs.KindOf(err) == errs.KindUnavailable {
			return e, err
		}
		return nil, err
	}
	return e, nil
}

// SiteParentField: field induk di master Sites (hierarki, mis. UID -> UP3 -> ULP).
const SiteParentField = "parent"

// ActorFor membentuk actor dari profil user: site akses (sites + defaultSite, diturunkan ke semua site anak),
// default site, organisasi, dan data profil. Belum punya profil = tanpa site.
func (s *Entries) ActorFor(ctx context.Context, userID, name string, super bool) (content.Actor, error) {
	a := content.Actor{ID: userID, Name: name, Super: super}
	e, err := s.d.Entries.Get(ctx, content.ProfileUID, userID)
	if errs.KindOf(err) == errs.KindNotFound || (err == nil && e.Deleted) {
		return a, nil
	}
	if err != nil {
		return a, err
	}
	a.Profile = e.Data
	var sites []string
	switch list := e.Data[content.ProfileSites].(type) {
	case []any:
		for _, x := range list {
			if v, ok := x.(string); ok {
				sites = append(sites, v)
			}
		}
	case []string:
		sites = append(sites, list...)
	}
	a.DefaultSite, _ = e.Data[content.ProfileDefaultSite].(string)
	a.Organization, _ = e.Data[content.ProfileOrganization].(string)
	if a.DefaultSite != "" {
		sites = append(sites, a.DefaultSite)
	}
	a.Sites, err = s.withChildSites(ctx, sites)
	return a, err
}

// withChildSites: site + semua turunannya lewat field parent master Sites.
// ponytail: memuat semua site tiap request (cukup untuk ratusan site); cache kalau jumlah site besar.
func (s *Entries) withChildSites(ctx context.Context, sites []string) ([]string, error) {
	if len(sites) == 0 {
		return nil, nil
	}
	all, err := s.d.Entries.All(ctx, content.SiteUID)
	if errs.KindOf(err) == errs.KindNotFound {
		return sites, nil
	}
	if err != nil {
		return nil, err
	}
	children := map[string][]string{}
	for _, site := range all {
		if p, _ := site.Data[SiteParentField].(string); p != "" {
			children[p] = append(children[p], site.ID)
		}
	}
	out := []string{}
	queue := slices.Clone(sites)
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if slices.Contains(out, id) { // juga mencegah siklus parent
			continue
		}
		out = append(out, id)
		queue = append(queue, children[id]...)
	}
	return out, nil
}
