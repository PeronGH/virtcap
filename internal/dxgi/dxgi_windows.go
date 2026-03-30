//go:build windows

package dxgi

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const dxgiErrorNotFound syscall.Errno = 0x887A0002

var (
	procCreateDXGIFactory1 = windows.NewLazySystemDLL("dxgi.dll").NewProc("CreateDXGIFactory1")

	iidIDXGIFactory1 = windows.GUID{Data1: 0x770aae78, Data2: 0xf26f, Data3: 0x4dba, Data4: [8]byte{0xa8, 0x29, 0x25, 0x3c, 0x83, 0xd1, 0xb3, 0x87}}
)

type rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type dxgiOutputDesc struct {
	DeviceName         [32]uint16
	DesktopCoordinates rect
	AttachedToDesktop  int32
	Rotation           uint32
	Monitor            windows.Handle
}

type dxgiFactory1 struct {
	lpVtbl *dxgiFactory1Vtbl
}

type dxgiFactory1Vtbl struct {
	QueryInterface      uintptr
	AddRef              uintptr
	Release             uintptr
	SetPrivateData      uintptr
	SetPrivateDataIface uintptr
	GetPrivateData      uintptr
	GetParent           uintptr
	EnumAdapters        uintptr
	MakeWindowAssoc     uintptr
	GetWindowAssoc      uintptr
	CreateSwapChain     uintptr
	CreateSoftwareAdap  uintptr
	EnumAdapters1       uintptr
	IsCurrent           uintptr
}

type dxgiAdapter1 struct {
	lpVtbl *dxgiAdapter1Vtbl
}

type dxgiAdapter1Vtbl struct {
	QueryInterface      uintptr
	AddRef              uintptr
	Release             uintptr
	SetPrivateData      uintptr
	SetPrivateDataIface uintptr
	GetPrivateData      uintptr
	GetParent           uintptr
	EnumOutputs         uintptr
	GetDesc             uintptr
	CheckInterfaceSup   uintptr
	GetDesc1            uintptr
}

type dxgiOutput struct {
	lpVtbl *dxgiOutputVtbl
}

type dxgiOutputVtbl struct {
	QueryInterface              uintptr
	AddRef                      uintptr
	Release                     uintptr
	SetPrivateData              uintptr
	SetPrivateDataIface         uintptr
	GetPrivateData              uintptr
	GetParent                   uintptr
	GetDesc                     uintptr
	FindClosestMatchingMode     uintptr
	WaitForVBlank               uintptr
	TakeOwnership               uintptr
	ReleaseOwnership            uintptr
	GetGammaControlCapabilities uintptr
	SetGammaControl             uintptr
	GetGammaControl             uintptr
	SetDisplaySurface           uintptr
	GetDisplaySurfaceData       uintptr
	GetFrameStatistics          uintptr
}

func EnumerateOutputs() ([]Output, error) {
	factory, err := createFactory1()
	if err != nil {
		return nil, err
	}
	defer factory.Release()

	outputs := make([]Output, 0)
	for adapterIndex := 0; ; adapterIndex++ {
		adapter, err := factory.EnumAdapters1(uint32(adapterIndex))
		if err != nil {
			if err == dxgiErrorNotFound {
				break
			}

			return nil, fmt.Errorf("enumerate DXGI adapter %d: %w", adapterIndex, err)
		}

		for outputIndex := 0; ; outputIndex++ {
			output, err := adapter.EnumOutputs(uint32(outputIndex))
			if err != nil {
				if err == dxgiErrorNotFound {
					break
				}

				adapter.Release()
				return nil, fmt.Errorf("enumerate DXGI output %d on adapter %d: %w", outputIndex, adapterIndex, err)
			}

			desc, err := output.GetDesc()
			output.Release()
			if err != nil {
				adapter.Release()
				return nil, fmt.Errorf("get DXGI output desc for adapter %d output %d: %w", adapterIndex, outputIndex, err)
			}

			if desc.AttachedToDesktop == 0 {
				continue
			}

			outputs = append(outputs, Output{
				AdapterIndex: adapterIndex,
				OutputIndex:  outputIndex,
				DeviceName:   windows.UTF16ToString(desc.DeviceName[:]),
			})
		}

		adapter.Release()
	}

	return outputs, nil
}

func createFactory1() (*dxgiFactory1, error) {
	var factory *dxgiFactory1
	hr, _, _ := procCreateDXGIFactory1.Call(
		uintptr(unsafe.Pointer(&iidIDXGIFactory1)),
		uintptr(unsafe.Pointer(&factory)),
	)
	if int32(hr) < 0 {
		return nil, syscall.Errno(hr)
	}

	return factory, nil
}

func (f *dxgiFactory1) EnumAdapters1(index uint32) (*dxgiAdapter1, error) {
	var adapter *dxgiAdapter1
	hr, _, _ := syscall.SyscallN(
		f.lpVtbl.EnumAdapters1,
		uintptr(unsafe.Pointer(f)),
		uintptr(index),
		uintptr(unsafe.Pointer(&adapter)),
	)
	if int32(hr) < 0 {
		return nil, syscall.Errno(hr)
	}

	return adapter, nil
}

func (f *dxgiFactory1) Release() {
	if f == nil {
		return
	}

	syscall.SyscallN(f.lpVtbl.Release, uintptr(unsafe.Pointer(f)))
}

func (a *dxgiAdapter1) EnumOutputs(index uint32) (*dxgiOutput, error) {
	var output *dxgiOutput
	hr, _, _ := syscall.SyscallN(
		a.lpVtbl.EnumOutputs,
		uintptr(unsafe.Pointer(a)),
		uintptr(index),
		uintptr(unsafe.Pointer(&output)),
	)
	if int32(hr) < 0 {
		return nil, syscall.Errno(hr)
	}

	return output, nil
}

func (a *dxgiAdapter1) Release() {
	if a == nil {
		return
	}

	syscall.SyscallN(a.lpVtbl.Release, uintptr(unsafe.Pointer(a)))
}

func (o *dxgiOutput) GetDesc() (dxgiOutputDesc, error) {
	var desc dxgiOutputDesc
	hr, _, _ := syscall.SyscallN(
		o.lpVtbl.GetDesc,
		uintptr(unsafe.Pointer(o)),
		uintptr(unsafe.Pointer(&desc)),
	)
	if int32(hr) < 0 {
		return dxgiOutputDesc{}, syscall.Errno(hr)
	}

	return desc, nil
}

func (o *dxgiOutput) Release() {
	if o == nil {
		return
	}

	syscall.SyscallN(o.lpVtbl.Release, uintptr(unsafe.Pointer(o)))
}
