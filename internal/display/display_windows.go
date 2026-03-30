//go:build windows

package display

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	displayDeviceActive       = 0x00000001
	displayDeviceAttached     = 0x00000002
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
	paths, err := getDisplayPaths()
	if err != nil {
		return nil, err
	}

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

			if monitor.StateFlags&displayDeviceAttached == 0 {
				continue
			}

			interfaceName := windows.UTF16ToString(monitor.DeviceID[:])
			displayCode := ParseDisplayCode(interfaceName)
			if !strings.EqualFold(displayCode, ParsecDisplayCode) {
				continue
			}

			path, found := matchDisplayPath(paths, interfaceName)
			if !found {
				continue
			}

			key := strings.ToUpper(path)
			if _, exists := seen[key]; exists {
				continue
			}

			seen[key] = struct{}{}
			displays = append(displays, Snapshot{
				InterfaceName: interfaceName,
				DeviceName:    adapterName,
				DisplayCode:   displayCode,
				DisplayIndex:  parseDisplayIndex(path),
				Active:        monitor.StateFlags&displayDeviceActive != 0,
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

func getDisplayPaths() ([]string, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Services\monitor\Enum`, registry.QUERY_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("open monitor enum registry key: %w", err)
	}
	defer key.Close()

	count, _, err := key.GetIntegerValue("Count")
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("read monitor enum count: %w", err)
	}

	paths := make([]string, 0, int(count))
	for i := uint64(0); i < count; i++ {
		value, _, err := key.GetStringValue(strconv.FormatUint(i, 10))
		if err != nil {
			return nil, fmt.Errorf("read monitor enum path %d: %w", i, err)
		}

		paths = append(paths, value)
	}

	return paths, nil
}

func matchDisplayPath(paths []string, interfaceName string) (string, bool) {
	for _, path := range paths {
		if strings.Contains(interfaceName, strings.ReplaceAll(path, `\`, `#`)) {
			return path, true
		}
	}

	return "", false
}

func parseDisplayIndex(path string) int {
	upper := strings.ToUpper(path)
	index := strings.LastIndex(upper, "UID")
	if index < 0 {
		return -1
	}

	address, err := strconv.Atoi(path[index+3:])
	if err != nil {
		return -1
	}

	return address - 0x100
}
