// Package memory: adapter in-memory untuk semua port dynamic. Dipakai test & dev tanpa Mongo/ES.
package memory

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"

	"eam/services/common/errs"
	"eam/services/dynamic/internal/domain/content"
	"eam/services/dynamic/internal/domain/media"
)

// ---------- schema ----------

type Schemas struct {
	mu    sync.Mutex
	cts   map[string]content.ContentType
	comps map[string]content.Component
}

func NewSchemas() *Schemas {
	return &Schemas{cts: map[string]content.ContentType{}, comps: map[string]content.Component{}}
}

func (s *Schemas) ListContentTypes(context.Context) ([]*content.ContentType, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []*content.ContentType{}
	for _, ct := range s.cts {
		out = append(out, &ct)
	}
	slices.SortFunc(out, func(a, b *content.ContentType) int { return strings.Compare(a.Label, b.Label) })
	return out, nil
}

func (s *Schemas) GetContentType(_ context.Context, uid string) (*content.ContentType, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ct, ok := s.cts[uid]
	if !ok {
		return nil, errs.NotFound("content type")
	}
	return &ct, nil
}

func (s *Schemas) SaveContentType(_ context.Context, ct *content.ContentType) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cts[ct.UID] = *ct
	return nil
}

func (s *Schemas) DeleteContentType(_ context.Context, uid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cts, uid)
	return nil
}

func (s *Schemas) ListComponents(context.Context) ([]*content.Component, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []*content.Component{}
	for _, c := range s.comps {
		out = append(out, &c)
	}
	slices.SortFunc(out, func(a, b *content.Component) int { return strings.Compare(a.Label, b.Label) })
	return out, nil
}

func (s *Schemas) GetComponent(_ context.Context, uid string) (*content.Component, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.comps[uid]
	if !ok {
		return nil, errs.NotFound("component")
	}
	return &c, nil
}

func (s *Schemas) SaveComponent(_ context.Context, c *content.Component) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.comps[c.UID] = *c
	return nil
}

func (s *Schemas) DeleteComponent(_ context.Context, uid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.comps, uid)
	return nil
}

// ---------- entries ----------

type Entries struct {
	mu        sync.Mutex
	data      map[string]map[string]content.Entry // uid -> id -> entry
	unique    map[string][]string
	revisions map[string][]content.Revision // uid/id -> revisi
	seq       map[string]int64
}

func NewEntries() *Entries {
	return &Entries{data: map[string]map[string]content.Entry{}, unique: map[string][]string{}, revisions: map[string][]content.Revision{}, seq: map[string]int64{}}
}

func (r *Entries) PeekSequence(_ context.Context, key string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.seq[key] + 1, nil
}

func (r *Entries) NextSequence(_ context.Context, key string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq[key]++
	return r.seq[key], nil
}

func (r *Entries) Get(_ context.Context, uid, id string) (*content.Entry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.data[uid][id]
	if !ok {
		return nil, errs.NotFound("data")
	}
	return &e, nil
}

func (r *Entries) Save(_ context.Context, uid string, e *content.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, f := range r.unique[uid] {
		for id, o := range r.data[uid] {
			// unique hanya antar entri yang belum dihapus
			if id != e.ID && !e.Deleted && !o.Deleted && e.Data[f] != nil && o.Data[f] == e.Data[f] {
				return errs.Conflict("nilai %s sudah dipakai entri lain", f)
			}
		}
	}
	if r.data[uid] == nil {
		r.data[uid] = map[string]content.Entry{}
	}
	r.data[uid][e.ID] = *e
	return nil
}

func (r *Entries) All(_ context.Context, uid string) ([]*content.Entry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []*content.Entry{}
	for _, e := range r.data[uid] {
		if !e.Deleted {
			out = append(out, &e)
		}
	}
	slices.SortFunc(out, func(a, b *content.Entry) int { return a.CreatedAt.Compare(b.CreatedAt) })
	return out, nil
}

func (r *Entries) CountExisting(_ context.Context, uid string, ids []string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, id := range ids {
		if e, ok := r.data[uid][id]; ok && !e.Deleted {
			n++
		}
	}
	return n, nil
}

func (r *Entries) FindOne(_ context.Context, uid, field string, value any) (*content.Entry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.data[uid] {
		if !e.Deleted && e.Data[field] == value {
			return &e, nil
		}
	}
	return nil, errs.NotFound("data")
}

func (r *Entries) AddRevision(_ context.Context, uid string, rev *content.Revision) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := uid + "/" + rev.EntryID
	r.revisions[k] = append(r.revisions[k], *rev)
	return nil
}

func (r *Entries) Revisions(_ context.Context, uid, id string) ([]*content.Revision, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []*content.Revision{}
	for _, rev := range r.revisions[uid+"/"+id] {
		out = append(out, &rev)
	}
	return out, nil
}

func (r *Entries) SyncUnique(_ context.Context, uid string, fields []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, f := range fields {
		seen := map[any]bool{}
		for _, e := range r.data[uid] {
			if v := e.Data[f]; v != nil && !e.Deleted {
				if seen[v] {
					return errs.Conflict("field %s tidak bisa dibuat unique: sudah ada nilai duplikat", f)
				}
				seen[v] = true
			}
		}
	}
	r.unique[uid] = fields
	return nil
}

func (r *Entries) Drop(_ context.Context, uid string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.data, uid)
	delete(r.unique, uid)
	return nil
}

// ---------- search index ----------

// Index: pencarian substring sederhana atas semua nilai teks. Fail = simulasi Elasticsearch mati.
type Index struct {
	mu       sync.Mutex
	docs     map[string]map[string]content.Entry
	rebuilds map[string]int
	Fail     bool
}

func NewIndex() *Index {
	return &Index{docs: map[string]map[string]content.Entry{}, rebuilds: map[string]int{}}
}

var errDown = errors.New("index down")

func (x *Index) Exists(uid string) bool {
	x.mu.Lock()
	defer x.mu.Unlock()
	_, ok := x.docs[uid]
	return ok
}

func (x *Index) Rebuilds(uid string) int {
	x.mu.Lock()
	defer x.mu.Unlock()
	return x.rebuilds[uid]
}

func (x *Index) Rebuild(_ context.Context, ct *content.ContentType, entries []*content.Entry) error {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.Fail {
		return errDown
	}
	x.docs[ct.UID] = map[string]content.Entry{}
	for _, e := range entries {
		x.docs[ct.UID][e.ID] = *e
	}
	x.rebuilds[ct.UID]++
	return nil
}

func (x *Index) Put(_ context.Context, ct *content.ContentType, e *content.Entry) error {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.Fail {
		return errDown
	}
	if x.docs[ct.UID] == nil {
		x.docs[ct.UID] = map[string]content.Entry{}
	}
	x.docs[ct.UID][e.ID] = *e
	return nil
}

func (x *Index) Delete(_ context.Context, uid, id string) error {
	x.mu.Lock()
	defer x.mu.Unlock()
	delete(x.docs[uid], id)
	return nil
}

func (x *Index) Drop(_ context.Context, uid string) error {
	x.mu.Lock()
	defer x.mu.Unlock()
	delete(x.docs, uid)
	return nil
}

func (x *Index) Search(_ context.Context, ct *content.ContentType, q content.Query) (content.Page, error) {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.Fail {
		return content.Page{}, errDown
	}
	var hits []*content.Entry
	for _, e := range x.docs[ct.UID] {
		if len(q.States) > 0 && !slices.Contains(q.States, e.Status) {
			continue
		}
		if q.Scope != nil {
			if v, _ := e.Data[q.Scope.Field].(string); !slices.Contains(q.Scope.Values, v) {
				continue
			}
		}
		if !content.Matches(&e, q.Filters) {
			continue
		}
		if q.Q == "" || strings.Contains(strings.ToLower(fmt.Sprint(e.Data)), strings.ToLower(q.Q)) {
			hits = append(hits, &e)
		}
	}
	slices.SortFunc(hits, func(a, b *content.Entry) int { return b.CreatedAt.Compare(a.CreatedAt) })
	from := min(max(q.Page-1, 0)*q.Size, len(hits))
	return content.Page{Total: len(hits), Items: hits[from:min(from+q.Size, len(hits))]}, nil
}

// ---------- media ----------

type Media struct {
	mu sync.Mutex
	m  map[string]media.Media
}

func NewMedia() *Media { return &Media{m: map[string]media.Media{}} }

func (r *Media) Save(_ context.Context, m *media.Media) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[m.ID] = *m
	return nil
}

func (r *Media) Get(_ context.Context, id string) (*media.Media, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.m[id]
	if !ok {
		return nil, errs.NotFound("file")
	}
	return &m, nil
}

func (r *Media) List(_ context.Context, ids []string) ([]*media.Media, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []*media.Media{}
	for _, id := range ids {
		if m, ok := r.m[id]; ok {
			out = append(out, &m)
		}
	}
	return out, nil
}

func (r *Media) CountExisting(ctx context.Context, ids []string) (int, error) {
	l, err := r.List(ctx, ids)
	return len(l), err
}

type Files struct {
	mu sync.Mutex
	m  map[string][]byte
}

func NewFiles() *Files { return &Files{m: map[string][]byte{}} }

func (f *Files) Count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.m)
}

func (f *Files) Write(id string, r io.Reader) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.m[id] = b
	return nil
}

func (f *Files) Open(id string) (io.ReadCloser, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.m[id]
	if !ok {
		return nil, errs.NotFound("file")
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

func (f *Files) Remove(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.m, id)
	return nil
}
