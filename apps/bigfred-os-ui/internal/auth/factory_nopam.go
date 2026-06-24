//go:build !pam

package auth

import "fmt"

// New creates a session service. Without the pam build tag only static credentials work.
func New(cfg Config) (*Service, error) {
	if cfg.Username != "" && cfg.Password != "" {
		return newService(NewStaticChecker(cfg.Username, cfg.Password), cfg.TTL)
	}
	if cfg.PAMService != "" {
		return nil, fmt.Errorf("PAM authentication requires building with -tags pam")
	}
	return nil, fmt.Errorf("username and password are required without PAM")
}
