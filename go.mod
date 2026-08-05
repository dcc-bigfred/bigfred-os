module github.com/keskad/bigfred-os

go 1.26

require (
	github.com/coder/websocket v1.8.15
	github.com/creack/pty v1.1.24
	github.com/dcc-bigfred/microinit/go v0.3.0
	github.com/go-chi/chi/v5 v5.3.0
	github.com/go-chi/cors v1.2.2
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/msteinert/pam v1.2.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/alicebob/miniredis/v2 v2.38.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/redis/go-redis/v9 v9.20.1 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
)

// Coordination with the microinit migration PRs: use the local SDK checkout
// so bigfred-os-ui picks up ReadFrame (per-frame read-deadline) and the
// stable error `code` field. Drop this and bump the require version once
// a new microinit/go tag is published.
replace github.com/dcc-bigfred/microinit/go => ../microinit/go

