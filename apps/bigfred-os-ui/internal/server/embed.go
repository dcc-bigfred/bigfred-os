package server

import (
	"io/fs"
)

// StaticSub returns the embedded web/dist subtree when present.
func StaticSub(root fs.FS, dir string) (fs.FS, error) {
	sub, err := fs.Sub(root, dir)
	if err != nil {
		return nil, err
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil, err
	}
	return sub, nil
}
