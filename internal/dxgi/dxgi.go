package dxgi

import (
	"errors"
	"strings"
)

var (
	ErrOutputNotFound  = errors.New("DXGI output not found")
	ErrOutputAmbiguous = errors.New("multiple DXGI outputs matched the display")
)

type Vendor int

const (
	VendorUnknown Vendor = iota
	VendorNVIDIA
	VendorAMD
	VendorIntel
)

type Output struct {
	AdapterIndex int
	OutputIndex  int
	DeviceName   string
	Vendor       Vendor
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

func (v Vendor) String() string {
	switch v {
	case VendorNVIDIA:
		return "NVIDIA"
	case VendorAMD:
		return "AMD"
	case VendorIntel:
		return "Intel"
	default:
		return "Unknown"
	}
}
