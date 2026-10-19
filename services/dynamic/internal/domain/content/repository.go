package content

import (
	"context"
	"time"
)

// Entry: satu baris data content type. Data = nilai field yang sudah divalidasi ValidateEntry.
type Entry struct {
	ID        string
	Data      map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
	CreatedBy string
	UpdatedBy string
	Status    string // "" bila content type tanpa workflow
	History   []StatusChange
	Version   int // naik tiap create/update/delete/transisi
	// Soft delete: entri tetap di database, hilang dari list/pencarian/relasi/export.
	Deleted   bool
	DeletedAt time.Time
	DeletedBy string
}

// SchemaRepository: penyimpanan definisi content type & component. Get* -> errs.NotFound.
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

// EntryRepository: penyimpanan entri per content type. Save -> errs.Conflict bila melanggar unique
// (unique hanya berlaku antar entri yang belum dihapus). Get mengembalikan entri terhapus juga (cek Deleted);
// All & CountExisting mengabaikan entri terhapus.
type EntryRepository interface {
	Get(ctx context.Context, uid, id string) (*Entry, error)
	Save(ctx context.Context, uid string, e *Entry) error
	// ponytail: All memuat semua entri ke memori (export & reindex); ganti ke iterator/stream kalau data besar.
	All(ctx context.Context, uid string) ([]*Entry, error)
	CountExisting(ctx context.Context, uid string, ids []string) (int, error)
	// FindOne: entri aktif pertama dengan field = value (errs.NotFound bila tidak ada).
	FindOne(ctx context.Context, uid, field string, value any) (*Entry, error)
	// SyncUnique menyamakan constraint unique dengan schema; errs.Conflict bila data lama sudah duplikat.
	SyncUnique(ctx context.Context, uid string, fields []string) error
	Drop(ctx context.Context, uid string) error
	// Audit trail: revisi disimpan terpisah, urut versi.
	AddRevision(ctx context.Context, uid string, r *Revision) error
	Revisions(ctx context.Context, uid, entryID string) ([]*Revision, error)
	// NextSequence: nomor urut berikutnya (mulai 1) untuk kunci counter, atomik antar request.
	NextSequence(ctx context.Context, key string) (int64, error)
	// PeekSequence: nomor yang akan diberikan NextSequence berikutnya, tanpa memakainya (preview form).
	PeekSequence(ctx context.Context, key string) (int64, error)
}

type Query struct {
	Q          string
	Page, Size int
	Sort       string   // "<field>:asc|desc"
	States     []string // filter status (view workflow); kosong = semua
	Scope      *Scope   // filter site akses (pembatasan per site); nil = tidak dibatasi
	Filters    []Scope  // filter relasi (where): semua harus cocok
}

// Scope: hanya entri dengan Field berisi salah satu Values.
type Scope struct {
	Field  string
	Values []string
}

type Page struct {
	Total int
	Items []*Entry
}

// SearchIndex: list & pencarian full-text (Elasticsearch di produksi).
type SearchIndex interface {
	Rebuild(ctx context.Context, ct *ContentType, entries []*Entry) error
	Put(ctx context.Context, ct *ContentType, e *Entry) error
	Delete(ctx context.Context, uid, id string) error
	Drop(ctx context.Context, uid string) error
	Search(ctx context.Context, ct *ContentType, q Query) (Page, error)
}
