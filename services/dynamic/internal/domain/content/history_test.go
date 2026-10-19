package content

import (
	"reflect"
	"testing"
)

func TestDiffFieldLevel(t *testing.T) {
	before := map[string]any{"plate": "L 1", "year": 2020.0, "tags": []any{"a"}, "gone": "x"}
	after := map[string]any{"plate": "L 1", "year": 2021.0, "tags": []any{"a", "b"}, "new": true}
	got := Diff(before, after)
	want := []Change{
		{Field: "gone", From: "x", To: nil},
		{Field: "new", From: nil, To: true},
		{Field: "tags", From: []any{"a"}, To: []any{"a", "b"}},
		{Field: "year", From: 2020.0, To: 2021.0},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Diff =\n%+v\nwant\n%+v", got, want)
	}
}

func TestDiffCreateAndNoChange(t *testing.T) {
	if got := Diff(nil, map[string]any{"a": 1.0}); len(got) != 1 || got[0].From != nil || got[0].To != 1.0 {
		t.Errorf("create: %+v", got)
	}
	if got := Diff(map[string]any{"a": "x"}, map[string]any{"a": "x"}); len(got) != 0 {
		t.Errorf("no change: %+v", got)
	}
	// []string vs []any with the same content = unchanged (Mongo returns []any).
	if got := Diff(map[string]any{"m": []any{"1"}}, map[string]any{"m": []string{"1"}}); len(got) != 0 {
		t.Errorf("different slice type, same content: %+v", got)
	}
}

func TestNewRevision(t *testing.T) {
	e := &Entry{ID: "e1", Version: 2}
	r := NewRevision(e, ActionUpdate, Actor{ID: "u1", Name: "Budi"}, now(), []Change{{Field: "a"}}, "note")
	if r.EntryID != "e1" || r.Version != 2 || r.Action != ActionUpdate || r.ByName != "Budi" || r.Note != "note" {
		t.Errorf("revision: %+v", r)
	}
}
