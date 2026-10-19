package content

import (
	"reflect"
	"testing"
)

func TestDomainValuesOf(t *testing.T) {
	e := &Entry{Data: map[string]any{"domainCode": "PRIORITY", "values": []any{
		map[string]any{"valueId": "HIGH", "valueDescription": "High", "sortOrder": 3.0},
		map[string]any{"valueId": "LOW", "valueDescription": "Low", "sortOrder": 1.0, "isDefault": true},
		map[string]any{"valueId": "OLD", "valueDescription": "Old", "isActive": false},
		map[string]any{"valueId": "MED", "valueDescription": "Medium", "sortOrder": 2.0, "isActive": true},
	}}}
	got := DomainValuesOf(e)
	want := []DomainValue{{ID: "LOW", Description: "Low", Default: true}, {ID: "MED", Description: "Medium"}, {ID: "HIGH", Description: "High"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
	if d := DefaultDomainValue(got); d != "LOW" {
		t.Errorf("default = %q", d)
	}
	if d := DefaultDomainValue(nil); d != "" {
		t.Errorf("no values = %q", d)
	}
}
