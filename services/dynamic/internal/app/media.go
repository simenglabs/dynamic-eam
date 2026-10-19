package app

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"eam/services/dynamic/internal/domain/media"
)

type Media struct{ d Deps }

func NewMedia(d Deps) *Media { return &Media{d} }

// Upload: the file type is detected from the first 512 bytes of the file content, not from the client header.
func (s *Media) Upload(ctx context.Context, name string, size int64, r io.Reader, by string) (*media.Media, error) {
	head := make([]byte, 512)
	n, err := io.ReadFull(r, head)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, err
	}
	m, err := media.New(s.d.NewID(), name, http.DetectContentType(head[:n]), size, by, s.d.Now())
	if err != nil {
		return nil, err
	}
	if err := s.d.Files.Write(m.ID, io.MultiReader(bytes.NewReader(head[:n]), r)); err != nil {
		return nil, err
	}
	if err := s.d.MediaRepo.Save(ctx, m); err != nil {
		s.d.Files.Remove(m.ID)
		return nil, err
	}
	return m, nil
}

func (s *Media) Meta(ctx context.Context, ids []string) ([]*media.Media, error) {
	return s.d.MediaRepo.List(ctx, ids)
}

func (s *Media) Open(ctx context.Context, id string) (*media.Media, io.ReadCloser, error) {
	m, err := s.d.MediaRepo.Get(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	rc, err := s.d.Files.Open(id)
	return m, rc, err
}
