package dxgi

import (
	"errors"
	"testing"
)

func TestMatchOutputByDeviceName(t *testing.T) {
	outputs := []Output{
		{AdapterIndex: 0, OutputIndex: 0, DeviceName: `\\.\DISPLAY1`},
		{AdapterIndex: 1, OutputIndex: 0, DeviceName: `\\.\DISPLAY2`},
	}

	got, err := MatchOutputByDeviceName(outputs, `\\.\display2`)
	if err != nil {
		t.Fatalf("MatchOutputByDeviceName() error = %v", err)
	}

	if got.AdapterIndex != 1 || got.OutputIndex != 0 {
		t.Fatalf("MatchOutputByDeviceName() = %+v, want adapter 1 output 0", got)
	}
}

func TestMatchOutputByDeviceNameNotFound(t *testing.T) {
	_, err := MatchOutputByDeviceName(nil, `\\.\DISPLAY9`)
	if !errors.Is(err, ErrOutputNotFound) {
		t.Fatalf("MatchOutputByDeviceName() error = %v, want %v", err, ErrOutputNotFound)
	}
}

func TestMatchOutputByDeviceNameAmbiguous(t *testing.T) {
	outputs := []Output{
		{AdapterIndex: 0, OutputIndex: 0, DeviceName: `\\.\DISPLAY1`},
		{AdapterIndex: 1, OutputIndex: 0, DeviceName: `\\.\DISPLAY1`},
	}

	_, err := MatchOutputByDeviceName(outputs, `\\.\DISPLAY1`)
	if !errors.Is(err, ErrOutputAmbiguous) {
		t.Fatalf("MatchOutputByDeviceName() error = %v, want %v", err, ErrOutputAmbiguous)
	}
}
