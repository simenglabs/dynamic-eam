// Package mongo: Mongo adapter for user.Repository & role.Repository.
// Documents keep the legacy shape (compatible with existing data).
package mongo

import (
	"context"
	"errors"
	"strings"
	"time"

	"eam/services/administration/internal/domain/role"
	"eam/services/administration/internal/domain/user"
	"eam/services/common/errs"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// New sets up indexes + schema migration, then returns both repositories.
func New(ctx context.Context, db *mongo.Database) (*Users, *Roles, error) {
	users, roles := db.Collection("users"), db.Collection("roles")
	_, err := users.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "usernameLower", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "emailLower", Value: 1}}, Options: options.Index().SetUnique(true)},
	})
	if err == nil {
		_, err = roles.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "name", Value: 1}}, Options: options.Index().SetUnique(true)})
	}
	if err == nil {
		// Migration: legacy field "role" (string) -> "roles" (array).
		_, err = users.UpdateMany(ctx, bson.M{"role": bson.M{"$exists": true}, "roles": bson.M{"$exists": false}},
			mongo.Pipeline{{{Key: "$set", Value: bson.M{"roles": bson.A{"$role"}}}}, {{Key: "$unset", Value: "role"}}})
	}
	return &Users{users}, &Roles{roles}, err
}

func notFound(err error, what string) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return errs.NotFound(what)
	}
	return err
}

// ---------- users ----------

type userDoc struct {
	ID             string    `bson:"_id"`
	Username       string    `bson:"username"`
	DisplayName    string    `bson:"displayName"`
	Email          string    `bson:"email"`
	Roles          []string  `bson:"roles"`
	IsActive       bool      `bson:"isActive"`
	IsLocked       bool      `bson:"isLocked"`
	FailedAttempts int       `bson:"failedAttempts"`
	CreatedAt      time.Time `bson:"createdAt"`
	UpdatedAt      time.Time `bson:"updatedAt"`
	UsernameLower  string    `bson:"usernameLower"`
	EmailLower     string    `bson:"emailLower"`
	Hash           []byte    `bson:"hash"`
}

func toUserDoc(u *user.User) userDoc {
	return userDoc{u.ID, u.Username, u.DisplayName, u.Email, u.Roles, u.Active, u.Locked, u.FailedAttempts,
		u.CreatedAt, u.UpdatedAt, u.UsernameKey(), u.EmailKey(), u.PasswordHash}
}

func (d userDoc) toUser() *user.User {
	return &user.User{ID: d.ID, Username: d.Username, DisplayName: d.DisplayName, Email: d.Email, Roles: d.Roles,
		Active: d.IsActive, Locked: d.IsLocked, FailedAttempts: d.FailedAttempts, PasswordHash: d.Hash,
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt}
}

type Users struct{ c *mongo.Collection }

func (r *Users) findOne(ctx context.Context, filter bson.M) (*user.User, error) {
	var d userDoc
	if err := r.c.FindOne(ctx, filter).Decode(&d); err != nil {
		return nil, notFound(err, "user")
	}
	return d.toUser(), nil
}

func (r *Users) Get(ctx context.Context, id string) (*user.User, error) {
	return r.findOne(ctx, bson.M{"_id": id})
}

func (r *Users) FindByLogin(ctx context.Context, login string) (*user.User, error) {
	l := strings.ToLower(strings.TrimSpace(login))
	return r.findOne(ctx, bson.M{"$or": bson.A{bson.M{"usernameLower": l}, bson.M{"emailLower": l}}})
}

func (r *Users) List(ctx context.Context) ([]*user.User, error) {
	cur, err := r.c.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "usernameLower", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var docs []userDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]*user.User, len(docs))
	for i, d := range docs {
		out[i] = d.toUser()
	}
	return out, nil
}

func dupErr(err error, msg string) error {
	if mongo.IsDuplicateKeyError(err) {
		return errs.Conflict("%s", msg)
	}
	return err
}

func (r *Users) Insert(ctx context.Context, u *user.User) error {
	_, err := r.c.InsertOne(ctx, toUserDoc(u))
	return dupErr(err, "username or email is already taken")
}

func (r *Users) Update(ctx context.Context, u *user.User) error {
	res, err := r.c.ReplaceOne(ctx, bson.M{"_id": u.ID}, toUserDoc(u))
	if err == nil && res.MatchedCount == 0 {
		return errs.NotFound("user")
	}
	return dupErr(err, "email is already taken")
}

func (r *Users) Delete(ctx context.Context, id string) error {
	res, err := r.c.DeleteOne(ctx, bson.M{"_id": id})
	if err == nil && res.DeletedCount == 0 {
		return errs.NotFound("user")
	}
	return err
}

func (r *Users) RecordFailedLogin(ctx context.Context, id string) (int, bool, error) {
	// Single update pipeline: increment + lock when the limit is reached (atomic against parallel logins).
	next := bson.M{"$add": bson.A{"$failedAttempts", 1}}
	var d userDoc
	err := r.c.FindOneAndUpdate(ctx, bson.M{"_id": id},
		mongo.Pipeline{{{Key: "$set", Value: bson.M{
			"failedAttempts": next,
			"isLocked":       bson.M{"$or": bson.A{"$isLocked", bson.M{"$gte": bson.A{next, user.MaxFailedLogins}}}},
		}}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&d)
	if err != nil {
		return 0, false, notFound(err, "user")
	}
	return d.FailedAttempts, d.IsLocked, nil
}

func (r *Users) ResetFailedLogins(ctx context.Context, id string) error {
	_, err := r.c.UpdateByID(ctx, id, bson.M{"$set": bson.M{"failedAttempts": 0}})
	return err
}

func (r *Users) CountWithRole(ctx context.Context, roleID string) (int, error) {
	n, err := r.c.CountDocuments(ctx, bson.M{"roles": roleID})
	return int(n), err
}

// ---------- roles ----------

type roleDoc struct {
	ID          string              `bson:"_id"`
	Name        string              `bson:"name"`
	Description string              `bson:"description"`
	System      bool                `bson:"system"`
	Permissions map[string][]string `bson:"permissions"`
	CreatedAt   time.Time           `bson:"createdAt"`
	UpdatedAt   time.Time           `bson:"updatedAt"`
}

func toRoleDoc(r *role.Role) roleDoc {
	return roleDoc{r.ID, r.Name, r.Description, r.System, r.Permissions, r.CreatedAt, r.UpdatedAt}
}

func (d roleDoc) toRole() *role.Role {
	return &role.Role{ID: d.ID, Name: d.Name, Description: d.Description, System: d.System,
		Permissions: role.Permissions(d.Permissions), CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt}
}

type Roles struct{ c *mongo.Collection }

func (r *Roles) List(ctx context.Context) ([]*role.Role, error) {
	cur, err := r.c.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "system", Value: -1}, {Key: "name", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var docs []roleDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]*role.Role, len(docs))
	for i, d := range docs {
		out[i] = d.toRole()
	}
	return out, nil
}

func (r *Roles) Get(ctx context.Context, id string) (*role.Role, error) {
	var d roleDoc
	if err := r.c.FindOne(ctx, bson.M{"_id": id}).Decode(&d); err != nil {
		return nil, notFound(err, "role")
	}
	return d.toRole(), nil
}

func (r *Roles) Insert(ctx context.Context, x *role.Role) error {
	_, err := r.c.InsertOne(ctx, toRoleDoc(x))
	return dupErr(err, "role ID or name is already taken")
}

func (r *Roles) Update(ctx context.Context, x *role.Role) error {
	_, err := r.c.ReplaceOne(ctx, bson.M{"_id": x.ID}, toRoleDoc(x))
	return dupErr(err, "role name is already taken")
}

func (r *Roles) Delete(ctx context.Context, id string) error {
	_, err := r.c.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *Roles) CountExisting(ctx context.Context, ids []string) (int, error) {
	n, err := r.c.CountDocuments(ctx, bson.M{"_id": bson.M{"$in": ids}})
	return int(n), err
}

func (r *Roles) InsertIfMissing(ctx context.Context, x *role.Role) error {
	_, err := r.c.UpdateOne(ctx, bson.M{"_id": x.ID}, bson.M{"$setOnInsert": toRoleDoc(x)}, options.UpdateOne().SetUpsert(true))
	return err
}
