package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultZoneinfoDir  = "/usr/share/zoneinfo"
	defaultLocaltimePath = "/etc/localtime"
	fakeHwclockName     = "fake-hwclock"
	timezoneName        = "timezone"
	localtimeName       = "localtime"
)

// setSystemClock sets the kernel wall clock (UTC epoch seconds).
// Overridden in tests.
var setSystemClock = func(epoch int64) error {
	out, err := exec.Command("date", "-u", "-s", fmt.Sprintf("@%d", epoch)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("date -s: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// rebindLocaltime refreshes the bind mount of persistent localtime over /etc/localtime.
var rebindLocaltime = func(src, dst string) error {
	_ = exec.Command("umount", dst).Run()
	out, err := exec.Command("mount", "--bind", src, dst).CombinedOutput()
	if err != nil {
		return fmt.Errorf("mount --bind: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

type timeResponse struct {
	Epoch int64  `json:"epoch"`
	ISO   string `json:"iso"`
	UTC   string `json:"utc"`
}

type setTimeRequest struct {
	Epoch *int64  `json:"epoch"`
	ISO   *string `json:"iso"`
}

type timezoneResponse struct {
	Timezone   string   `json:"timezone"`
	Available  []string `json:"available"`
}

type setTimezoneRequest struct {
	Timezone string `json:"timezone"`
}

func zoneinfoDir(cfg Config) string {
	if cfg.ZoneinfoDir != "" {
		return cfg.ZoneinfoDir
	}
	return defaultZoneinfoDir
}

func localtimePath(cfg Config) string {
	if cfg.LocaltimePath != "" {
		return cfg.LocaltimePath
	}
	return defaultLocaltimePath
}

func dataEtc(cfg Config) string {
	if cfg.EtcDir != "" {
		return cfg.EtcDir
	}
	return "/data/etc"
}

func getTimeHandler(_ Config) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		now := time.Now()
		writeJSON(w, http.StatusOK, timeResponse{
			Epoch: now.Unix(),
			ISO:   now.Format(time.RFC3339),
			UTC:   now.UTC().Format(time.RFC3339),
		})
	}
}

func setTimeHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req setTimeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_body")
			return
		}

		var epoch int64
		switch {
		case req.Epoch != nil:
			epoch = *req.Epoch
		case req.ISO != nil && strings.TrimSpace(*req.ISO) != "":
			t, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.ISO))
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid_iso")
				return
			}
			epoch = t.Unix()
		default:
			writeJSONError(w, http.StatusBadRequest, "missing_time")
			return
		}

		// Reject absurd values (before 2000-01-01 or after year 2100).
		if epoch < 946684800 || epoch > 4102444800 {
			writeJSONError(w, http.StatusBadRequest, "time_out_of_range")
			return
		}

		if err := setSystemClock(epoch); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "set_time_failed",
				"message": err.Error(),
			})
			return
		}

		if err := writeFakeHwclock(dataEtc(cfg), epoch); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "persist_failed",
				"message": err.Error(),
			})
			return
		}

		now := time.Unix(epoch, 0)
		writeJSON(w, http.StatusOK, timeResponse{
			Epoch: epoch,
			ISO:   now.Local().Format(time.RFC3339),
			UTC:   now.UTC().Format(time.RFC3339),
		})
	}
}

func getTimezoneHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		tz := readTimezone(dataEtc(cfg))
		available, err := listTimezones(zoneinfoDir(cfg))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "list_failed",
				"message": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, timezoneResponse{
			Timezone:  tz,
			Available: available,
		})
	}
}

func setTimezoneHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req setTimezoneRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_body")
			return
		}
		tz := strings.TrimSpace(req.Timezone)
		if tz == "" {
			writeJSONError(w, http.StatusBadRequest, "missing_timezone")
			return
		}
		if err := validateTimezone(zoneinfoDir(cfg), tz); err != nil {
			writeJSONError(w, http.StatusBadRequest, "unknown_timezone")
			return
		}

		etc := dataEtc(cfg)
		if err := os.MkdirAll(etc, 0o755); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "persist_failed",
				"message": err.Error(),
			})
			return
		}
		if err := os.WriteFile(filepath.Join(etc, timezoneName), []byte(tz+"\n"), 0o644); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "persist_failed",
				"message": err.Error(),
			})
			return
		}

		srcZone := filepath.Join(zoneinfoDir(cfg), filepath.FromSlash(tz))
		dstLocal := filepath.Join(etc, localtimeName)
		data, err := os.ReadFile(srcZone)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "copy_failed",
				"message": err.Error(),
			})
			return
		}
		if err := os.WriteFile(dstLocal, data, 0o644); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "copy_failed",
				"message": err.Error(),
			})
			return
		}

		if err := rebindLocaltime(dstLocal, localtimePath(cfg)); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "bind_failed",
				"message": err.Error(),
			})
			return
		}

		available, _ := listTimezones(zoneinfoDir(cfg))
		writeJSON(w, http.StatusOK, timezoneResponse{
			Timezone:  tz,
			Available: available,
		})
	}
}

func writeFakeHwclock(etcDir string, epoch int64) error {
	if err := os.MkdirAll(etcDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(etcDir, fakeHwclockName), []byte(strconv.FormatInt(epoch, 10)+"\n"), 0o644)
}

func readTimezone(etcDir string) string {
	raw, err := os.ReadFile(filepath.Join(etcDir, timezoneName))
	if err == nil {
		tz := strings.TrimSpace(string(raw))
		if tz != "" {
			return tz
		}
	}
	// Fallback: image default.
	raw, err = os.ReadFile("/etc/timezone")
	if err == nil {
		tz := strings.TrimSpace(string(raw))
		if tz != "" {
			return tz
		}
	}
	return "UTC"
}

func validateTimezone(zoneRoot, tz string) error {
	if tz == "" || strings.Contains(tz, "..") || strings.HasPrefix(tz, "/") {
		return errors.New("invalid")
	}
	abs := filepath.Join(zoneRoot, filepath.FromSlash(tz))
	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		return errors.New("unknown")
	}
	// Ensure resolved path stays under zoneRoot.
	rel, err := filepath.Rel(zoneRoot, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return errors.New("invalid")
	}
	return nil
}

func listTimezones(zoneRoot string) ([]string, error) {
	info, err := os.Stat(zoneRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s: not a directory", zoneRoot)
	}

	skipNames := map[string]struct{}{
		"leapseconds": {}, "iso3166.tab": {}, "zone.tab": {}, "zone1970.tab": {},
		"tzdata.zi": {}, "leap-seconds.list": {},
	}
	var out []string
	err = filepath.WalkDir(zoneRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		rel, err := filepath.Rel(zoneRoot, path)
		if err != nil {
			return nil
		}
		if d.IsDir() {
			base := filepath.Base(path)
			if base == "right" || base == "posix" {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() && d.Type()&fs.ModeSymlink == 0 {
			return nil
		}
		name := filepath.Base(rel)
		if _, skip := skipNames[name]; skip {
			return nil
		}
		slash := filepath.ToSlash(rel)
		// Prefer Area/City style and Etc/*; skip bare country abbreviations.
		if !strings.Contains(slash, "/") && slash != "UTC" && slash != "GMT" {
			return nil
		}
		out = append(out, slash)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	if out == nil {
		out = []string{}
	}
	return out, nil
}
