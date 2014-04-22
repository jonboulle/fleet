package unit

import (
	"fmt"
	"strings"
)

// Description returns the first Description option found in the [Unit] section.
// If the option is not defined, an empty string is returned.
func (self *Unit) Description() string {
	if values := self.Contents["Unit"]["Description"]; len(values) > 0 {
		return values[0]
	}
	return ""
}

func NewUnit(raw string) *Unit {
	parsed := deserializeUnitFile(raw)
	return &Unit{parsed, raw}
}

// NewUnitFromLegacyContents creates a Unit object from an obsolete unit
// file datastructure. This should only be used to remain backwards-compatible where necessary.
func NewUnitFromLegacyContents(contents map[string]map[string]string) *Unit {
	var serialized string
	for section, keyMap := range contents {
		serialized += fmt.Sprintf("[%s]\n", section)
		for key, value := range keyMap {
			serialized += fmt.Sprintf("%s=%s\n", key, value)
		}
		serialized += "\n"
	}
	return NewUnit(serialized)
}

// deserializeUnitFile parses a systemd unit file and attempts to map its various sections and values.
// Currently this function is dangerously simple and should be rewritten to match the systemd unit file spec
func deserializeUnitFile(raw string) map[string]map[string][]string {
	sections := make(map[string]map[string][]string)
	var section string
	for _, line := range strings.Split(raw, "\n") {
		// Ignore commented-out lines
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		line = strings.Trim(line, " ")

		// Ignore blank lines
		if len(line) == 0 {
			continue
		}

		// Check for section
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = line[1 : len(line)-1]
			sections[section] = make(map[string][]string)
			continue
		}

		// Check for key=value
		if strings.ContainsAny(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			key := strings.Trim(parts[0], " ")
			value := strings.Trim(parts[1], " ")

			if len(section) > 0 {
				sections[section][key] = append(sections[section][key], value)
			}

		}
	}

	return sections
}
