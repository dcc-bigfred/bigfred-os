package etcdir

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// DefaultDir is the hub configuration directory on persistent storage.
	DefaultDir = "/data/etc"
	// MaxFileSize is the largest file the UI may read or write.
	MaxFileSize = 512 * 1024
)

var (
	ErrNotFound    = errors.New("file not found")
	ErrInvalidPath = errors.New("invalid path")
	ErrTooLarge    = errors.New("file too large")
	ErrNotFile     = errors.New("not a regular file")
)

// Entry describes one editable file under the etc root.
type Entry struct {
	Path     string    `json:"path"`
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

// FileContent is the text body of a configuration file.
type FileContent struct {
	Path     string    `json:"path"`
	Content  string    `json:"content"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

// List returns regular files under root (recursive).
func List(root string) ([]Entry, error) {
	rootAbs, err := absRoot(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(rootAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s: not a directory", rootAbs)
	}

	var out []Entry
	err = filepath.WalkDir(rootAbs, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}

		rel, err := filepath.Rel(rootAbs, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil
		}

		fi, err := d.Info()
		if err != nil {
			return nil
		}

		out = append(out, Entry{
			Path:     filepath.ToSlash(rel),
			Name:     d.Name(),
			Size:     fi.Size(),
			Modified: fi.ModTime(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Path < out[j].Path
	})
	if out == nil {
		out = []Entry{}
	}
	return out, nil
}

// Read returns the UTF-8 text content of relPath under root.
func Read(root, relPath string) (FileContent, error) {
	abs, err := resolveFile(root, relPath)
	if err != nil {
		return FileContent{}, err
	}

	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return FileContent{}, ErrNotFound
		}
		return FileContent{}, err
	}
	if !info.Mode().IsRegular() {
		return FileContent{}, ErrNotFile
	}
	if info.Size() > MaxFileSize {
		return FileContent{}, ErrTooLarge
	}

	raw, err := os.ReadFile(abs)
	if err != nil {
		return FileContent{}, err
	}

	rel := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(relPath))))
	return FileContent{
		Path:     rel,
		Content:  string(raw),
		Size:     info.Size(),
		Modified: info.ModTime(),
	}, nil
}

// Write saves content to relPath under root, preserving mode bits when possible.
func Write(root, relPath, content string) (FileContent, error) {
	if len(content) > MaxFileSize {
		return FileContent{}, ErrTooLarge
	}

	abs, err := resolveFile(root, relPath)
	if err != nil {
		return FileContent{}, err
	}

	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return FileContent{}, err
	}

	mode := fs.FileMode(0o640)
	if info, err := os.Stat(abs); err == nil {
		if !info.Mode().IsRegular() {
			return FileContent{}, ErrNotFile
		}
		mode = info.Mode().Perm()
	}

	tmp, err := os.CreateTemp(filepath.Dir(abs), ".etc-edit-*")
	if err != nil {
		return FileContent{}, err
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }

	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		cleanup()
		return FileContent{}, err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		cleanup()
		return FileContent{}, err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return FileContent{}, err
	}
	if err := os.Rename(tmpPath, abs); err != nil {
		cleanup()
		return FileContent{}, err
	}

	return Read(root, relPath)
}

func resolveFile(root, relPath string) (string, error) {
	rootAbs, err := absRoot(root)
	if err != nil {
		return "", err
	}
	rel := filepath.Clean(filepath.FromSlash(strings.TrimSpace(relPath)))
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrInvalidPath
	}
	abs := filepath.Join(rootAbs, rel)
	abs, err = filepath.Abs(abs)
	if err != nil {
		return "", err
	}
	inside, err := filepath.Rel(rootAbs, abs)
	if err != nil || strings.HasPrefix(inside, "..") {
		return "", ErrInvalidPath
	}
	return abs, nil
}

func absRoot(root string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		root = DefaultDir
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}
