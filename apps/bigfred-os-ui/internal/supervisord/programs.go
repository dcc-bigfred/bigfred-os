package supervisord

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

const DefaultConfigPath = "/data/etc/supervisord/supervisord.conf"

var (
	ErrInvalidName   = errors.New("invalid program name")
	ErrInvalidAction = errors.New("invalid action")
	ErrNotFound      = errors.New("program not found")
)

var programNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

var supervisorctlBin = "supervisorctl"

// Program describes one supervisord program for the admin UI.
type Program struct {
	Name      string `json:"name"`
	Group     string `json:"group,omitempty"`
	Command   string `json:"command,omitempty"`
	Autostart bool   `json:"autostart"`
	Status    string `json:"status"`
	PID       int    `json:"pid,omitempty"`
}

// List returns programs defined in configPath merged with supervisorctl status.
func List(configPath string) ([]Program, error) {
	defs, err := ParsePrograms(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	statusByName := map[string]ProgramStatus{}
	ctl := Ctl{ConfigPath: configPath, Bin: supervisorctlBin}
	ctx, cancel := execTimeout(10 * time.Second)
	defer cancel()
	if rows, err := ctl.Status(ctx); err == nil {
		for _, row := range rows {
			statusByName[row.Name] = row
		}
	}

	out := make([]Program, 0, len(defs))
	for _, def := range defs {
		prog := Program{
			Name:      def.Name,
			Group:     def.Group,
			Command:   def.Command,
			Autostart: def.Autostart,
			Status:    "STOPPED",
		}
		if st, ok := statusByName[ctlName(def)]; ok {
			prog.Status = st.Status
			prog.PID = st.PID
		}
		out = append(out, prog)
	}
	return out, nil
}

// ctlName returns the name supervisorctl uses for a program. Programs that
// belong to a [group:…] are addressed as "group:program"; supervisord also
// reports them that way in `supervisorctl status`.
func ctlName(def programDef) string {
	if def.Group != "" && def.Group != def.Name {
		return def.Group + ":" + def.Name
	}
	return def.Name
}

// Control runs supervisorctl start|stop|restart for a configured program.
func Control(configPath, name, action string) error {
	if err := validateName(name); err != nil {
		return err
	}
	switch action {
	case "start", "stop", "restart":
	default:
		return ErrInvalidAction
	}

	defs, err := ParsePrograms(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return err
	}
	def, ok := findProgram(defs, name)
	if !ok {
		return ErrNotFound
	}
	target := ctlName(def)

	ctl := Ctl{ConfigPath: configPath, Bin: supervisorctlBin}
	ctx, cancel := execTimeout(30 * time.Second)
	defer cancel()

	var ctlErr error
	switch action {
	case "start":
		ctlErr = ctl.StartProgram(ctx, target)
	case "stop":
		ctlErr = ctl.StopProgram(ctx, target)
	case "restart":
		ctlErr = ctl.RestartProgram(ctx, target)
	}
	if ctlErr != nil {
		return fmt.Errorf("%s", ctlErr.Error())
	}
	return nil
}

func validateName(name string) error {
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "..") {
		return ErrInvalidName
	}
	if !programNamePattern.MatchString(name) {
		return ErrInvalidName
	}
	return nil
}

func findProgram(defs []programDef, name string) (programDef, bool) {
	for _, d := range defs {
		if d.Name == name {
			return d, true
		}
	}
	return programDef{}, false
}

var execTimeout = func(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}
