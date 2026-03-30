package display

import (
	"errors"
	"strings"
)

const ParsecDisplayCode = "PSCCDD0"

var (
	ErrNoNewParsecDisplay        = errors.New("no new Parsec display found")
	ErrAmbiguousNewParsecDisplay = errors.New("multiple new Parsec displays found")
)

type Snapshot struct {
	InterfaceName string
	DeviceName    string
	DisplayCode   string
}

func FindNewParsecDisplay(before []Snapshot, after []Snapshot, displayCode string) (Snapshot, error) {
	known := make(map[string]struct{}, len(before))
	for _, snapshot := range before {
		known[strings.ToUpper(snapshot.InterfaceName)] = struct{}{}
	}

	var matches []Snapshot
	for _, snapshot := range after {
		if !strings.EqualFold(snapshot.DisplayCode, displayCode) {
			continue
		}

		if _, exists := known[strings.ToUpper(snapshot.InterfaceName)]; exists {
			continue
		}

		matches = append(matches, snapshot)
	}

	switch len(matches) {
	case 0:
		return Snapshot{}, ErrNoNewParsecDisplay
	case 1:
		return matches[0], nil
	default:
		return Snapshot{}, ErrAmbiguousNewParsecDisplay
	}
}

func ParseDisplayCode(deviceID string) string {
	parts := strings.Split(deviceID, "#")
	if len(parts) >= 2 {
		return parts[1]
	}

	return deviceID
}
