package media

import "testing"

func TestDisposition(t *testing.T) {
	cases := map[string]string{
		"image/png":                "inline",
		"image/jpeg":               "inline",
		"application/pdf":          "inline",
		"image/svg+xml":            "attachment", // may contain script
		"text/html; charset=utf-8": "attachment",
		"application/octet-stream": "attachment",
	}
	for mime, want := range cases {
		if got := (Media{Mime: mime}).Disposition(); got != want {
			t.Errorf("%s: %s, want %s", mime, got, want)
		}
	}
}

func TestSafeFilename(t *testing.T) {
	m := Media{Name: "../../etc/pa\"ss\r\nwd.png"}
	if got := m.SafeName(); got != "pass wd.png" && got != "passwd.png" {
		t.Errorf("SafeName = %q", got)
	}
}

func TestNewRejectsTooLarge(t *testing.T) {
	if _, err := New("id", "a.png", "image/png", MaxSize+1, "u1", t0); err == nil {
		t.Error("file > MaxSize must be rejected")
	}
	if _, err := New("id", "a.png", "image/png", 10, "u1", t0); err != nil {
		t.Error(err)
	}
}
