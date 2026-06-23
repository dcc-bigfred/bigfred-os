package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"gopkg.in/yaml.v3"
)

const (
	defaultAutostart = true
	defaultRetries   = 0
)

var scriptNameRe = regexp.MustCompile(`^S[0-9]{2}-(.+)$`)

type Config struct {
	Services []ServiceConfig `yaml:"services"`
}

type ServiceConfig struct {
	Name      string `yaml:"name"`
	Autostart bool   `yaml:"autostart"`
	Retries   int    `yaml:"retries"`
}

type serviceConfigRaw struct {
	Name      string `yaml:"name"`
	Autostart *bool  `yaml:"autostart"`
	Retries   *int   `yaml:"retries"`
}

type configRaw struct {
	Services []serviceConfigRaw `yaml:"services"`
}

type initScript struct {
	id    string
	path  string
	shell bool
}

func discoverScripts(initDir string) ([]initScript, error) {
	entries, err := os.ReadDir(initDir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", initDir, err)
	}

	var out []initScript
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		name := ent.Name()
		if name == "rcS" {
			continue
		}
		m := scriptNameRe.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		info, err := ent.Info()
		if err != nil {
			continue
		}
		if info.Mode()&0o111 == 0 {
			continue
		}

		path := filepath.Join(initDir, name)
		out = append(out, initScript{
			id:    m[1],
			path:  path,
			shell: filepath.Ext(name) == ".sh",
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].path < out[j].path
	})
	return out, nil
}

func defaultConfig(scripts []initScript) Config {
	services := make([]ServiceConfig, 0, len(scripts))
	for _, script := range scripts {
		services = append(services, ServiceConfig{
			Name:      script.id,
			Autostart: defaultAutostart,
			Retries:   defaultRetries,
		})
	}
	return Config{Services: services}
}

func mergeConfig(defaults, user Config) Config {
	byName := make(map[string]ServiceConfig, len(defaults.Services))
	for _, svc := range defaults.Services {
		byName[svc.Name] = svc
	}
	for _, svc := range user.Services {
		if svc.Name == "" {
			continue
		}
		base, ok := byName[svc.Name]
		if !ok {
			base = ServiceConfig{
				Name:      svc.Name,
				Autostart: defaultAutostart,
				Retries:   defaultRetries,
			}
		}
		base.Name = svc.Name
		base.Autostart = svc.Autostart
		base.Retries = svc.Retries
		byName[svc.Name] = base
	}

	out := make([]ServiceConfig, 0, len(byName))
	for _, svc := range byName {
		out = append(out, svc)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return Config{Services: out}
}

func (c Config) lookup(name string) (ServiceConfig, bool) {
	for _, svc := range c.Services {
		if svc.Name == name {
			return svc, true
		}
	}
	return ServiceConfig{}, false
}

func readConfigFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read %s: %w", path, err)
	}

	var raw configRaw
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}

	cfg := Config{Services: make([]ServiceConfig, 0, len(raw.Services))}
	for _, svc := range raw.Services {
		if svc.Name == "" {
			continue
		}
		entry := ServiceConfig{
			Name:      svc.Name,
			Autostart: defaultAutostart,
			Retries:   defaultRetries,
		}
		if svc.Autostart != nil {
			entry.Autostart = *svc.Autostart
		}
		if svc.Retries != nil {
			entry.Retries = *svc.Retries
		}
		cfg.Services = append(cfg.Services, entry)
	}
	return cfg, nil
}

func writeConfigFile(path string, cfg Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	header := []byte("# biginit service configuration\n\n")
	content := append(header, data...)
	return os.WriteFile(path, content, 0o644)
}
