// Package http: HTTP (gin) adapter for administration use cases. JSON DTOs = frontend contract.
package http

import (
	"net/http"
	"time"

	"eam/services/administration/internal/app"
	"eam/services/administration/internal/domain/role"
	"eam/services/administration/internal/domain/user"
	"eam/services/common"
	"eam/services/common/errs"
	"eam/services/common/httpx"
	"github.com/gin-gonic/gin"
)

// RBAC menu keys owned by this service (same as the paths in packages/ui/src/menu.ts).
const (
	PermUsers = "administration/user-management"
	PermRoles = "administration/role"
)

// Authorizer: RBAC port (common.Authz in production).
type Authorizer interface {
	Access(c *gin.Context) (*common.Access, error)
	Require(key, action string) gin.HandlerFunc
	RequireAny(action string, keys ...string) gin.HandlerFunc
	Forget(userID string)
}

type handler struct {
	auth  *app.Auth
	users *app.Users
	roles *app.Roles
	az    Authorizer
}

func Register(rg *gin.RouterGroup, auth *app.Auth, users *app.Users, roles *app.Roles, az Authorizer) {
	h := &handler{auth, users, roles, az}
	rg.POST("/auth/login", h.login)

	a := rg.Group("", common.Auth())
	a.GET("/auth/me", h.me)

	a.GET("/users", az.Require(PermUsers, common.Read), h.listUsers)
	a.GET("/users/:id", az.Require(PermUsers, common.Read), h.getUser)
	a.POST("/users", az.Require(PermUsers, common.Create), h.createUser)
	a.PUT("/users/:id", az.Require(PermUsers, common.Update), h.updateUser)
	a.DELETE("/users/:id", az.Require(PermUsers, common.Delete), h.deleteUser)

	// The role list is also needed by the user form (role picker).
	a.GET("/roles", az.RequireAny(common.Read, PermRoles, PermUsers), h.listRoles)
	a.GET("/roles/:id", az.Require(PermRoles, common.Read), h.getRole)
	a.POST("/roles", az.Require(PermRoles, common.Create), h.createRole)
	a.PUT("/roles/:id", az.Require(PermRoles, common.Update), h.updateRole)
	a.DELETE("/roles/:id", az.Require(PermRoles, common.Delete), h.deleteRole)
}

func (h *handler) actor(c *gin.Context) (app.Actor, bool) {
	acc, err := h.az.Access(c)
	if err != nil {
		httpx.Error(c, err)
		return app.Actor{}, false
	}
	return app.Actor{UserID: acc.UserID, SuperAdmin: acc.SuperAdmin}, true
}

// ---------- DTO ----------

type userDTO struct {
	ID             string    `json:"id"`
	Username       string    `json:"username"`
	DisplayName    string    `json:"displayName"`
	Email          string    `json:"email"`
	Roles          []string  `json:"roles"`
	IsActive       bool      `json:"isActive"`
	IsLocked       bool      `json:"isLocked"`
	FailedAttempts int       `json:"failedAttempts"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func toUserDTO(u *user.User) userDTO {
	return userDTO{u.ID, u.Username, u.DisplayName, u.Email, u.Roles, u.Active, u.Locked, u.FailedAttempts, u.CreatedAt, u.UpdatedAt}
}

type userInput struct {
	Username    string   `json:"username"`
	DisplayName string   `json:"displayName"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Password    string   `json:"password"`
	IsActive    *bool    `json:"isActive"`
	IsLocked    *bool    `json:"isLocked"`
}

type roleDTO struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	System      bool                `json:"system"`
	Permissions map[string][]string `json:"permissions"`
	CreatedAt   time.Time           `json:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt"`
}

func toRoleDTO(r *role.Role) roleDTO {
	return roleDTO{r.ID, r.Name, r.Description, r.System, r.Permissions, r.CreatedAt, r.UpdatedAt}
}

func mapSlice[T, D any](xs []T, f func(T) D) []D {
	out := make([]D, len(xs))
	for i, x := range xs {
		out[i] = f(x)
	}
	return out
}

// ---------- auth ----------

func (h *handler) login(c *gin.Context) {
	var in struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if !httpx.Bind(c, &in) {
		return
	}
	res, err := h.auth.Login(c, in.Username, in.Password)
	if err != nil {
		if isUnauthorized(err) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error(), "attemptsRemaining": res.AttemptsRemaining})
			return
		}
		httpx.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": res.Token, "expiresIn": res.ExpiresIn, "user": toUserDTO(res.User)})
}

func (h *handler) me(c *gin.Context) {
	acc, err := h.az.Access(c)
	if err != nil {
		httpx.Error(c, err)
		return
	}
	cl := common.CurrentUser(c)
	c.JSON(http.StatusOK, gin.H{"id": cl.Subject, "username": cl.Username, "name": cl.Name, "access": acc})
}

// ---------- users ----------

func (h *handler) listUsers(c *gin.Context) {
	list, err := h.users.List(c)
	if err != nil {
		httpx.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, mapSlice(list, toUserDTO))
}

func (h *handler) getUser(c *gin.Context) {
	u, err := h.users.Get(c, c.Param("id"))
	if err != nil {
		httpx.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, toUserDTO(u))
}

func (h *handler) createUser(c *gin.Context) {
	var in userInput
	actor, ok := h.actor(c)
	if !ok || !httpx.Bind(c, &in) {
		return
	}
	u, err := h.users.Create(c, actor, user.NewParams{Username: in.Username, DisplayName: in.DisplayName, Email: in.Email,
		Password: in.Password, Roles: in.Roles, Inactive: in.IsActive != nil && !*in.IsActive})
	if err != nil {
		httpx.Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, toUserDTO(u))
}

func (h *handler) updateUser(c *gin.Context) {
	var in userInput
	actor, ok := h.actor(c)
	if !ok || !httpx.Bind(c, &in) {
		return
	}
	u, err := h.users.Update(c, actor, c.Param("id"), user.EditParams{DisplayName: in.DisplayName, Email: in.Email,
		Password: in.Password, Roles: in.Roles, Active: in.IsActive, Locked: in.IsLocked})
	h.az.Forget(c.Param("id"))
	if err != nil {
		httpx.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, toUserDTO(u))
}

func (h *handler) deleteUser(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	if err := h.users.Delete(c, actor, c.Param("id")); err != nil {
		httpx.Error(c, err)
		return
	}
	h.az.Forget(c.Param("id"))
	c.Status(http.StatusNoContent)
}

// ---------- roles ----------

type roleInput struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Permissions map[string][]string `json:"permissions"`
}

func (in roleInput) toApp() app.RoleInput {
	return app.RoleInput{ID: in.ID, Name: in.Name, Description: in.Description, Permissions: role.Permissions(in.Permissions)}
}

func (h *handler) listRoles(c *gin.Context) {
	list, err := h.roles.List(c)
	if err != nil {
		httpx.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, mapSlice(list, toRoleDTO))
}

func (h *handler) getRole(c *gin.Context) {
	r, err := h.roles.Get(c, c.Param("id"))
	if err != nil {
		httpx.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, toRoleDTO(r))
}

func (h *handler) createRole(c *gin.Context) {
	var in roleInput
	if !httpx.Bind(c, &in) {
		return
	}
	r, err := h.roles.Create(c, in.toApp())
	if err != nil {
		httpx.Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, toRoleDTO(r))
}

func (h *handler) updateRole(c *gin.Context) {
	var in roleInput
	if !httpx.Bind(c, &in) {
		return
	}
	r, err := h.roles.Update(c, c.Param("id"), in.toApp())
	h.az.Forget("") // permissions of every user holding this role change
	if err != nil {
		httpx.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, toRoleDTO(r))
}

func (h *handler) deleteRole(c *gin.Context) {
	if err := h.roles.Delete(c, c.Param("id")); err != nil {
		httpx.Error(c, err)
		return
	}
	h.az.Forget("")
	c.Status(http.StatusNoContent)
}

func isUnauthorized(err error) bool { return errs.KindOf(err) == errs.KindUnauthorized }
