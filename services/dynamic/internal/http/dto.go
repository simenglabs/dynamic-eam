package http

import (
	"encoding/json"
	"time"

	"eam/services/dynamic/internal/domain/content"
)

// DTO JSON = kontrak frontend (apps/dynamic/app/_lib/types.ts).

type fieldDTO struct {
	Name       string         `json:"name"`
	Label      string         `json:"label"`
	Type       string         `json:"type"`
	Required   bool           `json:"required"`
	Unique     bool           `json:"unique,omitempty"`
	Default    any            `json:"default,omitempty"`
	Options    []string       `json:"options,omitempty"`
	Target     string         `json:"target,omitempty"`
	Multiple   bool           `json:"multiple,omitempty"`
	Component  string         `json:"component,omitempty"`
	Repeatable bool           `json:"repeatable,omitempty"`
	UniqueBy   string         `json:"uniqueBy,omitempty"`
	Domain     string         `json:"domain,omitempty"`
	AutoFill   *autoFillDTO   `json:"autoFill,omitempty"`
	Where      []conditionDTO `json:"where,omitempty"`
	From       string         `json:"from,omitempty"`
}

type conditionDTO struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

type autoFillDTO struct {
	Source  string `json:"source"`
	From    string `json:"from,omitempty"`
	Field   string `json:"field,omitempty"`
	Lock    bool   `json:"lock,omitempty"`
	Pattern string `json:"pattern,omitempty"`
}

type rowDTO struct {
	Columns int      `json:"columns"`
	Fields  []string `json:"fields"`
}

type sectionDTO struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Rows  []rowDTO `json:"rows"`
}

type layoutDTO struct {
	Sections []sectionDTO `json:"sections"`
}

type stateDTO struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Color    string `json:"color"`
	Editable bool   `json:"editable"`
}

type transitionDTO struct {
	Key    string     `json:"key"`
	Label  string     `json:"label"`
	From   []string   `json:"from"`
	To     string     `json:"to"`
	Fields []fieldDTO `json:"fields,omitempty"`
}

type viewDTO struct {
	Key    string   `json:"key"`
	Label  string   `json:"label"`
	Parent string   `json:"parent"`
	States []string `json:"states"`
}

type workflowDTO struct {
	States      []stateDTO      `json:"states"`
	Initial     string          `json:"initial"`
	Transitions []transitionDTO `json:"transitions"`
	Views       []viewDTO       `json:"views"`
}

type contentTypeDTO struct {
	UID          string       `json:"uid"`
	Label        string       `json:"label"`
	Parent       string       `json:"parent"`
	Hidden       bool         `json:"hidden"`
	ScopeField   string       `json:"scopeField,omitempty"`
	DisplayField string       `json:"displayField"`
	Fields       []fieldDTO   `json:"fields"`
	Layout       *layoutDTO   `json:"layout"`
	Workflow     *workflowDTO `json:"workflow"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
}

type componentDTO struct {
	UID       string     `json:"uid"`
	Label     string     `json:"label"`
	Fields    []fieldDTO `json:"fields"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

func mapAll[T, D any](xs []T, f func(T) D) []D {
	out := make([]D, len(xs))
	for i, x := range xs {
		out[i] = f(x)
	}
	return out
}

func toFieldDTO(f content.Field) fieldDTO {
	d := fieldDTO{f.Name, f.Label, f.Type, f.Required, f.Unique, f.Default, f.Options, f.Target, f.Multiple, f.Component, f.Repeatable, f.UniqueBy, f.Domain, nil,
		mapAll(f.Where, func(c content.Condition) conditionDTO { return conditionDTO{c.Field, c.Value} }), f.From}
	if af := f.AutoFill; af != nil {
		d.AutoFill = &autoFillDTO{af.Source, af.From, af.Field, af.Lock, af.Pattern}
	}
	return d
}

func (d fieldDTO) toField() content.Field {
	return content.Field{Name: d.Name, Label: d.Label, Type: d.Type, Required: d.Required, Unique: d.Unique, Default: d.Default,
		Options: d.Options, Target: d.Target, Multiple: d.Multiple, Component: d.Component, Repeatable: d.Repeatable, UniqueBy: d.UniqueBy, Domain: d.Domain,
		From: d.From, Where: mapAll(d.Where, func(c conditionDTO) content.Condition { return content.Condition{Field: c.Field, Value: c.Value} }),
		AutoFill: autoFillFromDTO(d.AutoFill)}
}

func autoFillFromDTO(d *autoFillDTO) *content.AutoFill {
	if d == nil {
		return nil
	}
	return &content.AutoFill{Source: d.Source, From: d.From, Field: d.Field, Lock: d.Lock, Pattern: d.Pattern}
}

func toContentTypeDTO(ct *content.ContentType) contentTypeDTO {
	d := contentTypeDTO{UID: ct.UID, Label: ct.Label, Parent: ct.Parent, Hidden: ct.Hidden, ScopeField: ct.ScopeField, DisplayField: ct.DisplayField,
		Fields: mapAll(ct.Fields, toFieldDTO), CreatedAt: ct.CreatedAt, UpdatedAt: ct.UpdatedAt}
	if ct.Layout != nil {
		d.Layout = &layoutDTO{mapAll(ct.Layout.Sections, func(s content.Section) sectionDTO {
			return sectionDTO{s.ID, s.Title, mapAll(s.Rows, func(r content.Row) rowDTO { return rowDTO{r.Columns, r.Fields} })}
		})}
	}
	if w := ct.Workflow; w != nil {
		d.Workflow = &workflowDTO{
			States:  mapAll(w.States, func(s content.State) stateDTO { return stateDTO{s.Key, s.Label, s.Color, s.Editable} }),
			Initial: w.Initial,
			Transitions: mapAll(w.Transitions, func(t content.Transition) transitionDTO {
				return transitionDTO{t.Key, t.Label, t.From, t.To, mapAll(t.Fields, toFieldDTO)}
			}),
			Views: mapAll(w.Views, func(v content.View) viewDTO { return viewDTO{v.Key, v.Label, v.Parent, v.States} }),
		}
	}
	return d
}

func (d contentTypeDTO) toContentType() *content.ContentType {
	ct := &content.ContentType{UID: d.UID, Label: d.Label, Parent: d.Parent, Hidden: d.Hidden, ScopeField: d.ScopeField, DisplayField: d.DisplayField, Fields: mapAll(d.Fields, fieldDTO.toField)}
	if d.Layout != nil {
		ct.Layout = &content.Layout{Sections: mapAll(d.Layout.Sections, func(s sectionDTO) content.Section {
			return content.Section{ID: s.ID, Title: s.Title, Rows: mapAll(s.Rows, func(r rowDTO) content.Row { return content.Row{Columns: r.Columns, Fields: r.Fields} })}
		})}
	}
	if w := d.Workflow; w != nil {
		ct.Workflow = &content.Workflow{
			States: mapAll(w.States, func(s stateDTO) content.State {
				return content.State{Key: s.Key, Label: s.Label, Color: s.Color, Editable: s.Editable}
			}),
			Initial: w.Initial,
			Transitions: mapAll(w.Transitions, func(t transitionDTO) content.Transition {
				return content.Transition{Key: t.Key, Label: t.Label, From: t.From, To: t.To, Fields: mapAll(t.Fields, func(f fieldDTO) content.Field { return f.toField() })}
			}),
			Views: mapAll(w.Views, func(v viewDTO) content.View {
				return content.View{Key: v.Key, Label: v.Label, Parent: v.Parent, States: v.States}
			}),
		}
	}
	return ct
}

func toComponentDTO(c *content.Component) componentDTO {
	return componentDTO{c.UID, c.Label, mapAll(c.Fields, toFieldDTO), c.CreatedAt, c.UpdatedAt}
}

// Entri = field data di level atas + metadata (nama metadata dicadangkan di domain).
func toEntryDTO(e *content.Entry) map[string]any {
	out := make(map[string]any, len(e.Data)+8)
	for k, v := range e.Data {
		out[k] = v
	}
	out["id"], out["createdAt"], out["updatedAt"], out["createdBy"], out["updatedBy"] = e.ID, e.CreatedAt, e.UpdatedAt, e.CreatedBy, e.UpdatedBy
	out["version"] = e.Version
	if e.Status != "" {
		out["workflowStatus"] = e.Status
		out["workflowHistory"] = mapAll(e.History, func(h content.StatusChange) map[string]any {
			return map[string]any{"from": h.From, "to": h.To, "transition": h.Transition, "by": h.By, "at": h.At, "note": h.Note}
		})
	}
	return out
}

// SeedEntries: data bawaan satu content type; Key = field pencocok (data dengan nilai Key yang sama tidak dibuat ulang).
type SeedEntries struct {
	UID  string           `json:"uid"`
	Key  string           `json:"key"`
	Rows []map[string]any `json:"rows"`
}

// DecodeSeed membaca file seed (format JSON sama dengan API) menjadi domain object + data bawaan.
func DecodeSeed(b []byte) ([]*content.Component, []*content.ContentType, []SeedEntries, error) {
	var in struct {
		Components   []componentDTO   `json:"components"`
		ContentTypes []contentTypeDTO `json:"contentTypes"`
		Entries      []SeedEntries    `json:"entries"`
	}
	if err := json.Unmarshal(b, &in); err != nil {
		return nil, nil, nil, err
	}
	comps := mapAll(in.Components, func(c componentDTO) *content.Component {
		return &content.Component{UID: c.UID, Label: c.Label, Fields: mapAll(c.Fields, fieldDTO.toField)}
	})
	return comps, mapAll(in.ContentTypes, contentTypeDTO.toContentType), in.Entries, nil
}

type changeDTO struct {
	Field string `json:"field"`
	From  any    `json:"from"`
	To    any    `json:"to"`
}

type revisionDTO struct {
	Version    int            `json:"version"`
	Action     string         `json:"action"`
	By         string         `json:"by"`
	ByName     string         `json:"byName"`
	At         time.Time      `json:"at"`
	Changes    []changeDTO    `json:"changes"`
	Note       string         `json:"note,omitempty"`
	Transition string         `json:"transition,omitempty"`
	Data       map[string]any `json:"data,omitempty"`
}

func toRevisionDTO(r *content.Revision) revisionDTO {
	return revisionDTO{r.Version, r.Action, r.By, r.ByName, r.At,
		mapAll(r.Changes, func(c content.Change) changeDTO { return changeDTO{c.Field, c.From, c.To} }), r.Note, r.Transition, r.Data}
}
