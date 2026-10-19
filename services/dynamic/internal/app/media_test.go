package app

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"eam/services/common/errs"
	"eam/services/dynamic/internal/domain/media"
)

func TestUploadDetectsMimeFromContent(t *testing.T) {
	e := newEnv()
	m, err := e.media.Upload(e.ctx, "foto.txt", 8, strings.NewReader("\x89PNG\r\n\x1a\n"), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if m.Mime != "image/png" || m.Name != "foto.txt" || m.CreatedBy != "u1" {
		t.Errorf("media: %+v", m)
	}
	got, rc, err := e.media.Open(e.ctx, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(rc)
	rc.Close()
	if got.ID != m.ID || !bytes.HasPrefix(body, []byte("\x89PNG")) {
		t.Error("file content must be stored intact")
	}
	meta, _ := e.media.Meta(e.ctx, []string{m.ID, "nope"})
	if len(meta) != 1 {
		t.Errorf("meta: %+v", meta)
	}
}

func TestUploadTooLarge(t *testing.T) {
	e := newEnv()
	if _, err := e.media.Upload(e.ctx, "besar.bin", media.MaxSize+1, strings.NewReader("x"), "u1"); errs.KindOf(err) != errs.KindInvalid {
		t.Errorf("err = %v", err)
	}
	if e.files.Count() != 0 {
		t.Error("a rejected file must not be stored")
	}
}
