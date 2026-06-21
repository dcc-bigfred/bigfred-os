package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const DefaultPath = "/data/etc/bigfred-os-ui.conf"

// File holds settings from a dotenv-style configuration file.
type File struct {
	HTTP         string
	Username     string
	Password     string
	LogRoot         string
	LogRoots        string
	InitDir         string
	SupervisordConf string
	RedisAddr       string
	EtcDir          string
	SecureCookie    *bool
}

// LoadOptional reads path when it exists. Missing file returns (nil, nil).
func LoadOptional(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return Parse(string(data)), nil
}

// Parse reads KEY=value lines (dotenv). Comments (#) and blank lines are ignored.
func Parse(text string) *File {
	f := &File{}
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if key == "" {
			continue
		}
		switch strings.ToUpper(key) {
		case "HTTP", "HTTP_ADDR", "LISTEN":
			f.HTTP = value
		case "USERNAME", "USER", "LOGIN":
			f.Username = value
		case "PASSWORD", "PASS":
			f.Password = value
		case "LOG_ROOT", "LOGROOT":
			f.LogRoot = value
		case "LOG_ROOTS", "LOGROOTS":
			f.LogRoots = value
		case "INIT_DIR", "INITDIR":
			f.InitDir = value
		case "SUPERVISORD_CONF", "SUPERVISORDCONF":
			f.SupervisordConf = value
		case "REDIS_ADDR", "REDISADDR":
			f.RedisAddr = value
		case "ETC_DIR", "ETCDIR":
			f.EtcDir = value
		case "SECURE_COOKIE", "SECURECOOKIE":
			v := parseBool(value)
			f.SecureCookie = &v
		}
	}
	return f
}

func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
