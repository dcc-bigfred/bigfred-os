package supervisord

import (
	"bufio"
	"os"
	"regexp"
	"sort"
	"strings"
)

var sectionRe = regexp.MustCompile(`^\[([^:\]]+):([^\]]+)\]$`)

// programDef is one [program:…] entry from supervisord.conf.
type programDef struct {
	Name      string
	Group     string
	Command   string
	Autostart bool
}

// ParsePrograms reads program definitions from a supervisord INI file.
func ParsePrograms(configPath string) ([]programDef, error) {
	f, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	groupPrograms := map[string]string{} // program name → group
	programs := map[string]programDef{}

	var sectionKind, sectionName string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if m := sectionRe.FindStringSubmatch(line); m != nil {
			sectionKind = strings.ToLower(m[1])
			sectionName = m[2]
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch sectionKind {
		case "group":
			if key == "programs" {
				for _, name := range strings.Split(value, ",") {
					name = strings.TrimSpace(name)
					if name != "" {
						groupPrograms[name] = sectionName
					}
				}
			}
		case "program":
			prog := programs[sectionName]
			prog.Name = sectionName
			switch key {
			case "command":
				prog.Command = value
			case "autostart":
				prog.Autostart = strings.EqualFold(value, "true")
			}
			programs[sectionName] = prog
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	out := make([]programDef, 0, len(programs))
	for name, prog := range programs {
		prog.Group = groupPrograms[name]
		out = append(out, prog)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out, nil
}
