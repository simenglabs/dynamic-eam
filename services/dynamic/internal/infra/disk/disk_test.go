package disk

import (
	"io"
	"strings"
	"testing"
)

func TestWriteOpenRemove(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Write("ABC123", strings.NewReader("isi")); err != nil {
		t.Fatal(err)
	}
	rc, err := s.Open("ABC123")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(rc)
	rc.Close()
	if string(b) != "isi" {
		t.Errorf("content = %q", b)
	}
	if err := s.Remove("ABC123"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open("ABC123"); err == nil {
		t.Error("file must be deleted")
	}
}

func TestRejectsPathTraversal(t *testing.T) {
	s, _ := New(t.TempDir())
	for _, id := range []string{"../x", "a/b", "", ".."} {
		if err := s.Write(id, strings.NewReader("x")); err == nil {
			t.Errorf("id %q must be rejected", id)
		}
	}
}
