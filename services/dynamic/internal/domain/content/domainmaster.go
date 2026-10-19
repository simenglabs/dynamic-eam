package content

import (
	"cmp"
	"fmt"
	"slices"
)

// Master Domain (built-in content type, seed master.json): option source for TDomain fields.
const (
	DomainUID       = "master-domain"
	DomainCodeField = "domainCode"
	domainValues    = "values"
)

type DomainValue struct {
	ID          string
	Description string
	Default     bool
}

// DomainValuesOf: active values (empty isActive = active) of a domain entry, ordered by sortOrder.
func DomainValuesOf(e *Entry) []DomainValue {
	list, _ := e.Data[domainValues].([]any)
	type row struct {
		v    DomainValue
		sort float64
	}
	var rows []row
	for _, x := range list {
		m, ok := x.(map[string]any)
		if !ok || m["isActive"] == false || m["valueId"] == nil {
			continue
		}
		sort, _ := m["sortOrder"].(float64)
		def, _ := m["isDefault"].(bool)
		desc, _ := m["valueDescription"].(string)
		rows = append(rows, row{DomainValue{ID: fmt.Sprint(m["valueId"]), Description: desc, Default: def}, sort})
	}
	slices.SortStableFunc(rows, func(a, b row) int { return cmp.Compare(a.sort, b.sort) })
	out := make([]DomainValue, len(rows))
	for i, r := range rows {
		out[i] = r.v
	}
	return out
}

func DefaultDomainValue(vals []DomainValue) string {
	if i := slices.IndexFunc(vals, func(v DomainValue) bool { return v.Default }); i >= 0 {
		return vals[i].ID
	}
	return ""
}
