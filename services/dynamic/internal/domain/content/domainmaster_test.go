package content

import (
	"reflect"
	"testing"
)

func TestDomainValuesOf(t *testing.T) {
	e := &Entry{Data: map[string]any{"domainCode": "PRIORITY", "values": []any{
		map[string]any{"valueId": "HIGH", "valueDescription": "Tinggi", "sortOrder": 3.0},
		map[string]any{"valueId": "LOW", "valueDescription": "Rendah", "sortOrder": 1.0, "isDefault": true},
		map[string]any{"valueId": "OLD", "valueDescription": "Lama", "isActive": false},
		map[string]any{"valueId": "MED", "valueDescription": "Sedang", "sortOrder": 2.0, "isActive": true},
	}}}
	got := DomainValuesOf(e)
	want := []DomainValue{{ID: "LOW", Description: "Rendah", Default: true}, {ID: "MED", Description: "Sedang"}, {ID: "HIGH", Description: "Tinggi"}}
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
