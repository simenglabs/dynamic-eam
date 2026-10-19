package common

import (
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// RBAC actions.
const (
	Read     = "read"
	Create   = "create"
	Update   = "update"
	Delete   = "delete"
	Download = "download"
)

var Actions = []string{Read, Create, Update, Delete, Download}

// Wildcard: permission for all menus.
const AllMenus = "*"

// Roles are stored in collection "roles". Permissions: menu key ("assets/tnd", "ct/<uid>", "*") -> actions.
type Role struct {
	ID          string              `json:"id" bson:"_id"`
	Name        string              `json:"name" bson:"name"`
	Description string              `json:"description" bson:"description"`
	System      bool                `json:"system" bson:"system"` // cannot be edited/deleted
	Permissions map[string][]string `json:"permissions" bson:"permissions"`
	CreatedAt   time.Time           `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt" bson:"updatedAt"`
}

// Access = effective permissions of one user (union of all their roles).
type Access struct {
	UserID      string              `json:"userId"`
	Roles       []string            `json:"roles"`
	SuperAdmin  bool                `json:"superAdmin"`
	Permissions map[string][]string `json:"permissions"`
}

func (a *Access) Can(key, action string) bool {
	return a.SuperAdmin || slices.Contains(a.Permissions[key], action) || slices.Contains(a.Permissions[AllMenus], action)
}

// Authz loads user roles & status from Mongo (same DB for all services).
// ponytail: in-process cache of 15s per user; role changes take <=15s to apply. Use pub/sub if instant is needed.
type Authz struct {
	db    *mongo.Database
	mu    sync.Mutex
	cache map[string]cached
}

type cached struct {
	access *Access
	err    error
	at     time.Time
}

const authzTTL = 15 * time.Second

func NewAuthz(db *mongo.Database) *Authz { return &Authz{db: db, cache: map[string]cached{}} }

var errInactive = &statusErr{http.StatusUnauthorized, "account is inactive or locked"}

type statusErr struct {
	code int
	msg  string
}

func (e *statusErr) Error() string { return e.msg }

func (z *Authz) Access(c *gin.Context) (*Access, error) {
	uid := CurrentUser(c).Subject
	z.mu.Lock()
	hit, ok := z.cache[uid]
	z.mu.Unlock()
	if ok && time.Since(hit.at) < authzTTL {
		return hit.access, hit.err
	}
	acc, err := z.load(c, uid)
	z.mu.Lock()
	z.cache[uid] = cached{acc, err, time.Now()}
	z.mu.Unlock()
	return acc, err
}

func (z *Authz) load(c *gin.Context, uid string) (*Access, error) {
	var u struct {
		Roles    []string `bson:"roles"`
		IsActive bool     `bson:"isActive"`
		IsLocked bool     `bson:"isLocked"`
	}
	if err := z.db.Collection("users").FindOne(c, bson.M{"_id": uid}).Decode(&u); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errInactive
		}
		return nil, err
	}
	if !u.IsActive || u.IsLocked {
		return nil, errInactive
	}
	acc := &Access{UserID: uid, Roles: u.Roles, SuperAdmin: slices.Contains(u.Roles, RoleSuperAdmin), Permissions: map[string][]string{}}
	cur, err := z.db.Collection("roles").Find(c, bson.M{"_id": bson.M{"$in": u.Roles}})
	if err != nil {
		return nil, err
	}
	var roles []Role
	if err := cur.All(c, &roles); err != nil {
		return nil, err
	}
	for _, r := range roles {
		for key, actions := range r.Permissions {
			for _, a := range actions {
				if !slices.Contains(acc.Permissions[key], a) {
					acc.Permissions[key] = append(acc.Permissions[key], a)
				}
			}
		}
	}
	return acc, nil
}

func (z *Authz) abort(c *gin.Context, err error) {
	if se, ok := err.(*statusErr); ok {
		c.AbortWithStatusJSON(se.code, gin.H{"error": se.msg})
		return
	}
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to load access rights"})
}

// Can is used in handlers for dynamic keys (e.g. "ct/"+uid). Returns false with the response already written when denied.
func (z *Authz) Can(c *gin.Context, key, action string) bool {
	acc, err := z.Access(c)
	if err != nil {
		z.abort(c, err)
		return false
	}
	if !acc.Can(key, action) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "you don't have permission to " + action + " on this menu"})
		return false
	}
	return true
}

// Require: static permission middleware. Mount after Auth().
func (z *Authz) Require(key, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if z.Can(c, key, action) {
			c.Next()
		}
	}
}

// RequireAny: passes if the user has that action on any of the menus.
func (z *Authz) RequireAny(action string, keys ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		acc, err := z.Access(c)
		if err != nil {
			z.abort(c, err)
			return
		}
		if !slices.ContainsFunc(keys, func(k string) bool { return acc.Can(k, action) }) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "you don't have permission to " + action + " on this menu"})
			return
		}
		c.Next()
	}
}

// SuperAdmin: superadmin-only middleware (Content-Type Builder, etc.).
func (z *Authz) SuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		acc, err := z.Access(c)
		if err != nil {
			z.abort(c, err)
			return
		}
		if !acc.SuperAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "superadmin only"})
			return
		}
		c.Next()
	}
}

// Forget clears the user cache (called when a role/user changes in the same service).
func (z *Authz) Forget(uid string) {
	z.mu.Lock()
	defer z.mu.Unlock()
	if uid == "" {
		clear(z.cache)
		return
	}
	delete(z.cache, uid)
}
