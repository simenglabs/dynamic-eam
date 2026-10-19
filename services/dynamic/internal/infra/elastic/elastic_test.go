package elastic

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"eam/services/dynamic/internal/domain/content"
)

// fakeES mencatat request dan membalas respons search tetap.
type fakeES struct {
	mu   sync.Mutex
	reqs []string // "METHOD path body"
}

func (f *fakeES) handler(w http.ResponseWriter, r *http.Request) {
	b, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	f.reqs = append(f.reqs, r.Method+" "+r.URL.RequestURI()+" "+string(b))
	f.mu.Unlock()
	switch {
	case strings.HasSuffix(r.URL.Path, "/_search"):
		w.Write([]byte(`{"hits":{"total":{"value":1},"hits":[{"_id":"e1","_source":{"plate":"L 1","workflowStatus":"draft","createdAt":"2026-10-19T10:00:00Z","createdBy":"u1"}}]}}`))
	case r.URL.Path == "/_bulk":
		w.Write([]byte(`{"errors":false}`))
	default:
		w.Write([]byte(`{}`))
	}
}

func (f *fakeES) find(prefix string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.reqs {
		if strings.HasPrefix(r, prefix) {
			return r
		}
	}
	return ""
}

func setup(t *testing.T) (*Index, *fakeES) {
	f := &fakeES{}
	srv := httptest.NewServer(http.HandlerFunc(f.handler))
	t.Cleanup(srv.Close)
	return New(srv.URL, ""), f
}

var ct = &content.ContentType{UID: "vehicle", DisplayField: "plate", Fields: []content.Field{
	{Name: "plate", Type: content.TText}, {Name: "notes", Type: content.TLongText}, {Name: "year", Type: content.TNumber},
	{Name: "status", Type: content.TSelect}, {Name: "addr", Type: content.TComponent},
}}

func TestRebuildCreatesMappingAndBulkIndexes(t *testing.T) {
	x, f := setup(t)
	e := &content.Entry{ID: "e1", Data: map[string]any{"plate": "L 1"}, Status: "draft", CreatedAt: time.Now()}
	if err := x.Rebuild(context.Background(), ct, []*content.Entry{e}); err != nil {
		t.Fatal(err)
	}
	if f.find("DELETE /eam-ct-vehicle") == "" {
		t.Error("index lama harus dihapus dulu")
	}
	put := f.find("PUT /eam-ct-vehicle ")
	for _, want := range []string{`"plate":{"fields":{"keyword"`, `"year":{"type":"double"}`, `"status":{"type":"keyword"}`, `"addr":{"enabled":false`, `"workflowStatus":{"type":"keyword"}`, `"dynamic":false`} {
		if !strings.Contains(put, want) {
			t.Errorf("mapping tanpa %s: %s", want, put)
		}
	}
	bulk := f.find("POST /_bulk")
	if !strings.Contains(bulk, `"_id":"e1"`) || !strings.Contains(bulk, `"workflowStatus":"draft"`) {
		t.Errorf("bulk: %s", bulk)
	}
}

func TestSearchQueryAndResult(t *testing.T) {
	x, f := setup(t)
	page, err := x.Search(context.Background(), ct, content.Query{Q: "hilu", Page: 2, Size: 10, Sort: "plate:asc", States: []string{"submitted"}})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || page.Items[0].ID != "e1" || page.Items[0].Data["plate"] != "L 1" || page.Items[0].Status != "draft" || page.Items[0].CreatedBy != "u1" {
		t.Errorf("hasil: %+v", page.Items[0])
	}
	var body map[string]any
	json.Unmarshal([]byte(strings.SplitN(f.find("POST /eam-ct-vehicle/_search"), " ", 3)[2]), &body)
	s, _ := json.Marshal(body)
	for _, want := range []string{`"from":10`, `"size":10`, `"plate.keyword"`, `"workflowStatus":["submitted"]`, `"bool_prefix"`, `"plate","notes"`} {
		if !strings.Contains(string(s), want) {
			t.Errorf("query tanpa %s: %s", want, s)
		}
	}
}

func TestSortOnlyOnKnownSortableFields(t *testing.T) {
	x, f := setup(t)
	x.Search(context.Background(), ct, content.Query{Page: 1, Size: 10, Sort: "addr:asc"})
	if !strings.Contains(f.find("POST /eam-ct-vehicle/_search"), `"createdAt"`) {
		t.Error("sort ke field component harus diabaikan (default createdAt)")
	}
}

func TestErrorStatusReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(503)
		w.Write([]byte(`{"error":"no_shard_available"}`))
	}))
	defer srv.Close()
	if _, err := New(srv.URL, "").Search(context.Background(), ct, content.Query{Page: 1, Size: 1}); err == nil {
		t.Error("503 harus jadi error")
	}
}

func TestMissingIndexIsEmptyPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(404) }))
	defer srv.Close()
	page, err := New(srv.URL, "").Search(context.Background(), ct, content.Query{Page: 1, Size: 1})
	if err != nil || page.Total != 0 {
		t.Errorf("index belum ada = kosong: %+v %v", page, err)
	}
}

func TestRebuildSetsReplicaCount(t *testing.T) {
	f := &fakeES{}
	srv := httptest.NewServer(http.HandlerFunc(f.handler))
	defer srv.Close()
	x := New(srv.URL, "")
	x.Replicas = 0 // server single-node
	if err := x.Rebuild(context.Background(), ct, nil); err != nil {
		t.Fatal(err)
	}
	if put := f.find("PUT /eam-ct-vehicle "); !strings.Contains(put, `"number_of_replicas":0`) {
		t.Errorf("settings replica tidak dikirim: %s", put)
	}
}

func TestDefaultReplicaIsOne(t *testing.T) {
	if New("http://x", "").Replicas != 1 {
		t.Error("default 1 replica (aman untuk cluster multi-node)")
	}
}

func TestDomainFieldMappedAsKeyword(t *testing.T) {
	x, f := setup(t)
	dct := &content.ContentType{UID: "task", Fields: []content.Field{{Name: "priority", Type: content.TDomain, Domain: "PRIORITY"}}}
	x.Rebuild(context.Background(), dct, nil)
	if put := f.find("PUT /eam-ct-task "); !strings.Contains(put, `"priority":{"type":"keyword"}`) {
		t.Errorf("mapping domain: %s", put)
	}
}

func TestSearchScopeFilter(t *testing.T) {
	x, f := setup(t)
	x.Search(context.Background(), ct, content.Query{Page: 1, Size: 10, Scope: &content.Scope{Field: "site", Values: []string{"S1", "S2"}}})
	if body := f.find("POST /eam-ct-vehicle/_search"); !strings.Contains(body, `{"terms":{"site":["S1","S2"]}}`) {
		t.Errorf("filter site tidak dikirim: %s", body)
	}
}

func TestSearchWhereFilters(t *testing.T) {
	x, f := setup(t)
	x.Search(context.Background(), ct, content.Query{Page: 1, Size: 10, Filters: []content.Scope{{Field: "plate", Values: []string{"B1"}}, {Field: "workflowStatus", Values: []string{"OK"}}}})
	body := f.find("POST /eam-ct-vehicle/_search")
	if !strings.Contains(body, `{"terms":{"plate.keyword":["B1"]}}`) || !strings.Contains(body, `{"terms":{"workflowStatus":["OK"]}}`) {
		t.Errorf("filter where: %s", body)
	}
}
