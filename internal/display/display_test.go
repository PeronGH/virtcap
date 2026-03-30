package display

import (
	"errors"
	"testing"
)

func TestFindNewParsecDisplay(t *testing.T) {
	before := []Snapshot{
		{InterfaceName: `\\?\DISPLAY#PSCCDD0#UID100#{monitor-guid}`, DeviceName: `\\.\DISPLAY1`, DisplayCode: ParsecDisplayCode},
	}
	after := []Snapshot{
		before[0],
		{InterfaceName: `\\?\DISPLAY#PSCCDD0#UID101#{monitor-guid}`, DeviceName: `\\.\DISPLAY2`, DisplayCode: ParsecDisplayCode},
	}

	got, err := FindNewParsecDisplay(before, after, ParsecDisplayCode)
	if err != nil {
		t.Fatalf("FindNewParsecDisplay() error = %v", err)
	}

	if got.InterfaceName != after[1].InterfaceName {
		t.Fatalf("FindNewParsecDisplay() interface = %q, want %q", got.InterfaceName, after[1].InterfaceName)
	}
}

func TestFindNewParsecDisplayNoMatch(t *testing.T) {
	before := []Snapshot{
		{InterfaceName: `\\?\DISPLAY#PSCCDD0#UID100#{monitor-guid}`, DeviceName: `\\.\DISPLAY1`, DisplayCode: ParsecDisplayCode},
	}

	_, err := FindNewParsecDisplay(before, before, ParsecDisplayCode)
	if !errors.Is(err, ErrNoNewParsecDisplay) {
		t.Fatalf("FindNewParsecDisplay() error = %v, want %v", err, ErrNoNewParsecDisplay)
	}
}

func TestFindNewParsecDisplayAmbiguous(t *testing.T) {
	before := []Snapshot{
		{InterfaceName: `\\?\DISPLAY#PSCCDD0#UID100#{monitor-guid}`, DeviceName: `\\.\DISPLAY1`, DisplayCode: ParsecDisplayCode},
	}
	after := []Snapshot{
		before[0],
		{InterfaceName: `\\?\DISPLAY#PSCCDD0#UID101#{monitor-guid}`, DeviceName: `\\.\DISPLAY2`, DisplayCode: ParsecDisplayCode},
		{InterfaceName: `\\?\DISPLAY#PSCCDD0#UID102#{monitor-guid}`, DeviceName: `\\.\DISPLAY3`, DisplayCode: ParsecDisplayCode},
	}

	_, err := FindNewParsecDisplay(before, after, ParsecDisplayCode)
	if !errors.Is(err, ErrAmbiguousNewParsecDisplay) {
		t.Fatalf("FindNewParsecDisplay() error = %v, want %v", err, ErrAmbiguousNewParsecDisplay)
	}
}

func TestParseDisplayCode(t *testing.T) {
	deviceID := `\\?\DISPLAY#PSCCDD0#5&31036591&0&UID4352#{e6f07b5f-ee97-4a90-b076-33f57bf4eaa7}`

	if got := ParseDisplayCode(deviceID); got != ParsecDisplayCode {
		t.Fatalf("ParseDisplayCode() = %q, want %q", got, ParsecDisplayCode)
	}
}
