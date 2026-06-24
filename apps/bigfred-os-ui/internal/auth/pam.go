//go:build pam

package auth

import (
	"fmt"
	"strings"

	"github.com/msteinert/pam"
)

const defaultPAMService = "bigfred-os-ui"

type pamChecker struct {
	service string
}

func newPAMChecker(service string) Checker {
	if strings.TrimSpace(service) == "" {
		service = defaultPAMService
	}
	return &pamChecker{service: service}
}

func (p *pamChecker) Authenticate(username, password string) error {
	t, err := pam.StartFunc(p.service, username, func(style pam.Style, _ string) (string, error) {
		if style == pam.PromptEchoOff {
			return password, nil
		}
		return "", nil
	})
	if err != nil {
		return ErrInvalidCredentials
	}
	if err := t.Authenticate(0); err != nil {
		return ErrInvalidCredentials
	}
	return nil
}

func (p *pamChecker) ChangePassword(username, current, newPassword string) error {
	newPrompts := 0
	t, err := pam.StartFunc(p.service, username, func(style pam.Style, msg string) (string, error) {
		if style != pam.PromptEchoOff {
			return "", nil
		}
		lower := strings.ToLower(msg)
		switch {
		case strings.Contains(lower, "current"):
			return current, nil
		case strings.Contains(lower, "retype"),
			strings.Contains(lower, "repeat"),
			strings.Contains(lower, "again"),
			strings.Contains(lower, "confirm"):
			return newPassword, nil
		case strings.Contains(lower, "new"):
			newPrompts++
			return newPassword, nil
		default:
			if newPrompts == 0 {
				return current, nil
			}
			return newPassword, nil
		}
	})
	if err != nil {
		return fmt.Errorf("%w: %w", ErrPasswordChangeFailed, err)
	}
	if err := t.Authenticate(0); err != nil {
		return ErrInvalidCredentials
	}
	if err := t.ChangeAuthTok(0); err != nil {
		return fmt.Errorf("%w: %w", ErrPasswordChangeFailed, err)
	}
	return nil
}
