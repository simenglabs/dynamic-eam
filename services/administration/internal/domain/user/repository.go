package user

import "context"

// Repository: User storage port (Mongo implementation in infra/mongo).
// Get/FindByLogin -> errs.NotFound; Insert/Update -> errs.Conflict when username/email collides.
type Repository interface {
	Get(ctx context.Context, id string) (*User, error)
	FindByLogin(ctx context.Context, usernameOrEmail string) (*User, error)
	List(ctx context.Context) ([]*User, error)
	Insert(ctx context.Context, u *User) error
	Update(ctx context.Context, u *User) error
	Delete(ctx context.Context, id string) error
	// RecordFailedLogin atomically increments the counter and locks the account when ShouldLock.
	RecordFailedLogin(ctx context.Context, id string) (attempts int, locked bool, err error)
	ResetFailedLogins(ctx context.Context, id string) error
	CountWithRole(ctx context.Context, roleID string) (int, error)
}
