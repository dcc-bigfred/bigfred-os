package services

import (
	"errors"
	"testing"

	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/microinit"
)

type fakeClient struct {
	list []microinit.ServiceStatus
	err  error
}

func (f *fakeClient) List() ([]microinit.ServiceStatus, error) {
	return f.list, f.err
}

func (f *fakeClient) Control(name, action string) error {
	if action == "pause" {
		return ErrInvalidAction
	}
	if name == "missing" {
		return ErrNotFound
	}
	return nil
}

func TestListMapsMicroinitStatus(t *testing.T) {
	pid := int32(9)
	list, err := List(&fakeClient{list: []microinit.ServiceStatus{{
		Name: "bigfred-os-ui", State: "running", PID: &pid, Restarts: 2, Enabled: true,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("len=%d", len(list))
	}
	s := list[0]
	if s.ID != "bigfred-os-ui" || s.Name != "bigfred os ui" || !s.Running || s.Restarts != 2 {
		t.Fatalf("%+v", s)
	}
}

func TestControlRejectsInvalidAction(t *testing.T) {
	if err := Control(&fakeClient{}, "demo", "pause"); !errors.Is(err, ErrInvalidAction) {
		t.Fatalf("got %v", err)
	}
}
