package role

import "context"

// Repository: port penyimpanan Role. Get -> errs.NotFound; Insert -> errs.Conflict bila ID/nama bentrok.
type Repository interface {
	List(ctx context.Context) ([]*Role, error)
	Get(ctx context.Context, id string) (*Role, error)
	Insert(ctx context.Context, r *Role) error
	Update(ctx context.Context, r *Role) error
	Delete(ctx context.Context, id string) error
	// CountExisting: berapa ID yang benar-benar ada (validasi role milik user).
	CountExisting(ctx context.Context, ids []string) (int, error)
	// InsertIfMissing: untuk seed role bawaan tanpa menimpa perubahan.
	InsertIfMissing(ctx context.Context, r *Role) error
}
