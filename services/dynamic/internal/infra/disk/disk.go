// Package disk: media.FileStore di disk lokal (MEDIA_DIR).
// ponytail: satu folder lokal; ganti ke S3/MinIO kalau service jalan >1 instance.
package disk

import (
	"io"
	"os"
	"path/filepath"
	"regexp"

	"eam/services/common/errs"
)

type Store struct{ dir string }

func New(dir string) (*Store, error) {
	return &Store{dir}, os.MkdirAll(dir, 0o750)
}

// ID dibuat server (rand.Text: A-Z2-7); apa pun selain itu ditolak supaya tidak bisa keluar dari folder.
var idRe = regexp.MustCompile(`^[A-Za-z0-9]{1,64}$`)

func (s *Store) path(id string) (string, error) {
	if !idRe.MatchString(id) {
		return "", errs.Invalid("id file tidak valid")
	}
	return filepath.Join(s.dir, id), nil
}

func (s *Store) Write(id string, r io.Reader) error {
	p, err := s.path(id)
	if err != nil {
		return err
	}
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		os.Remove(p)
		return err
	}
	return f.Close()
}

func (s *Store) Open(id string) (io.ReadCloser, error) {
	p, err := s.path(id)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if os.IsNotExist(err) {
		return nil, errs.NotFound("file")
	}
	return f, err
}

func (s *Store) Remove(id string) error {
	p, err := s.path(id)
	if err != nil {
		return err
	}
	return os.Remove(p)
}
