package content

import (
	"testing"

	"eam/services/common/errs"
)

func scoped() ContentType {
	ct := vehicle()
	ct.Fields = append(ct.Fields, Field{Name: "site", Label: "Site", Type: TRelation, Target: SiteUID})
	ct.ScopeField = "site"
	return ct
}

func TestScopeFieldValidation(t *testing.T) {
	cat := catalog()
	cat.ContentTypes[SiteUID] = true
	ok := scoped()
	if err := ok.Validate(cat); err != nil {
		t.Fatal(err)
	}
	for name, f := range map[string]string{"missing field": "nope", "not a relation to site": "location", "text": "plate"} {
		ct := scoped()
		ct.ScopeField = f
		if errs.KindOf(ct.Validate(cat)) != errs.KindInvalid {
			t.Errorf("%s: must be invalid", name)
		}
	}
}

func TestInScope(t *testing.T) {
	ct := scoped()
	in := &Entry{Data: map[string]any{"site": "S1"}}
	out := &Entry{Data: map[string]any{"site": "S2"}}
	none := &Entry{Data: map[string]any{}}
	user := Actor{ID: "u", Sites: []string{"S1"}}
	if !ct.InScope(in, user) || ct.InScope(out, user) || ct.InScope(none, user) {
		t.Error("user only sees accessible sites; records without a site are hidden")
	}
	if !ct.InScope(out, Actor{Super: true}) {
		t.Error("superadmin is not restricted")
	}
	plain := vehicle()
	if !plain.InScope(out, user) {
		t.Error("content type without scope is not restricted")
	}
	if !ct.Scoped(user) || ct.Scoped(Actor{Super: true}) || plain.Scoped(user) {
		t.Error("Scoped is wrong")
	}
}
