// Package services exposes hub services via the microinit control socket.
package services

import (
	"errors"
	"strings"

	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/microinit"
)

const DefaultSocket = microinit.DefaultSocket

var (
	ErrInvalidID     = microinit.ErrInvalidName
	ErrInvalidAction = microinit.ErrInvalidAction
	ErrNotFound      = microinit.ErrNotFound
)

// Service is one microinit-managed service for the admin API.
type Service struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	State    string `json:"state"`
	PID      *int32 `json:"pid,omitempty"`
	Restarts uint32 `json:"restarts"`
	Enabled  bool   `json:"enabled"`
	Running  bool   `json:"running"`
}

// Client abstracts microinit for tests.
type Client interface {
	List() ([]microinit.ServiceStatus, error)
	Control(name, action string) error
}

// List returns services from microinit.
func List(client Client) ([]Service, error) {
	if client == nil {
		return nil, errors.New("microinit client is nil")
	}
	raw, err := client.List()
	if err != nil {
		return nil, err
	}
	out := make([]Service, 0, len(raw))
	for _, s := range raw {
		out = append(out, fromStatus(s))
	}
	return out, nil
}

// Control runs start|stop|restart via microinit.
func Control(client Client, id, action string) error {
	if client == nil {
		return errors.New("microinit client is nil")
	}
	return client.Control(id, action)
}

func fromStatus(s microinit.ServiceStatus) Service {
	state := s.State
	if state == "" {
		state = "unknown"
	}
	return Service{
		ID:       s.Name,
		Name:     displayName(s.Name),
		State:    state,
		PID:      s.PID,
		Restarts: s.Restarts,
		Enabled:  s.Enabled,
		Running:  isRunningState(state),
	}
}

func isRunningState(state string) bool {
	switch strings.ToLower(state) {
	case "running", "starting", "restarting", "waiting_for_dependency":
		return true
	default:
		return false
	}
}

func displayName(id string) string {
	id = strings.TrimSuffix(id, ".example")
	return strings.ReplaceAll(id, "-", " ")
}
