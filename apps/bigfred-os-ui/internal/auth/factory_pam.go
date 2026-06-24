//go:build pam

package auth

// New creates a PAM-backed session service.
func New(cfg Config) (*Service, error) {
	service := cfg.PAMService
	if service == "" {
		service = defaultPAMService
	}
	return newService(newPAMChecker(service), cfg.TTL)
}
