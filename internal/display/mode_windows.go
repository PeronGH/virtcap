//go:build windows

package display

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const enumCurrentSettings = ^uint32(0)

const (
	dmPelsWidth        = 0x00080000
	dmPelsHeight       = 0x00100000
	dmDisplayFrequency = 0x00400000
)

const (
	dispChangeSuccessful = 0
	dispChangeBadMode    = -2
	dispChangeRestart    = 1
)

var (
	procEnumDisplaySettingsW  = windows.NewLazySystemDLL("user32.dll").NewProc("EnumDisplaySettingsW")
	procChangeDisplaySettings = windows.NewLazySystemDLL("user32.dll").NewProc("ChangeDisplaySettingsExW")
)

type devMode struct {
	DeviceName         [32]uint16
	SpecVersion        uint16
	DriverVersion      uint16
	Size               uint16
	DriverExtra        uint16
	Fields             uint32
	PositionX          int32
	PositionY          int32
	DisplayOrientation uint32
	DisplayFixedOutput uint32
	Color              int16
	Duplex             int16
	YResolution        int16
	TTOption           int16
	Collate            int16
	FormName           [32]uint16
	LogPixels          uint16
	BitsPerPel         uint32
	PelsWidth          uint32
	PelsHeight         uint32
	DisplayFlags       uint32
	DisplayFrequency   uint32
	ICMMethod          uint32
	ICMIntent          uint32
	MediaType          uint32
	DitherType         uint32
	Reserved1          uint32
	Reserved2          uint32
	PanningWidth       uint32
	PanningHeight      uint32
}

func ApplyMode(deviceName string, width int, height int, hz int) error {
	namePtr, err := windows.UTF16PtrFromString(deviceName)
	if err != nil {
		return err
	}

	mode := devMode{
		Size: uint16(unsafe.Sizeof(devMode{})),
	}

	ok, err := enumDisplaySettings(namePtr, enumCurrentSettings, &mode)
	if !ok {
		return fmt.Errorf("enumerate current mode for %s: %w", deviceName, err)
	}

	mode.PelsWidth = uint32(width)
	mode.PelsHeight = uint32(height)
	mode.DisplayFrequency = uint32(hz)
	mode.Fields |= dmPelsWidth | dmPelsHeight | dmDisplayFrequency

	result, err := changeDisplaySettings(namePtr, &mode)
	switch result {
	case dispChangeSuccessful:
		return nil
	case dispChangeBadMode:
		return fmt.Errorf("display mode %dx%d@%d is not supported on %s", width, height, hz, deviceName)
	case dispChangeRestart:
		return fmt.Errorf("display mode %dx%d@%d on %s requires a restart", width, height, hz, deviceName)
	default:
		return fmt.Errorf("change display mode on %s returned %d: %w", deviceName, result, err)
	}
}

func enumDisplaySettings(deviceName *uint16, modeNum uint32, mode *devMode) (bool, error) {
	r1, _, err := procEnumDisplaySettingsW.Call(
		uintptr(unsafe.Pointer(deviceName)),
		uintptr(modeNum),
		uintptr(unsafe.Pointer(mode)),
	)
	if r1 != 0 {
		return true, nil
	}

	if err == nil {
		return false, windows.ERROR_SUCCESS
	}

	return false, err
}

func changeDisplaySettings(deviceName *uint16, mode *devMode) (int, error) {
	r1, _, err := procChangeDisplaySettings.Call(
		uintptr(unsafe.Pointer(deviceName)),
		uintptr(unsafe.Pointer(mode)),
		0,
		0,
		0,
	)

	return int(int32(r1)), err
}
