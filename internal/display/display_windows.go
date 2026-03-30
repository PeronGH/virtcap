//go:build windows

package display

import (
	"errors"
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	displayDeviceActive       = 0x00000001
	eddGetDeviceInterfaceName = 0x00000001
)

var procEnumDisplayDevicesW = windows.NewLazySystemDLL("user32.dll").NewProc("EnumDisplayDevicesW")

type displayDevice struct {
	Cb           uint32
	DeviceName   [32]uint16
	DeviceString [128]uint16
	StateFlags   uint32
	DeviceID     [128]uint16
	DeviceKey    [128]uint16
}

func EnumerateParsecDisplays() ([]Snapshot, error) {
	seen := make(map[string]struct{})
	displays := make([]Snapshot, 0)

	for adapterIndex := uint32(0); ; adapterIndex++ {
		adapter := displayDevice{Cb: uint32(unsafe.Sizeof(displayDevice{}))}
		ok, err := enumDisplayDevices(nil, adapterIndex, &adapter, 0)
		if !ok {
			if errorsIsSuccess(err) {
				break
			}

			return nil, fmt.Errorf("enumerate adapters at index %d: %w", adapterIndex, err)
		}

		adapterName := windows.UTF16ToString(adapter.DeviceName[:])
		if adapterName == "" {
			continue
		}

		for monitorIndex := uint32(0); ; monitorIndex++ {
			monitor := displayDevice{Cb: uint32(unsafe.Sizeof(displayDevice{}))}
			ok, err = enumDisplayDevices(&adapter.DeviceName[0], monitorIndex, &monitor, eddGetDeviceInterfaceName)
			if !ok {
				if errorsIsSuccess(err) {
					break
				}

				return nil, fmt.Errorf("enumerate monitors for %s at index %d: %w", adapterName, monitorIndex, err)
			}

			if monitor.StateFlags&displayDeviceActive == 0 {
				continue
			}

			interfaceName := windows.UTF16ToString(monitor.DeviceID[:])
			displayCode := ParseDisplayCode(interfaceName)
			if !strings.EqualFold(displayCode, ParsecDisplayCode) {
				continue
			}

			key := strings.ToUpper(interfaceName)
			if _, exists := seen[key]; exists {
				continue
			}

			seen[key] = struct{}{}
			displays = append(displays, Snapshot{
				InterfaceName: interfaceName,
				DeviceName:    adapterName,
				DisplayCode:   displayCode,
			})
		}
	}

	return displays, nil
}

func enumDisplayDevices(device *uint16, index uint32, out *displayDevice, flags uint32) (bool, error) {
	r1, _, err := procEnumDisplayDevicesW.Call(
		uintptr(unsafe.Pointer(device)),
		uintptr(index),
		uintptr(unsafe.Pointer(out)),
		uintptr(flags),
	)
	if r1 != 0 {
		return true, nil
	}

	if err == nil {
		return false, windows.ERROR_SUCCESS
	}

	return false, err
}

func errorsIsSuccess(err error) bool {
	return err == nil || errors.Is(err, windows.ERROR_SUCCESS)
}
