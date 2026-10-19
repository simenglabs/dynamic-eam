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

// scope: add a site filter to the query when the content type is restricted for the actor. false = actor without site access (empty result).
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

// Get: entries that were deleted (soft delete) or are outside site access are treated as nonexistent.
func (s *Entries) Get(ctx context.Context, uid, id string, by content.Actor) (*content.Entry, error) {
	ct, err := s.d.Schemas.GetContentType(ctx, uid)
	if err != nil {
		return nil, err
	}
	return s.visible(ctx, ct, id, by)
}

// visible: entry is active & within the actor's site access (outside access = NotFound, so its existence does not leak).
func (s *Entries) visible(ctx context.Context, ct *content.ContentType, id string, by content.Actor) (*content.Entry, error) {
	e, err := s.live(ctx, ct.UID, id)
	if err == nil && !ct.InScope(e, by) {
		return nil, errs.NotFound("record")
	}
	return e, err
}

func (s *Entries) live(ctx context.Context, uid, id string) (*content.Entry, error) {
	e, err := s.d.Entries.Get(ctx, uid, id)
	if err == nil && e.Deleted {
		return nil, errs.NotFound("record")
	}
	return e, err
}

// History: entry audit trail (create/update/delete/transition), ordered by version. Still readable after deletion.
func (s *Entries) History(ctx context.Context, uid, id string, by content.Actor) ([]*content.Revision, error) {
	ct, err := s.d.Schemas.GetContentType(ctx, uid)
	if err != nil {
		return nil, err
	}
	// The history of a deleted entry is still viewable, but still restricted by site.
	if e, err := s.d.Entries.Get(ctx, uid, id); err == nil && !ct.InScope(e, by) {
		return nil, errs.NotFound("record")
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

// Upsert: save an entry with an external ID (e.g. user profile = user ID); created if missing / previously deleted.
func (s *Entries) Upsert(ctx context.Context, uid, id string, data map[string]any, by content.Actor) (*content.Entry, error) {
	if !content.ExternalIDRe.MatchString(id) {
		return nil, errs.Invalid("id is invalid")
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
		e.Version = old.Version // recreated after deletion: version number continues
	}
	return s.save(ctx, ct, e, data, by, true)
}

// Update only changes data; status & history only change via Transition.
func (s *Entries) Update(ctx context.Context, uid, id string, data map[string]any, by content.Actor) (*content.Entry, error) {
	ct, e, err := s.editable(ctx, uid, id, by)
	if err != nil {
		return nil, err
	}
	return s.save(ctx, ct, e, data, by, false)
}

// editable loads the entry (within site access) and rejects it if the current status is locked by the workflow.
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
		return nil, nil, errs.Forbidden("records with status %s cannot be edited or deleted", e.Status)
	}
	return ct, e, nil
}

// Preview: auto fill field values for the current form data (before Save), computed the same way as save.
// Sequence numbers are only peeked, not consumed. Empty id = new record.
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
		return e, nil // no changes: version & history are not incremented
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

// commit: bump the version, save entry + revision, then sync to the index (unindex if deleted).
func (s *Entries) commit(ctx context.Context, ct *content.ContentType, e *content.Entry, rev *content.Revision, deleted bool) error {
	e.Version++
	rev.Version = e.Version
	if err := s.d.Entries.Save(ctx, ct.UID, e); err != nil {
		e.Version--
		return err
	}
	// ponytail: entry & revision are not in one Mongo transaction; use a session transaction if the audit must be 100% consistent.
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

// checkRefs: every referenced relation & media id must exist.
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
			return errs.Invalid("selected %s not found", what)
		}
		return nil
	}
	for target, ids := range r.Relations {
		err := check(ids, "relation "+target, func(ids []string) (int, error) { return s.d.Entries.CountExisting(ctx, target, ids) })
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
			return errs.Invalid("domain %s has not been created in Master › Domain", code)
		}
		if err != nil {
			return err
		}
		for _, id := range ids {
			if !slices.ContainsFunc(vals, func(v content.DomainValue) bool { return v.ID == id }) {
				return errs.Invalid("value %q does not exist or is inactive in domain %s", id, code)
			}
		}
	}
	return nil
}

// related: loads target entry data for auto fill from a relation (deleted entry = nonexistent).
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

// fillActorDefaults: on create, an empty relation to Sites gets the user's default site, to Organizations gets their organization.
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

// fillDomainDefaults: top-level domain fields left empty on create use the domain's isDefault value.
func (s *Entries) fillDomainDefaults(ctx context.Context, ct *content.ContentType, data map[string]any) (map[string]any, error) {
	out, copied := data, false
	for _, f := range ct.Fields {
		if f.Type != content.TDomain || (data[f.Name] != nil && data[f.Name] != "") {
			continue
		}
		vals, err := s.DomainValues(ctx, f.Domain)
		if errs.KindOf(err) == errs.KindNotFound {
			continue // reported during validation if the field is filled / required
		}
		if err != nil {
			return nil, err
		}
		if d := content.DefaultDomainValue(vals); d != "" {
			if !copied { // copy once so the caller's input is not modified
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

// DomainValues: active domain values (for dropdowns & validation).
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

// DomainCodes: list of domains (for choosing a domain in the builder).
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

// EnsureEntries: create missing built-in records (matched via keyField); existing ones are not overwritten.
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

// Delete: soft delete (isDeleted); the record stays stored & recorded in history.
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

// Options: choices for a relation field (label = display field). Max 100.
// where: field = value filter (relation filter in the form).
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

// All: schema + all entries (for CSV export).
func (s *Entries) All(ctx context.Context, uid string, by content.Actor) (*content.ContentType, []*content.Entry, error) {
	ct, err := s.d.Schemas.GetContentType(ctx, uid)
	if err != nil {
		return nil, nil, err
	}
	all, err := s.d.Entries.All(ctx, uid)
	return ct, slices.DeleteFunc(all, func(e *content.Entry) bool { return !ct.InScope(e, by) }), err
}

// Transition runs a workflow action (e.g. submit/approve) and records history. Permissions are checked in the http layer.
// data = transition form input (Transition.Fields), validated like entry fields and recorded in history.
func (s *Entries) Transition(ctx context.Context, uid, id, key string, by content.Actor, note string, data map[string]any) (*content.Entry, error) {
	ct, err := s.d.Schemas.GetContentType(ctx, uid)
	if err != nil {
		return nil, err
	}
	if ct.Workflow == nil {
		return nil, errs.NotFound("transition")
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

// SiteParentField: parent field in the Sites master (hierarchy, e.g. UID -> UP3 -> ULP).
const SiteParentField = "parent"

// ActorFor builds the actor from the user profile: site access (sites + defaultSite, inherited by all child sites),
// default site, organization, and profile data. No profile yet = no sites.
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

// withChildSites: site + all its descendants via the parent field of the Sites master.
// ponytail: loads every site on each request (fine for hundreds of sites); cache it if the site count grows large.
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
		if slices.Contains(out, id) { // also prevents parent cycles
			continue
		}
		out = append(out, id)
		queue = append(queue, children[id]...)
	}
	return out, nil
}
