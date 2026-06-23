// biginit runs SysV boot scripts from /etc/init.d (BusyBox rcS replacement).
package main

import (
	"fmt"
	"os"
)

const (
	defaultInitDir     = "/etc/init.d"
	defaultConfigPath  = "/data/etc/biginit.yaml"
	defaultDefaultsPath = "/data/etc/biginit.yaml.defaults"
)

func main() {
	initDir := envOr("BIGINIT_INIT_DIR", defaultInitDir)
	configPath := envOr("BIGINIT_CONFIG", defaultConfigPath)
	defaultsPath := envOr("BIGINIT_DEFAULTS", defaultDefaultsPath)

	if err := runBoot(initDir, configPath, defaultsPath); err != nil {
		fmt.Fprintf(os.Stderr, "biginit: %v\n", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func runBoot(initDir, configPath, defaultsPath string) error {
	scripts, err := discoverScripts(initDir)
	if err != nil {
		return err
	}

	r := &runner{
		initDir:      initDir,
		configPath:   configPath,
		defaultsPath: defaultsPath,
		scripts:      scripts,
	}

	for _, script := range scripts {
		if script.id == "mount" {
			if err := r.startScript(script); err != nil {
				fmt.Fprintf(os.Stderr, "biginit: %s: %v\n", script.path, err)
			}
			if err := r.loadOrCreateConfig(); err != nil {
				fmt.Fprintf(os.Stderr, "biginit: config: %v\n", err)
			}
			continue
		}

		cfg := r.serviceConfig(script.id)
		if !cfg.Autostart {
			continue
		}

		var lastErr error
		attempts := cfg.Retries + 1
		for attempt := 1; attempt <= attempts; attempt++ {
			lastErr = r.startScript(script)
			if lastErr == nil {
				break
			}
			if attempt < attempts {
				fmt.Fprintf(os.Stderr, "biginit: %s: attempt %d/%d failed: %v\n",
					script.id, attempt, attempts, lastErr)
			}
		}
		if lastErr != nil {
			fmt.Fprintf(os.Stderr, "biginit: %s: failed after %d attempt(s): %v\n",
				script.id, attempts, lastErr)
		}
	}

	return nil
}

type runner struct {
	initDir      string
	configPath   string
	defaultsPath string
	scripts      []initScript
	config       Config
}

func (r *runner) loadOrCreateConfig() error {
	defaults := defaultConfig(r.scripts)
	if err := writeConfigFile(r.defaultsPath, defaults); err != nil {
		return fmt.Errorf("write defaults: %w", err)
	}

	if _, err := os.Stat(r.configPath); os.IsNotExist(err) {
		if err := writeConfigFile(r.configPath, defaults); err != nil {
			return fmt.Errorf("create config: %w", err)
		}
	}

	cfg, err := readConfigFile(r.configPath)
	if err != nil {
		return err
	}
	r.config = mergeConfig(defaults, cfg)
	return nil
}

func (r *runner) serviceConfig(name string) ServiceConfig {
	if cfg, ok := r.config.lookup(name); ok {
		return cfg
	}
	return ServiceConfig{
		Name:      name,
		Autostart: true,
		Retries:   0,
	}
}

func (r *runner) startScript(script initScript) error {
	if script.shell {
		return startShellScript(script.path)
	}
	return startInitScript(script.path)
}
