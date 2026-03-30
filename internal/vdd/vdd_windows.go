//go:build windows

package vdd

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/sys/windows"
)

const (
	ioctlAdd    = 0x0022e004
	ioctlUpdate = 0x0022a00c

	driverAddTimeout    = 5 * time.Second
	driverUpdateTimeout = time.Second
)

var (
	adapterGUID = windows.GUID{Data1: 0x00b41627, Data2: 0x04c4, Data3: 0x429e, Data4: [8]byte{0xa2, 0x6e, 0x02, 0x65, 0xcf, 0x50, 0xc8, 0xfa}}
	classGUID   = windows.GUID{Data1: 0x4d36e968, Data2: 0xe325, Data3: 0x11ce, Data4: [8]byte{0xbf, 0xc1, 0x08, 0x00, 0x2b, 0xe1, 0x03, 0x18}}
)

const hardwareID = `Root\Parsec\VDA`

const (
	cmProbNeedRestart      = 0x0000000E
	cmProbDisabled         = 0x00000016
	cmProbHardwareDisabled = 0x0000001D
	cmProbDisabledService  = 0x00000020
	cmProbFailedPostStart  = 0x0000002B
)

type Status int

const (
	StatusOK Status = iota
	StatusInaccessible
	StatusUnknown
	StatusUnknownProblem
	StatusDisabled
	StatusDriverError
	StatusRestartRequired
	StatusDisabledService
	StatusNotInstalled
)

type Device struct {
	handle windows.Handle
}

func (s Status) String() string {
	switch s {
	case StatusOK:
		return "OK"
	case StatusInaccessible:
		return "INACCESSIBLE"
	case StatusUnknown:
		return "UNKNOWN"
	case StatusUnknownProblem:
		return "UNKNOWN_PROBLEM"
	case StatusDisabled:
		return "DISABLED"
	case StatusDriverError:
		return "DRIVER_ERROR"
	case StatusRestartRequired:
		return "RESTART_REQUIRED"
	case StatusDisabledService:
		return "DISABLED_SERVICE"
	case StatusNotInstalled:
		return "NOT_INSTALLED"
	default:
		return fmt.Sprintf("Status(%d)", s)
	}
}

func QueryStatus() (Status, string, error) {
	devInfo, err := windows.SetupDiGetClassDevsEx(&classGUID, "", 0, windows.DIGCF_PRESENT, 0, "")
	if err != nil {
		return StatusInaccessible, "", err
	}
	defer devInfo.Close()

	for index := 0; ; index++ {
		devInfoData, err := devInfo.EnumDeviceInfo(index)
		if err != nil {
			if err == windows.ERROR_NO_MORE_ITEMS {
				return StatusNotInstalled, "", nil
			}

			return StatusInaccessible, "", err
		}

		if !matchesHardwareID(devInfo, devInfoData, hardwareID) {
			continue
		}

		instanceID, err := devInfo.DeviceInstanceID(devInfoData)
		if err != nil {
			return StatusInaccessible, "", err
		}

		status, err := queryDevNodeStatus(devInfoData.DevInst)
		return status, instanceID, err
	}
}

func Open(instanceID string) (*Device, error) {
	interfaces, err := windows.CM_Get_Device_Interface_List(instanceID, &adapterGUID, windows.CM_GET_DEVICE_INTERFACE_LIST_PRESENT)
	if err != nil {
		return nil, fmt.Errorf("query device interfaces: %w", err)
	}

	if len(interfaces) == 0 {
		return nil, errors.New("no live device interface found for Parsec VDD")
	}

	pathPtr, err := windows.UTF16PtrFromString(interfaces[0])
	if err != nil {
		return nil, err
	}

	handle, err := windows.CreateFile(
		pathPtr,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_NO_BUFFERING|windows.FILE_FLAG_OVERLAPPED|windows.FILE_FLAG_WRITE_THROUGH,
		0,
	)
	if err != nil {
		return nil, err
	}

	device := &Device{handle: handle}
	if err := device.Update(); err != nil {
		device.Close()
		return nil, fmt.Errorf("initial update: %w", err)
	}

	return device, nil
}

func (d *Device) AddDisplay() (int, error) {
	output := make([]byte, 4)
	if _, err := d.ioctl(ioctlAdd, nil, output, driverAddTimeout); err != nil {
		return 0, err
	}

	if err := d.Update(); err != nil {
		return 0, err
	}

	return int(int32(binary.LittleEndian.Uint32(output))), nil
}

func (d *Device) Update() error {
	_, err := d.ioctl(ioctlUpdate, nil, nil, driverUpdateTimeout)
	return err
}

func (d *Device) Close() error {
	if d == nil || d.handle == 0 || d.handle == windows.InvalidHandle {
		return nil
	}

	err := windows.CloseHandle(d.handle)
	d.handle = 0
	return err
}

func RunKeepAlive(ctx context.Context, device *Device, interval time.Duration) error {
	if err := device.Update(); err != nil {
		return err
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := device.Update(); err != nil {
				return err
			}
		}
	}
}

func (d *Device) ioctl(code uint32, input []byte, output []byte, timeout time.Duration) (uint32, error) {
	inBuffer := make([]byte, 32)
	copy(inBuffer, input)

	var inPtr *byte
	if len(inBuffer) > 0 {
		inPtr = &inBuffer[0]
	}

	var outPtr *byte
	if len(output) > 0 {
		outPtr = &output[0]
	}

	event, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(event)

	overlapped := windows.Overlapped{HEvent: event}
	err = windows.DeviceIoControl(d.handle, code, inPtr, uint32(len(inBuffer)), outPtr, uint32(len(output)), nil, &overlapped)
	if err != nil && !errors.Is(err, windows.ERROR_IO_PENDING) {
		return 0, err
	}

	waitStatus, err := windows.WaitForSingleObject(event, durationToMilliseconds(timeout))
	if err != nil {
		return 0, err
	}

	if waitStatus == uint32(windows.WAIT_TIMEOUT) {
		return 0, fmt.Errorf("ioctl 0x%x timed out", code)
	}

	if waitStatus != windows.WAIT_OBJECT_0 {
		return 0, fmt.Errorf("ioctl 0x%x wait returned unexpected status 0x%x", code, waitStatus)
	}

	var transferred uint32
	if err := windows.GetOverlappedResult(d.handle, &overlapped, &transferred, false); err != nil {
		return 0, err
	}

	return transferred, nil
}

func durationToMilliseconds(duration time.Duration) uint32 {
	if duration <= 0 {
		return 0
	}

	ms := duration / time.Millisecond
	if ms > time.Duration(^uint32(0)) {
		return ^uint32(0)
	}

	return uint32(ms)
}

func matchesHardwareID(devInfo windows.DevInfo, devInfoData *windows.DevInfoData, want string) bool {
	property, err := devInfo.DeviceRegistryProperty(devInfoData, windows.SPDRP_HARDWAREID)
	if err != nil {
		return false
	}

	switch value := property.(type) {
	case string:
		return strings.EqualFold(value, want)
	case []string:
		for _, candidate := range value {
			if strings.EqualFold(candidate, want) {
				return true
			}
		}
	}

	return false
}

func queryDevNodeStatus(devInst windows.DEVINST) (Status, error) {
	var devStatus uint32
	var problemCode uint32
	if err := windows.CM_Get_DevNode_Status(&devStatus, &problemCode, devInst, 0); err != nil {
		return StatusNotInstalled, err
	}

	if devStatus&(windows.DN_DRIVER_LOADED|windows.DN_STARTED) != 0 {
		return StatusOK, nil
	}

	if devStatus&windows.DN_HAS_PROBLEM != 0 {
		switch problemCode {
		case cmProbNeedRestart:
			return StatusRestartRequired, nil
		case cmProbDisabled, cmProbHardwareDisabled:
			return StatusDisabled, nil
		case cmProbDisabledService:
			return StatusDisabledService, nil
		case cmProbFailedPostStart:
			return StatusDriverError, nil
		default:
			return StatusUnknownProblem, nil
		}
	}

	return StatusUnknown, nil
}
