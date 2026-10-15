// Package media verwaltet die Videodateien der Aufnahmen auf der Platte.
package media

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Storage struct {
	dir string
}

func New(dir string) (*Storage, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Storage{dir: dir}, nil
}

// ExtForMime liefert eine Dateiendung für die gängigen MediaRecorder-Formate.
func ExtForMime(mime string) string {
	base := strings.TrimSpace(strings.SplitN(mime, ";", 2)[0])
	switch base {
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	case "video/quicktime":
		return ".mov"
	default:
		return ".bin"
	}
}

// IsAllowedMime prüft, ob es sich um ein Videoformat handelt.
func IsAllowedMime(mime string) bool {
	return ExtForMime(mime) != ".bin"
}

// Save schreibt r in eine neue Datei mit zufälligem Namen und gibt Name und Größe zurück.
func (s *Storage) Save(r io.Reader, mime string) (string, int64, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", 0, err
	}
	name := hex.EncodeToString(buf) + ExtForMime(mime)
	tmp, err := os.CreateTemp(s.dir, ".upload-*")
	if err != nil {
		return "", 0, err
	}
	n, err := io.Copy(tmp, r)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp.Name())
		return "", 0, err
	}
	if err := os.Rename(tmp.Name(), filepath.Join(s.dir, name)); err != nil {
		os.Remove(tmp.Name())
		return "", 0, err
	}
	return name, n, nil
}

// Path liefert den Pfad zu einer gespeicherten Datei; verhindert Path-Traversal.
func (s *Storage) Path(name string) (string, error) {
	if name == "" || name != filepath.Base(name) || strings.HasPrefix(name, ".") {
		return "", fmt.Errorf("invalid media name %q", name)
	}
	return filepath.Join(s.dir, name), nil
}

func (s *Storage) Delete(name string) error {
	p, err := s.Path(name)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
