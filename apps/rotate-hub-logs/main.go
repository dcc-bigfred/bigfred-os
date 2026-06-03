// rotate-hub-logs rotates and prunes hub log files under /data/logs (§8.9).
package main

import (
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	defaultLogRoot       = "/data/logs"
	defaultRetentionDays = 14
	defaultMaxBytes      = 512 * 1024 * 1024
	defaultRotateSize    = 10 * 1024 * 1024
)

type config struct {
	logRoot       string
	retentionDays int
	maxBytes      int64
	rotateSize    int64
}

func main() {
	cfg := config{
		logRoot:       defaultLogRoot,
		retentionDays: defaultRetentionDays,
		maxBytes:      defaultMaxBytes,
		rotateSize:    defaultRotateSize,
	}
	flag.StringVar(&cfg.logRoot, "logroot", cfg.logRoot, "root directory for service logs")
	flag.IntVar(&cfg.retentionDays, "retention-days", cfg.retentionDays, "delete .gz files older than this many days")
	flag.Int64Var(&cfg.maxBytes, "max-bytes", cfg.maxBytes, "maximum total size of all .gz under logroot")
	flag.Int64Var(&cfg.rotateSize, "rotate-size", cfg.rotateSize, "rotate .log files when larger than this many bytes")
	flag.Parse()

	if err := run(cfg); err != nil {
		log.Fatal(err)
	}
}

func run(cfg config) error {
	info, err := os.Stat(cfg.logRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s: not a directory", cfg.logRoot)
	}

	cutoff := time.Now().AddDate(0, 0, -cfg.retentionDays)

	entries, err := os.ReadDir(cfg.logRoot)
	if err != nil {
		return err
	}

	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		dir := filepath.Join(cfg.logRoot, ent.Name())
		if err := rotateLogsInDir(dir, cfg.rotateSize); err != nil {
			return err
		}
		if err := deleteExpiredGzip(dir, cutoff); err != nil {
			return err
		}
	}

	return enforceMaxGzipTotal(cfg.logRoot, cfg.maxBytes)
}

func rotateLogsInDir(dir string, rotateSize int64) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".log") {
			continue
		}
		path := filepath.Join(dir, ent.Name())
		info, err := ent.Info()
		if err != nil {
			return err
		}
		if info.Size() <= rotateSize {
			continue
		}
		if err := rotateFile(path); err != nil {
			return fmt.Errorf("rotate %s: %w", path, err)
		}
	}
	return nil
}

func rotateFile(path string) error {
	ts := time.Now().Format("20060102150405")
	archive := path + "." + ts

	if err := copyFile(path, archive); err != nil {
		return err
	}
	if err := os.Truncate(path, 0); err != nil {
		return err
	}
	return gzipFile(archive)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func gzipFile(path string) error {
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(path+".gz", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	gw, err := gzip.NewWriterLevel(out, gzip.BestCompression)
	if err != nil {
		return err
	}
	if _, err := io.Copy(gw, in); err != nil {
		_ = gw.Close()
		return err
	}
	if err := gw.Close(); err != nil {
		return err
	}
	return os.Remove(path)
}

func deleteExpiredGzip(dir string, cutoff time.Time) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".gz") {
			continue
		}
		info, err := ent.Info()
		if err != nil {
			return err
		}
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(filepath.Join(dir, ent.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

type gzipFileInfo struct {
	path string
	mod  time.Time
	size int64
}

func collectGzipFiles(logRoot string) ([]gzipFileInfo, int64, error) {
	var files []gzipFileInfo
	var total int64

	err := filepath.WalkDir(logRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".gz") {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		files = append(files, gzipFileInfo{
			path: path,
			mod:  info.ModTime(),
			size: info.Size(),
		})
		total += info.Size()
		return nil
	})
	return files, total, err
}

func enforceMaxGzipTotal(logRoot string, maxBytes int64) error {
	files, total, err := collectGzipFiles(logRoot)
	if err != nil {
		return err
	}
	if total <= maxBytes {
		return nil
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].mod.Before(files[j].mod)
	})

	for _, f := range files {
		if total <= maxBytes {
			break
		}
		if err := os.Remove(f.path); err != nil {
			return err
		}
		total -= f.size
	}
	return nil
}
