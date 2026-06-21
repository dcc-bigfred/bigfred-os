// Package main is the BigFred hub OS admin UI (HTTP + WebSocket + embedded React).
package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/auth"
	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/redis"
	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/config"
	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/etcdir"
	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/logs"
	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/server"
	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/services"
	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/supervisord"
)

//go:embed all:web/dist
var embeddedWeb embed.FS

func main() {
	os.Exit(run())
}

func run() int {
	var (
		configPath   string
		httpAddr     string
		username     string
		password     string
		logRoots     string
		legacyLogRoot string
		secureCookie bool
		staticDir    string
		initDir          string
		supervisordConf  string
		redisAddr        string
		etcDir           string
	)

	flag.StringVar(&configPath, "config", config.DefaultPath,
		"dotenv configuration file (KEY=value)")
	flag.StringVar(&httpAddr, "http", "0.0.0.0:8090", "HTTP listen address")
	flag.StringVar(&username, "username", "", "login username (required)")
	flag.StringVar(&password, "password", "", "login password (required)")
	flag.StringVar(&logRoots, "log-roots", "", "comma-separated log directories (default: /data/logs,/var/log)")
	flag.StringVar(&legacyLogRoot, "log-root", "", "deprecated: single log directory (use --log-roots)")
	flag.BoolVar(&secureCookie, "secure-cookie", false, "set Secure flag on session cookie")
	flag.StringVar(&staticDir, "static-dir", "", "serve frontend from disk instead of embedded bundle (dev)")
	flag.StringVar(&initDir, "init-dir", services.DefaultInitDir, "SysV init scripts directory")
	flag.StringVar(&supervisordConf, "supervisord-conf", supervisord.DefaultConfigPath, "supervisord configuration file")
	flag.StringVar(&redisAddr, "redis-addr", redis.DefaultAddr, "Redis server address")
	flag.StringVar(&etcDir, "etc-dir", etcdir.DefaultDir, "editable configuration directory")
	flag.Parse()

	if err := mergeConfigFile(configPath, &httpAddr, &username, &password, &logRoots, &legacyLogRoot, &secureCookie, &initDir, &supervisordConf, &redisAddr, &etcDir); err != nil {
		fmt.Fprintf(os.Stderr, "bigfred-os-ui: %v\n", err)
		return 1
	}

	authSvc, err := auth.New(username, password, 24*time.Hour)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bigfred-os-ui: %v\n", err)
		return 1
	}

	staticFS, err := loadStatic(staticDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bigfred-os-ui: static files: %v\n", err)
		return 1
	}

	handler := server.NewRouter(server.Config{
		Auth:            authSvc,
		LogRoots:        logs.ParseRoots(logRoots, legacyLogRoot),
		InitDir:         initDir,
		SupervisordConf: supervisordConf,
		RedisAddr:       redisAddr,
		EtcDir:          etcDir,
		StaticFS:        staticFS,
		SecureCookie:    secureCookie,
		DevOrigins: []string{
			"http://localhost:5174",
			"http://127.0.0.1:5174",
		},
	})

	srv := &http.Server{
		Addr:              httpAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("bigfred-os-ui listening on %s", httpAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server error: %v", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	return 0
}

func mergeConfigFile(path string, httpAddr, username, password, logRoots, legacyLogRoot *string, secureCookie *bool, initDir, supervisordConf, redisAddr, etcDir *string) error {
	fc, err := config.LoadOptional(path)
	if err != nil {
		return err
	}
	if fc == nil {
		return nil
	}
	if !flagPassed("http") && fc.HTTP != "" {
		*httpAddr = fc.HTTP
	}
	if !flagPassed("username") && fc.Username != "" {
		*username = fc.Username
	}
	if !flagPassed("password") && fc.Password != "" {
		*password = fc.Password
	}
	if !flagPassed("log-roots") && fc.LogRoots != "" {
		*logRoots = fc.LogRoots
	}
	if !flagPassed("log-root") && fc.LogRoot != "" && *logRoots == "" {
		*legacyLogRoot = fc.LogRoot
	}
	if !flagPassed("secure-cookie") && fc.SecureCookie != nil {
		*secureCookie = *fc.SecureCookie
	}
	if !flagPassed("init-dir") && fc.InitDir != "" {
		*initDir = fc.InitDir
	}
	if !flagPassed("supervisord-conf") && fc.SupervisordConf != "" {
		*supervisordConf = fc.SupervisordConf
	}
	if !flagPassed("redis-addr") && fc.RedisAddr != "" {
		*redisAddr = fc.RedisAddr
	}
	if !flagPassed("etc-dir") && fc.EtcDir != "" {
		*etcDir = fc.EtcDir
	}
	return nil
}

func flagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func loadStatic(dir string) (fs.FS, error) {
	if dir != "" {
		return os.DirFS(dir), nil
	}
	return server.StaticSub(embeddedWeb, "web/dist")
}
