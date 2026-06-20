package winapi

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modDwmapi                 = windows.NewLazySystemDLL("dwmapi.dll")
	procDwmGetWindowAttribute = modDwmapi.NewProc("DwmGetWindowAttribute")
)

// DwmGetWindowAttribute retrieves a DWM attribute for a window.
// Returns false if the call fails (HRESULT != S_OK).
func DwmGetWindowAttribute(hwnd uintptr, dwAttribute uint32, pvAttribute unsafe.Pointer, cbAttribute uint32) bool {
	ret, _, _ := procDwmGetWindowAttribute.Call(hwnd, uintptr(dwAttribute), uintptr(pvAttribute), uintptr(cbAttribute))
	return ret == 0 // S_OK
}
