package logs

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Entry describes one log file exposed in the UI.
type Entry struct {
	ID      string `json:"id"`
	Root    string `json:"root"`
	Service string `json:"service"`
	Name    string `json:"name"`
	Size    int64  `json:"size"`
}

// ListAll returns log files from every configured root directory.
func ListAll(roots []string) ([]Entry, error) {
	seen := make(map[string]struct{})
	var out []Entry
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		entries, err := listRoot(root)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if _, ok := seen[e.ID]; ok {
				continue
			}
			seen[e.ID] = struct{}{}
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Root != out[j].Root {
			return out[i].Root < out[j].Root
		}
		if out[i].Service != out[j].Service {
			return out[i].Service < out[j].Service
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func listRoot(root string) ([]Entry, error) {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s: not a directory", root)
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	rootID := rootSlug(rootAbs)

	var out []Entry
	err = filepath.WalkDir(rootAbs, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if path == rootAbs {
				return nil
			}
			rel, err := filepath.Rel(rootAbs, path)
			if err != nil {
				return nil
			}
			if strings.Count(rel, string(filepath.Separator)) >= 2 {
				return filepath.SkipDir
			}
			return nil
		}

		if !isLogFile(d.Name()) {
			return nil
		}

		rel, err := filepath.Rel(rootAbs, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil
		}
		if strings.Count(rel, string(filepath.Separator)) > 2 {
			return nil
		}

		fi, err := d.Info()
		if err != nil {
			return nil
		}

		service := filepath.ToSlash(filepath.Dir(rel))
		if service == "." {
			service = filepath.Base(rootAbs)
		}
		out = append(out, Entry{
			ID:      rootID + ":" + filepath.ToSlash(rel),
			Root:    rootAbs,
			Service: service,
			Name:    d.Name(),
			Size:    fi.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func rootSlug(abs string) string {
	switch filepath.Clean(abs) {
	case "/data/logs":
		return "data"
	case "/var/log":
		return "var"
	default:
		s := strings.Trim(abs, "/")
		s = strings.ReplaceAll(s, "/", "_")
		if s == "" {
			return "log"
		}
		return s
	}
}

func isLogFile(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".gz") || strings.HasSuffix(lower, ".xz") ||
		strings.HasSuffix(lower, ".bz2") || strings.HasSuffix(lower, ".zip") {
		return false
	}
	if strings.HasSuffix(lower, ".log") {
		return true
	}
	switch lower {
	case "messages", "syslog", "auth.log", "kern.log", "daemon.log", "user.log", "cron":
		return true
	}
	return false
}

// ResolvePath maps a log id (rootSlug:relative/path) to an absolute path.
func ResolvePath(roots []string, id string) (string, error) {
	rootID, rel, ok := strings.Cut(id, ":")
	if !ok {
		return "", fmt.Errorf("invalid path")
	}
	rel = filepath.Clean(filepath.FromSlash(rel))
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid path")
	}
	if !isLogFile(filepath.Base(rel)) {
		return "", fmt.Errorf("not a log file")
	}

	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		rootAbs, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		if rootSlug(rootAbs) != rootID {
			continue
		}
		abs := filepath.Join(rootAbs, rel)
		abs, err = filepath.Abs(abs)
		if err != nil {
			return "", err
		}
		inside, err := filepath.Rel(rootAbs, abs)
		if err != nil || strings.HasPrefix(inside, "..") {
			return "", fmt.Errorf("invalid path")
		}
		return abs, nil
	}
	return "", fmt.Errorf("invalid path")
}

// TailLast reads up to maxLines from the end of a file.
func TailLast(path string, maxLines int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	lines, err := readLastLines(f, maxLines)
	if err != nil {
		return nil, err
	}
	return lines, nil
}

func readLastLines(r io.Reader, maxLines int) ([]string, error) {
	sc := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)

	var ring []string
	for sc.Scan() {
		line := sc.Text()
		if len(ring) < maxLines {
			ring = append(ring, line)
			continue
		}
		copy(ring, ring[1:])
		ring[maxLines-1] = line
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return ring, nil
}

// Tailer follows a file from a byte offset, emitting new lines.
type Tailer struct {
	path   string
	offset int64
}

func NewTailer(path string, startOffset int64) *Tailer {
	return &Tailer{path: path, offset: startOffset}
}

func (t *Tailer) ReadNew() ([]string, int64, error) {
	f, err := os.Open(t.path)
	if err != nil {
		return nil, t.offset, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, t.offset, err
	}
	size := info.Size()
	if size < t.offset {
		t.offset = 0
	}
	if _, err := f.Seek(t.offset, io.SeekStart); err != nil {
		return nil, t.offset, err
	}

	var lines []string
	reader := bufio.NewReader(f)
	for {
		part, err := reader.ReadString('\n')
		if len(part) > 0 {
			lines = append(lines, strings.TrimRight(part, "\r\n"))
		}
		if err == io.EOF {
			t.offset += int64(len(part))
			break
		}
		if err != nil {
			return lines, t.offset, err
		}
		t.offset += int64(len(part))
	}
	return lines, t.offset, nil
}

func FileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// ParseRoots splits a comma-separated root list and applies defaults.
func ParseRoots(logRoots, legacyLogRoot string) []string {
	raw := strings.TrimSpace(logRoots)
	if raw == "" {
		raw = strings.TrimSpace(legacyLogRoot)
	}
	if raw == "" {
		return []string{"/data/logs", "/var/log"}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"/data/logs", "/var/log"}
	}
	return out
}
