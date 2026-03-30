package dxgi

import (
	"errors"
	"strings"
)

var (
	ErrOutputNotFound  = errors.New("DXGI output not found")
	ErrOutputAmbiguous = errors.New("multiple DXGI outputs matched the display")
)

type Output struct {
	AdapterIndex int
	OutputIndex  int
	DeviceName   string
}

func MatchOutputByDeviceName(outputs []Output, deviceName string) (Output, error) {
	var matches []Output
	for _, output := range outputs {
		if strings.EqualFold(output.DeviceName, deviceName) {
			matches = append(matches, output)
		}
	}

	switch len(matches) {
	case 0:
		return Output{}, ErrOutputNotFound
	case 1:
		return matches[0], nil
	default:
		return Output{}, ErrOutputAmbiguous
	}
}
