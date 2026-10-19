// Package media: uploaded files referenced by media-type fields.
package media

import (
	"path/filepath"
	"slices"
	"strings"
	"time"

	"eam/services/common/errs"
)

const MaxSize = 20 << 20 // 20MB

type Media struct {
	ID        string
	Name      string
	Mime      string // detected from file content, not from the client
	Size      int64
	CreatedBy string
	CreatedAt time.Time
}

func New(id, name, mime string, size int64, by string, now time.Time) (*Media, error) {
	if size > MaxSize {
		return nil, errs.Invalid("maximum file size is %dMB", MaxSize>>20)
	}
	return &Media{ID: id, Name: filepath.Base(name), Mime: mime, Size: size, CreatedBy: by, CreatedAt: now}, nil
}

// Types safe to display inline. SVG/HTML intentionally excluded (may contain script).
var inline = []string{"image/png", "image/jpeg", "image/gif", "image/webp", "application/pdf"}

func (m Media) Disposition() string {
	if slices.Contains(inline, strings.TrimSpace(strings.Split(m.Mime, ";")[0])) {
		return "inline"
	}
	return "attachment"
}

// SafeName: file name safe for the Content-Disposition header.
func (m Media) SafeName() string {
	return strings.NewReplacer(`"`, "", "\r", "", "\n", "").Replace(filepath.Base(m.Name))
}
