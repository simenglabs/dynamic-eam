package content

import (
	"context"
	"time"
)

// Entry: one record of a content type. Data = field values already validated by ValidateEntry.
type Entry struct {
	ID        string
	Data      map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
	CreatedBy string
	UpdatedBy string
	Status    string // "" if the content type has no workflow
	History   []StatusChange
	Version   int // incremented on each create/update/delete/transition
	// Soft delete: the entry stays in the database but is hidden from list/search/relations/export.
	Deleted   bool
	DeletedAt time.Time
	DeletedBy string
}

// SchemaRepository: storage for content type & component definitions. Get* -> errs.NotFound.
type SchemaRepository interface {
	ListContentTypes(ctx context.Context) ([]*ContentType, error)
	GetContentType(ctx context.Context, uid string) (*ContentType, error)
	SaveContentType(ctx context.Context, ct *ContentType) error
	DeleteContentType(ctx context.Context, uid string) error
	ListComponents(ctx context.Context) ([]*Component, error)
	GetComponent(ctx context.Context, uid string) (*Component, error)
	SaveComponent(ctx context.Context, c *Component) error
	DeleteComponent(ctx context.Context, uid string) error
}

// EntryRepository: entry storage per content type. Save -> errs.Conflict on a unique violation
// (unique applies only among non-deleted entries). Get also returns deleted entries (check Deleted);
// All & CountExisting ignore deleted entries.
type EntryRepository interface {
	Get(ctx context.Context, uid, id string) (*Entry, error)
	Save(ctx context.Context, uid string, e *Entry) error
	// ponytail: All loads every entry into memory (export & reindex); switch to an iterator/stream if records grow large.
	All(ctx context.Context, uid string) ([]*Entry, error)
	CountExisting(ctx context.Context, uid string, ids []string) (int, error)
	// FindOne: first active entry with field = value (errs.NotFound if none).
	FindOne(ctx context.Context, uid, field string, value any) (*Entry, error)
	// SyncUnique aligns unique constraints with the schema; errs.Conflict if existing records already have duplicates.
	SyncUnique(ctx context.Context, uid string, fields []string) error
	Drop(ctx context.Context, uid string) error
	// Audit trail: revisions stored separately, ordered by version.
	AddRevision(ctx context.Context, uid string, r *Revision) error
	Revisions(ctx context.Context, uid, entryID string) ([]*Revision, error)
	// NextSequence: next sequence number (starting at 1) for a counter key, atomic across requests.
	NextSequence(ctx context.Context, key string) (int64, error)
	// PeekSequence: the number the next NextSequence would return, without consuming it (form preview).
	PeekSequence(ctx context.Context, key string) (int64, error)
}

type Query struct {
	Q          string
	Page, Size int
	Sort       string   // "<field>:asc|desc"
	States     []string // status filter (workflow view); empty = all
	Scope      *Scope   // accessible-site filter (per-site scoping); nil = unrestricted
	Filters    []Scope  // relation filters (where): all must match
}

// Scope: only entries whose Field contains one of Values.
type Scope struct {
	Field  string
	Values []string
}

type Page struct {
	Total int
	Items []*Entry
}

// SearchIndex: list & full-text search (Elasticsearch in production).
type SearchIndex interface {
	Rebuild(ctx context.Context, ct *ContentType, entries []*Entry) error
	Put(ctx context.Context, ct *ContentType, e *Entry) error
	Delete(ctx context.Context, uid, id string) error
	Drop(ctx context.Context, uid string) error
	Search(ctx context.Context, ct *ContentType, q Query) (Page, error)
}
