package role

import "context"

// Repository: Role storage port. Get -> errs.NotFound; Insert -> errs.Conflict when ID/name collides.
type Repository interface {
	List(ctx context.Context) ([]*Role, error)
	Get(ctx context.Context, id string) (*Role, error)
	Insert(ctx context.Context, r *Role) error
	Update(ctx context.Context, r *Role) error
	Delete(ctx context.Context, id string) error
	// CountExisting: how many IDs actually exist (validates a user's roles).
	CountExisting(ctx context.Context, ids []string) (int, error)
	// InsertIfMissing: seeds built-in roles without overwriting changes.
	InsertIfMissing(ctx context.Context, r *Role) error
}
