// Package mongo: Mongo adapter for schemas, entries (collection ct_<uid>) and media metadata.
// The document format is compatible with pre-refactor data.
package mongo

import (
	"context"
	"errors"
	"strings"
	"time"

	"eam/services/common/errs"
	"eam/services/dynamic/internal/domain/content"
	"eam/services/dynamic/internal/domain/media"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Store struct{ db *mongo.Database }

func New(db *mongo.Database) *Store { return &Store{db} }

func notFound(err error, what string) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return errs.NotFound(what)
	}
	return err
}

// ---------- schema docs ----------

type fieldDoc struct {
	Name       string         `bson:"name"`
	Label      string         `bson:"label"`
	Type       string         `bson:"type"`
	Required   bool           `bson:"required"`
	Unique     bool           `bson:"unique,omitempty"`
	Default    any            `bson:"default,omitempty"`
	Options    []string       `bson:"options,omitempty"`
	Target     string         `bson:"target,omitempty"`
	Multiple   bool           `bson:"multiple,omitempty"`
	Component  string         `bson:"component,omitempty"`
	Repeatable bool           `bson:"repeatable,omitempty"`
	UniqueBy   string         `bson:"uniqueBy,omitempty"`
	Domain     string         `bson:"domain,omitempty"`
	AutoFill   *autoFillDoc   `bson:"autoFill,omitempty"`
	Where      []conditionDoc `bson:"where,omitempty"`
	From       string         `bson:"from,omitempty"`
}

type conditionDoc struct {
	Field string `bson:"field"`
	Value string `bson:"value"`
}

type autoFillDoc struct {
	Source  string `bson:"source"`
	From    string `bson:"from,omitempty"`
	Field   string `bson:"field,omitempty"`
	Lock    bool   `bson:"lock,omitempty"`
	Pattern string `bson:"pattern,omitempty"`
}

type rowDoc struct {
	Columns int      `bson:"columns"`
	Fields  []string `bson:"fields"`
}

type sectionDoc struct {
	ID    string   `bson:"id"`
	Title string   `bson:"title"`
	Rows  []rowDoc `bson:"rows"`
}

type stateDoc struct {
	Key      string `bson:"key"`
	Label    string `bson:"label"`
	Color    string `bson:"color"`
	Editable bool   `bson:"editable"`
}

type transitionDoc struct {
	Key    string     `bson:"key"`
	Label  string     `bson:"label"`
	From   []string   `bson:"from"`
	To     string     `bson:"to"`
	Fields []fieldDoc `bson:"fields,omitempty"`
}

type viewDoc struct {
	Key    string   `bson:"key"`
	Label  string   `bson:"label"`
	Parent string   `bson:"parent"`
	States []string `bson:"states"`
}

type workflowDoc struct {
	States      []stateDoc      `bson:"states"`
	Initial     string          `bson:"initial"`
	Transitions []transitionDoc `bson:"transitions"`
	Views       []viewDoc       `bson:"views"`
}

type ctDoc struct {
	UID          string       `bson:"_id"`
	Label        string       `bson:"label"`
	Parent       string       `bson:"parent"`
	Hidden       bool         `bson:"hidden,omitempty"`
	ScopeField   string       `bson:"scopeField,omitempty"`
	DisplayField string       `bson:"displayField"`
	Fields       []fieldDoc   `bson:"fields"`
	Layout       []sectionDoc `bson:"layout,omitempty"`
	Workflow     *workflowDoc `bson:"workflow,omitempty"`
	CreatedAt    time.Time    `bson:"createdAt"`
	UpdatedAt    time.Time    `bson:"updatedAt"`
}

type compDoc struct {
	UID       string     `bson:"_id"`
	Label     string     `bson:"label"`
	Fields    []fieldDoc `bson:"fields"`
	CreatedAt time.Time  `bson:"createdAt"`
	UpdatedAt time.Time  `bson:"updatedAt"`
}

func mapAll[T, D any](xs []T, f func(T) D) []D {
	out := make([]D, len(xs))
	for i, x := range xs {
		out[i] = f(x)
	}
	return out
}

func toFieldDoc(f content.Field) fieldDoc {
	d := fieldDoc{f.Name, f.Label, f.Type, f.Required, f.Unique, f.Default, f.Options, f.Target, f.Multiple, f.Component, f.Repeatable, f.UniqueBy, f.Domain, nil,
		mapAll(f.Where, func(c content.Condition) conditionDoc { return conditionDoc{c.Field, c.Value} }), f.From}
	if af := f.AutoFill; af != nil {
		d.AutoFill = &autoFillDoc{af.Source, af.From, af.Field, af.Lock, af.Pattern}
	}
	return d
}

func (d fieldDoc) toField() content.Field {
	return content.Field{Name: d.Name, Label: d.Label, Type: d.Type, Required: d.Required, Unique: d.Unique, Default: normalize(d.Default),
		Options: d.Options, Target: d.Target, Multiple: d.Multiple, Component: d.Component, Repeatable: d.Repeatable, UniqueBy: d.UniqueBy, Domain: d.Domain,
		From: d.From, Where: mapAll(d.Where, func(c conditionDoc) content.Condition { return content.Condition{Field: c.Field, Value: c.Value} }),
		AutoFill: autoFillOf(d.AutoFill)}
}

func autoFillOf(d *autoFillDoc) *content.AutoFill {
	if d == nil {
		return nil
	}
	return &content.AutoFill{Source: d.Source, From: d.From, Field: d.Field, Lock: d.Lock, Pattern: d.Pattern}
}

func toCTDoc(ct *content.ContentType) ctDoc {
	d := ctDoc{UID: ct.UID, Label: ct.Label, Parent: ct.Parent, Hidden: ct.Hidden, ScopeField: ct.ScopeField, DisplayField: ct.DisplayField,
		Fields: mapAll(ct.Fields, toFieldDoc), CreatedAt: ct.CreatedAt, UpdatedAt: ct.UpdatedAt}
	if ct.Layout != nil {
		d.Layout = mapAll(ct.Layout.Sections, func(s content.Section) sectionDoc {
			return sectionDoc{s.ID, s.Title, mapAll(s.Rows, func(r content.Row) rowDoc { return rowDoc{r.Columns, r.Fields} })}
		})
	}
	if w := ct.Workflow; w != nil {
		d.Workflow = &workflowDoc{
			States:  mapAll(w.States, func(s content.State) stateDoc { return stateDoc{s.Key, s.Label, s.Color, s.Editable} }),
			Initial: w.Initial,
			Transitions: mapAll(w.Transitions, func(t content.Transition) transitionDoc {
				return transitionDoc{t.Key, t.Label, t.From, t.To, mapAll(t.Fields, toFieldDoc)}
			}),
			Views: mapAll(w.Views, func(v content.View) viewDoc { return viewDoc{v.Key, v.Label, v.Parent, v.States} }),
		}
	}
	return d
}

func (d ctDoc) toContentType() *content.ContentType {
	ct := &content.ContentType{UID: d.UID, Label: d.Label, Parent: d.Parent, Hidden: d.Hidden, ScopeField: d.ScopeField, DisplayField: d.DisplayField,
		Fields: mapAll(d.Fields, fieldDoc.toField), CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt}
	if d.Layout != nil {
		ct.Layout = &content.Layout{Sections: mapAll(d.Layout, func(s sectionDoc) content.Section {
			return content.Section{ID: s.ID, Title: s.Title, Rows: mapAll(s.Rows, func(r rowDoc) content.Row { return content.Row{Columns: r.Columns, Fields: r.Fields} })}
		})}
	}
	if w := d.Workflow; w != nil {
		ct.Workflow = &content.Workflow{
			States: mapAll(w.States, func(s stateDoc) content.State {
				return content.State{Key: s.Key, Label: s.Label, Color: s.Color, Editable: s.Editable}
			}),
			Initial: w.Initial,
			Transitions: mapAll(w.Transitions, func(t transitionDoc) content.Transition {
				return content.Transition{Key: t.Key, Label: t.Label, From: t.From, To: t.To, Fields: mapAll(t.Fields, func(f fieldDoc) content.Field { return f.toField() })}
			}),
			Views: mapAll(w.Views, func(v viewDoc) content.View {
				return content.View{Key: v.Key, Label: v.Label, Parent: v.Parent, States: v.States}
			}),
		}
	}
	return ct
}

func findAll[D any, T any](ctx context.Context, c *mongo.Collection, conv func(D) T) ([]T, error) {
	cur, err := c.Find(ctx, bson.M{}, options.Find().SetSort(bson.M{"label": 1}))
	if err != nil {
		return nil, err
	}
	var docs []D
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	return mapAll(docs, conv), nil
}

func (s *Store) ListContentTypes(ctx context.Context) ([]*content.ContentType, error) {
	return findAll(ctx, s.db.Collection("content_types"), ctDoc.toContentType)
}

func (s *Store) GetContentType(ctx context.Context, uid string) (*content.ContentType, error) {
	var d ctDoc
	if err := s.db.Collection("content_types").FindOne(ctx, bson.M{"_id": uid}).Decode(&d); err != nil {
		return nil, notFound(err, "content type")
	}
	return d.toContentType(), nil
}

func (s *Store) SaveContentType(ctx context.Context, ct *content.ContentType) error {
	_, err := s.db.Collection("content_types").ReplaceOne(ctx, bson.M{"_id": ct.UID}, toCTDoc(ct), options.Replace().SetUpsert(true))
	return err
}

func (s *Store) DeleteContentType(ctx context.Context, uid string) error {
	_, err := s.db.Collection("content_types").DeleteOne(ctx, bson.M{"_id": uid})
	return err
}

func (d compDoc) toComponent() *content.Component {
	return &content.Component{UID: d.UID, Label: d.Label, Fields: mapAll(d.Fields, fieldDoc.toField), CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt}
}

func (s *Store) ListComponents(ctx context.Context) ([]*content.Component, error) {
	return findAll(ctx, s.db.Collection("components"), compDoc.toComponent)
}

func (s *Store) GetComponent(ctx context.Context, uid string) (*content.Component, error) {
	var d compDoc
	if err := s.db.Collection("components").FindOne(ctx, bson.M{"_id": uid}).Decode(&d); err != nil {
		return nil, notFound(err, "component")
	}
	return d.toComponent(), nil
}

func (s *Store) SaveComponent(ctx context.Context, c *content.Component) error {
	d := compDoc{c.UID, c.Label, mapAll(c.Fields, toFieldDoc), c.CreatedAt, c.UpdatedAt}
	_, err := s.db.Collection("components").ReplaceOne(ctx, bson.M{"_id": c.UID}, d, options.Replace().SetUpsert(true))
	return err
}

func (s *Store) DeleteComponent(ctx context.Context, uid string) error {
	_, err := s.db.Collection("components").DeleteOne(ctx, bson.M{"_id": uid})
	return err
}

// ---------- entries: document = data fields at the top level + metadata ----------

const (
	kID, kCreatedAt, kUpdatedAt, kCreatedBy, kUpdatedBy = "_id", "createdAt", "updatedAt", "createdBy", "updatedBy"
	kStatus, kHistory                                   = "workflowStatus", "workflowHistory"
	kVersion, kDeleted, kDeletedAt, kDeletedBy          = "version", "isDeleted", "deletedAt", "deletedBy"
)

// notDeleted: filter for active entries (legacy documents without an isDeleted field are treated as active).
var notDeleted = bson.M{kDeleted: bson.M{"$ne": true}}

func isMeta(k string) bool {
	switch k {
	case kID, kCreatedAt, kUpdatedAt, kCreatedBy, kUpdatedBy, kStatus, kHistory, kVersion, kDeleted, kDeletedAt, kDeletedBy:
		return true
	}
	return false
}

type historyDoc struct {
	From       string    `bson:"from"`
	To         string    `bson:"to"`
	Transition string    `bson:"transition"`
	By         string    `bson:"by"`
	At         time.Time `bson:"at"`
	Note       string    `bson:"note,omitempty"`
}

func (s *Store) entries(uid string) *mongo.Collection { return s.db.Collection("ct_" + uid) }

func toEntryDoc(e *content.Entry) bson.M {
	doc := bson.M{}
	for k, v := range e.Data {
		doc[k] = v
	}
	doc[kID], doc[kCreatedAt], doc[kUpdatedAt], doc[kCreatedBy], doc[kUpdatedBy] = e.ID, e.CreatedAt, e.UpdatedAt, e.CreatedBy, e.UpdatedBy
	doc[kVersion], doc[kDeleted] = e.Version, e.Deleted
	if e.Deleted {
		doc[kDeletedAt], doc[kDeletedBy] = e.DeletedAt, e.DeletedBy
	}
	if e.Status != "" {
		doc[kStatus] = e.Status
	}
	if len(e.History) > 0 {
		doc[kHistory] = mapAll(e.History, func(h content.StatusChange) historyDoc {
			return historyDoc{h.From, h.To, h.Transition, h.By, h.At, h.Note}
		})
	}
	return doc
}

func fromEntryDoc(doc bson.M) *content.Entry {
	str := func(k string) string { s, _ := doc[k].(string); return s }
	tm := func(k string) time.Time {
		if d, ok := doc[k].(bson.DateTime); ok {
			return d.Time().UTC()
		}
		return time.Time{}
	}
	e := &content.Entry{ID: str(kID), CreatedAt: tm(kCreatedAt), UpdatedAt: tm(kUpdatedAt), CreatedBy: str(kCreatedBy), UpdatedBy: str(kUpdatedBy), Status: str(kStatus), Data: map[string]any{},
		DeletedAt: tm(kDeletedAt), DeletedBy: str(kDeletedBy)}
	e.Deleted, _ = doc[kDeleted].(bool)
	if v, ok := normalize(doc[kVersion]).(float64); ok {
		e.Version = int(v)
	}
	if raw, ok := doc[kHistory].(bson.A); ok {
		for _, h := range raw {
			m, _ := h.(bson.M)
			hs := func(k string) string { s, _ := m[k].(string); return s }
			at, _ := m["at"].(bson.DateTime)
			e.History = append(e.History, content.StatusChange{From: hs("from"), To: hs("to"), Transition: hs("transition"), By: hs("by"), At: at.Time().UTC(), Note: hs("note")})
		}
	}
	for k, v := range doc {
		if !isMeta(k) {
			e.Data[k] = normalize(v)
		}
	}
	return e
}

func normalizeMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out, _ := normalize(bson.M(m)).(map[string]any)
	return out
}

// normalize: BSON types -> plain JSON types (map[string]any, []any) so domain & http know nothing about BSON.
func normalize(v any) any {
	switch x := v.(type) {
	case bson.M:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[k] = normalize(val)
		}
		return out
	case bson.A:
		out := make([]any, len(x))
		for i, val := range x {
			out[i] = normalize(val)
		}
		return out
	case bson.DateTime:
		return x.Time().UTC()
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	}
	return v
}

func (s *Store) Get(ctx context.Context, uid, id string) (*content.Entry, error) {
	var doc bson.M
	if err := s.entries(uid).FindOne(ctx, bson.M{kID: id}).Decode(&doc); err != nil {
		return nil, notFound(err, "record")
	}
	return fromEntryDoc(doc), nil
}

func (s *Store) Save(ctx context.Context, uid string, e *content.Entry) error {
	_, err := s.entries(uid).ReplaceOne(ctx, bson.M{kID: e.ID}, toEntryDoc(e), options.Replace().SetUpsert(true))
	if mongo.IsDuplicateKeyError(err) {
		return errs.Conflict("a value that must be unique is already used by another record")
	}
	return err
}

func (s *Store) All(ctx context.Context, uid string) ([]*content.Entry, error) {
	cur, err := s.entries(uid).Find(ctx, notDeleted, options.Find().SetSort(bson.M{kCreatedAt: 1}))
	if err != nil {
		return nil, err
	}
	var docs []bson.M
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	return mapAll(docs, fromEntryDoc), nil
}

func (s *Store) FindOne(ctx context.Context, uid, field string, value any) (*content.Entry, error) {
	var doc bson.M
	if err := s.entries(uid).FindOne(ctx, bson.M{field: value, kDeleted: bson.M{"$ne": true}}).Decode(&doc); err != nil {
		return nil, notFound(err, "record")
	}
	return fromEntryDoc(doc), nil
}

func (s *Store) CountExisting(ctx context.Context, uid string, ids []string) (int, error) {
	n, err := s.entries(uid).CountDocuments(ctx, bson.M{kID: bson.M{"$in": ids}, kDeleted: bson.M{"$ne": true}})
	return int(n), err
}

// SyncUnique: partial index "uniq2_<field>" — only entries that have that field and are not deleted.
// The old "uniq_<field>" index (without the isDeleted filter) is replaced.
func (s *Store) SyncUnique(ctx context.Context, uid string, fields []string) error {
	col := s.entries(uid)
	// Legacy documents without isDeleted are set to false so they are included in the partial index.
	if _, err := col.UpdateMany(ctx, bson.M{kDeleted: bson.M{"$exists": false}}, bson.M{"$set": bson.M{kDeleted: false}}); err != nil {
		return err
	}
	specs, err := col.Indexes().ListSpecifications(ctx)
	if err != nil {
		return err
	}
	want := map[string]bool{}
	for _, f := range fields {
		want["uniq2_"+f] = true
	}
	have := map[string]bool{}
	for _, sp := range specs {
		if !strings.HasPrefix(sp.Name, "uniq_") && !strings.HasPrefix(sp.Name, "uniq2_") {
			continue
		}
		if !want[sp.Name] {
			if err := col.Indexes().DropOne(ctx, sp.Name); err != nil {
				return err
			}
			continue
		}
		have[sp.Name] = true
	}
	for _, f := range fields {
		if have["uniq2_"+f] {
			continue
		}
		_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys:    bson.D{{Key: f, Value: 1}},
			Options: options.Index().SetName("uniq2_" + f).SetUnique(true).SetPartialFilterExpression(bson.M{f: bson.M{"$exists": true}, kDeleted: false}),
		})
		if mongo.IsDuplicateKeyError(err) {
			return errs.Conflict("field %s cannot be made unique: duplicate values already exist", f)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Drop(ctx context.Context, uid string) error { return s.entries(uid).Drop(ctx) }

// ---------- audit trail: collection entry_history (kept even if the content type is deleted) ----------

type changeDoc struct {
	Field string `bson:"field"`
	From  any    `bson:"from"`
	To    any    `bson:"to"`
}

type revisionDoc struct {
	UID        string         `bson:"uid"`
	EntryID    string         `bson:"entryId"`
	Version    int            `bson:"version"`
	Action     string         `bson:"action"`
	By         string         `bson:"by"`
	ByName     string         `bson:"byName"`
	At         time.Time      `bson:"at"`
	Changes    []changeDoc    `bson:"changes"`
	Note       string         `bson:"note,omitempty"`
	Transition string         `bson:"transition,omitempty"`
	Data       map[string]any `bson:"data,omitempty"`
}

func (s *Store) history() *mongo.Collection { return s.db.Collection("entry_history") }

// EnsureIndexes: history indexes (called once at startup).
func (s *Store) EnsureIndexes(ctx context.Context) error {
	_, err := s.history().Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "uid", Value: 1}, {Key: "entryId", Value: 1}, {Key: "version", Value: 1}}})
	return err
}

// NextSequence: counters collection {_id: key, seq}; atomic $inc + upsert in Mongo.
func (s *Store) NextSequence(ctx context.Context, key string) (int64, error) {
	var d struct {
		Seq int64 `bson:"seq"`
	}
	err := s.db.Collection("counters").FindOneAndUpdate(ctx, bson.M{"_id": key}, bson.M{"$inc": bson.M{"seq": int64(1)}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)).Decode(&d)
	return d.Seq, err
}

func (s *Store) PeekSequence(ctx context.Context, key string) (int64, error) {
	var d struct {
		Seq int64 `bson:"seq"`
	}
	err := s.db.Collection("counters").FindOne(ctx, bson.M{"_id": key}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		err = nil
	}
	return d.Seq + 1, err
}

func (s *Store) AddRevision(ctx context.Context, uid string, r *content.Revision) error {
	d := revisionDoc{uid, r.EntryID, r.Version, r.Action, r.By, r.ByName, r.At,
		mapAll(r.Changes, func(c content.Change) changeDoc { return changeDoc{c.Field, c.From, c.To} }), r.Note, r.Transition, r.Data}
	_, err := s.history().InsertOne(ctx, d)
	return err
}

func (s *Store) Revisions(ctx context.Context, uid, entryID string) ([]*content.Revision, error) {
	cur, err := s.history().Find(ctx, bson.M{"uid": uid, "entryId": entryID}, options.Find().SetSort(bson.D{{Key: "version", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var docs []revisionDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	return mapAll(docs, func(d revisionDoc) *content.Revision {
		return &content.Revision{EntryID: d.EntryID, Version: d.Version, Action: d.Action, By: d.By, ByName: d.ByName, At: d.At.UTC(), Note: d.Note,
			Transition: d.Transition, Data: normalizeMap(d.Data),
			Changes: mapAll(d.Changes, func(c changeDoc) content.Change {
				return content.Change{Field: c.Field, From: normalize(c.From), To: normalize(c.To)}
			})}
	}), nil
}

// ---------- media ----------

type mediaDoc struct {
	ID        string    `bson:"_id"`
	Name      string    `bson:"name"`
	Mime      string    `bson:"mime"`
	Size      int64     `bson:"size"`
	CreatedBy string    `bson:"createdBy"`
	CreatedAt time.Time `bson:"createdAt"`
}

func (d mediaDoc) toMedia() *media.Media {
	return &media.Media{ID: d.ID, Name: d.Name, Mime: d.Mime, Size: d.Size, CreatedBy: d.CreatedBy, CreatedAt: d.CreatedAt}
}

type Media struct{ c *mongo.Collection }

func (s *Store) Media() *Media { return &Media{s.db.Collection("media")} }

func (r *Media) Save(ctx context.Context, m *media.Media) error {
	_, err := r.c.InsertOne(ctx, mediaDoc{m.ID, m.Name, m.Mime, m.Size, m.CreatedBy, m.CreatedAt})
	return err
}

func (r *Media) Get(ctx context.Context, id string) (*media.Media, error) {
	var d mediaDoc
	if err := r.c.FindOne(ctx, bson.M{"_id": id}).Decode(&d); err != nil {
		return nil, notFound(err, "file")
	}
	return d.toMedia(), nil
}

func (r *Media) List(ctx context.Context, ids []string) ([]*media.Media, error) {
	cur, err := r.c.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, err
	}
	var docs []mediaDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	return mapAll(docs, mediaDoc.toMedia), nil
}

func (r *Media) CountExisting(ctx context.Context, ids []string) (int, error) {
	n, err := r.c.CountDocuments(ctx, bson.M{"_id": bson.M{"$in": ids}})
	return int(n), err
}
