package content

import (
	"slices"

	"eam/services/common/errs"
)

// Masters & profile fields used for per-site record scoping and auto fill.
const (
	SiteUID             = "master-site"
	OrganizationUID     = "master-organization"
	ProfileSites        = "sites"
	ProfileDefaultSite  = "defaultSite"
	ProfileName         = "name" // user name (auto filled), label of relations to the profile
	ProfileOrganization = "organization"
)

// Scoped: records of this content type are restricted per site for the actor (superadmin is not).
func (ct *ContentType) Scoped(a Actor) bool { return ct.ScopeField != "" && !a.Super }

// InScope: the entry is visible to the actor. Records without a site are hidden from restricted users (safe by default).
func (ct *ContentType) InScope(e *Entry, a Actor) bool {
	if !ct.Scoped(a) {
		return true
	}
	v, ok := e.Data[ct.ScopeField].(string)
	return ok && slices.Contains(a.Sites, v)
}

// CheckScope: the site value of the record being saved must be one of the actor's accessible sites.
func (ct *ContentType) CheckScope(data map[string]any, a Actor) error {
	if !ct.Scoped(a) {
		return nil
	}
	v, _ := data[ct.ScopeField].(string)
	if v == "" {
		return errs.Invalid("%s is required (records are restricted per site)", ct.fieldLabel(ct.ScopeField))
	}
	if !slices.Contains(a.Sites, v) {
		return errs.Forbidden("the selected site is outside your access")
	}
	return nil
}

func (ct *ContentType) fieldLabel(name string) string {
	if i := slices.IndexFunc(ct.Fields, func(f Field) bool { return f.Name == name }); i >= 0 {
		return ct.Fields[i].Label
	}
	return name
}

func (ct *ContentType) validateScope() error {
	if ct.ScopeField == "" {
		return nil
	}
	i := slices.IndexFunc(ct.Fields, func(f Field) bool { return f.Name == ct.ScopeField })
	if i < 0 || ct.Fields[i].Type != TRelation || ct.Fields[i].Target != SiteUID || ct.Fields[i].Multiple {
		return errs.Invalid("the site scope field must be a single relation to %s", SiteUID)
	}
	return nil
}
