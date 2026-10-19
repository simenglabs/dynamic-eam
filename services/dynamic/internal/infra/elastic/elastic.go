// Package elastic: content.SearchIndex via REST Elasticsearch (tanpa SDK). Satu index per content type.
package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"eam/services/dynamic/internal/domain/content"
)

type Index struct {
	base, apiKey string
	hc           *http.Client
	// Replicas: jumlah replica per index. Server single-node harus 0 (replica tidak bisa di node yang sama = status yellow).
	Replicas int
}

func New(base, apiKey string) *Index {
	return &Index{base: strings.TrimRight(base, "/"), apiKey: apiKey, hc: &http.Client{Timeout: 20 * time.Second}, Replicas: 1}
}

func name(uid string) string { return "eam-ct-" + uid }

// Field metadata di dokumen index (selain field schema).
const (
	fCreatedAt = "createdAt"
	fUpdatedAt = "updatedAt"
	fCreatedBy = "createdBy"
	fUpdatedBy = "updatedBy"
	fStatus    = "workflowStatus"
)

func (x *Index) do(ctx context.Context, method, path string, body io.Reader, out any) (int, error) {
	req, err := http.NewRequestWithContext(ctx, method, x.base+path, body)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if path == "/_bulk" || strings.HasPrefix(path, "/_bulk?") {
		req.Header.Set("Content-Type", "application/x-ndjson")
	}
	if x.apiKey != "" {
		req.Header.Set("Authorization", "ApiKey "+x.apiKey)
	}
	res, err := x.hc.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode == http.StatusNotFound {
		return res.StatusCode, nil
	}
	if res.StatusCode >= 300 {
		return res.StatusCode, fmt.Errorf("elasticsearch %s %s: %d %.300s", method, path, res.StatusCode, raw)
	}
	if out != nil {
		return res.StatusCode, json.Unmarshal(raw, out)
	}
	return res.StatusCode, nil
}

func jsonBody(v any) io.Reader {
	b, _ := json.Marshal(v)
	return bytes.NewReader(b)
}

func mapping(ct *content.ContentType, replicas int) map[string]any {
	kw := map[string]any{"type": "keyword"}
	date := map[string]any{"type": "date"}
	props := map[string]any{fCreatedAt: date, fUpdatedAt: date, fCreatedBy: kw, fUpdatedBy: kw, fStatus: kw}
	for _, f := range ct.Fields {
		switch f.Type {
		case content.TText:
			props[f.Name] = map[string]any{"type": "text", "fields": map[string]any{"keyword": map[string]any{"type": "keyword", "ignore_above": 256}}}
		case content.TLongText:
			props[f.Name] = map[string]any{"type": "text"}
		case content.TNumber:
			props[f.Name] = map[string]any{"type": "double"}
		case content.TBoolean:
			props[f.Name] = map[string]any{"type": "boolean"}
		case content.TDate:
			props[f.Name] = map[string]any{"type": "date", "format": "yyyy-MM-dd"}
		case content.TSelect, content.TRelation, content.TMedia, content.TDomain:
			props[f.Name] = kw
		case content.TComponent:
			props[f.Name] = map[string]any{"type": "object", "enabled": false}
		}
	}
	// dynamic=false: field di luar schema tetap di _source tapi tidak di-index (cegah konflik mapping).
	return map[string]any{
		"settings": map[string]any{"number_of_replicas": replicas},
		"mappings": map[string]any{"dynamic": false, "properties": props},
	}
}

func source(e *content.Entry) map[string]any {
	doc := make(map[string]any, len(e.Data)+5)
	for k, v := range e.Data {
		doc[k] = v
	}
	doc[fCreatedAt], doc[fUpdatedAt], doc[fCreatedBy], doc[fUpdatedBy] = e.CreatedAt, e.UpdatedAt, e.CreatedBy, e.UpdatedBy
	if e.Status != "" {
		doc[fStatus] = e.Status
	}
	return doc
}

// Rebuild: hapus index, buat ulang dengan mapping terbaru, isi ulang semua entri.
func (x *Index) Rebuild(ctx context.Context, ct *content.ContentType, entries []*content.Entry) error {
	idx := name(ct.UID)
	if _, err := x.do(ctx, http.MethodDelete, "/"+idx, nil, nil); err != nil {
		return err
	}
	if _, err := x.do(ctx, http.MethodPut, "/"+idx, jsonBody(mapping(ct, x.Replicas)), nil); err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, e := range entries {
		enc.Encode(map[string]any{"index": map[string]any{"_index": idx, "_id": e.ID}})
		enc.Encode(source(e))
	}
	var res struct {
		Errors bool `json:"errors"`
	}
	if _, err := x.do(ctx, http.MethodPost, "/_bulk?refresh=wait_for", &buf, &res); err != nil {
		return err
	}
	if res.Errors {
		return fmt.Errorf("elasticsearch bulk: sebagian dokumen gagal di-index")
	}
	return nil
}

func (x *Index) Put(ctx context.Context, ct *content.ContentType, e *content.Entry) error {
	_, err := x.do(ctx, http.MethodPut, fmt.Sprintf("/%s/_doc/%s?refresh=wait_for", name(ct.UID), e.ID), jsonBody(source(e)), nil)
	return err
}

func (x *Index) Delete(ctx context.Context, uid, id string) error {
	_, err := x.do(ctx, http.MethodDelete, fmt.Sprintf("/%s/_doc/%s?refresh=wait_for", name(uid), id), nil, nil)
	return err
}

func (x *Index) Drop(ctx context.Context, uid string) error {
	_, err := x.do(ctx, http.MethodDelete, "/"+name(uid), nil, nil)
	return err
}

func (x *Index) Search(ctx context.Context, ct *content.ContentType, q content.Query) (content.Page, error) {
	var text []string
	sortField, dir := fCreatedAt, "desc"
	field, d, _ := strings.Cut(q.Sort, ":")
	for _, f := range ct.Fields {
		if f.Type == content.TText || f.Type == content.TLongText {
			text = append(text, f.Name)
		}
		if f.Name == field && (d == "asc" || d == "desc") && f.Type != content.TComponent && f.Type != content.TLongText {
			sortField, dir = f.Name, d
			if f.Type == content.TText {
				sortField += ".keyword"
			}
		}
	}
	must := any(map[string]any{"match_all": map[string]any{}})
	if q.Q != "" && len(text) > 0 {
		must = map[string]any{"multi_match": map[string]any{"query": q.Q, "type": "bool_prefix", "fields": text, "operator": "and"}}
	}
	boolQ := map[string]any{"must": must}
	var filters []any
	if len(q.States) > 0 {
		filters = append(filters, map[string]any{"terms": map[string]any{fStatus: q.States}})
	}
	if q.Scope != nil { // pembatasan per site (field relasi = keyword)
		filters = append(filters, map[string]any{"terms": map[string]any{q.Scope.Field: q.Scope.Values}})
	}
	for _, f := range q.Filters { // filter relasi: field teks dicocokkan via subfield keyword
		field := f.Field
		if i := slices.IndexFunc(ct.Fields, func(x content.Field) bool { return x.Name == f.Field }); i >= 0 && ct.Fields[i].Type == content.TText {
			field += ".keyword"
		}
		filters = append(filters, map[string]any{"terms": map[string]any{field: f.Values}})
	}
	if len(filters) > 0 {
		boolQ["filter"] = filters
	}
	body := map[string]any{
		"query": map[string]any{"bool": boolQ}, "from": max(q.Page-1, 0) * q.Size, "size": q.Size, "track_total_hits": true,
		"sort": []any{map[string]any{sortField: map[string]any{"order": dir, "unmapped_type": "keyword"}}},
	}
	var res struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				ID     string         `json:"_id"`
				Source map[string]any `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	status, err := x.do(ctx, http.MethodPost, "/"+name(ct.UID)+"/_search", jsonBody(body), &res)
	if err != nil || status == http.StatusNotFound {
		return content.Page{Items: []*content.Entry{}}, err
	}
	page := content.Page{Total: res.Hits.Total.Value, Items: make([]*content.Entry, 0, len(res.Hits.Hits))}
	for _, h := range res.Hits.Hits {
		page.Items = append(page.Items, fromSource(h.ID, h.Source))
	}
	return page, nil
}

func fromSource(id string, src map[string]any) *content.Entry {
	e := &content.Entry{ID: id, Data: map[string]any{}}
	str := func(k string) string { s, _ := src[k].(string); return s }
	tm := func(k string) time.Time { t, _ := time.Parse(time.RFC3339Nano, str(k)); return t }
	e.CreatedAt, e.UpdatedAt, e.CreatedBy, e.UpdatedBy, e.Status = tm(fCreatedAt), tm(fUpdatedAt), str(fCreatedBy), str(fUpdatedBy), str(fStatus)
	for k, v := range src {
		switch k {
		case fCreatedAt, fUpdatedAt, fCreatedBy, fUpdatedBy, fStatus:
		default:
			e.Data[k] = v
		}
	}
	return e
}
