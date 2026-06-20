package winapi

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modKernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procGetModuleHandleW           = modKernel32.NewProc("GetModuleHandleW")
	procQueryFullProcessImageNameW = modKernel32.NewProc("QueryFullProcessImageNameW")
	procCloseHandle                = modKernel32.NewProc("CloseHandle")
	procGetTickCount64             = modKernel32.NewProc("GetTickCount64")
	procOpenProcess                = modKernel32.NewProc("OpenProcess")
	procSetConsoleCtrlHandler      = modKernel32.NewProc("SetConsoleCtrlHandler")
)

// --- Constants ---

const (
	PROCESS_ALL_ACCESS                = 0x001F0FFF
	PROCESS_TERMINATE                 = 0x00000001
	PROCESS_CREATE_THREAD             = 0x00000002
	PROCESS_VM_OPERATION              = 0x00000008
	PROCESS_VM_READ                   = 0x00000010
	PROCESS_VM_WRITE                  = 0x00000020
	PROCESS_DUP_HANDLE                = 0x00000040
	PROCESS_CREATE_PROCESS            = 0x00000080
	PROCESS_SET_QUOTA                 = 0x00000100
	PROCESS_SET_INFORMATION           = 0x00000200
	PROCESS_QUERY_INFORMATION         = 0x00000400
	PROCESS_QUERY_LIMITED_INFORMATION = 0x00001000
	PROCESS_SYNCHRONIZE               = 0x00100000
)

// --- Functions ---

func GetModuleHandle(lpModuleName *uint16) uintptr {
	var pName uintptr
	if lpModuleName != nil {
		pName = uintptr(unsafe.Pointer(lpModuleName))
	}
	ret, _, _ := procGetModuleHandleW.Call(pName)
	return ret
}

func QueryFullProcessImageName(hProcess uintptr, dwFlags uint32, lpExeName *[260]uint16, lpdwSize *uint32) bool {
	ret, _, _ := procQueryFullProcessImageNameW.Call(
		hProcess,
		uintptr(dwFlags),
		uintptr(unsafe.Pointer(lpExeName)),
		uintptr(unsafe.Pointer(lpdwSize)),
	)
	return ret != 0
}

func CloseHandle(hObject uintptr) bool {
	ret, _, _ := procCloseHandle.Call(hObject)
	return ret != 0
}

func GetTickCount64() uint64 {
	ret, _, _ := procGetTickCount64.Call()
	return uint64(ret)
}

func OpenProcess(dwDesiredAccess uint32, bInheritHandle bool, dwProcessId uint32) uintptr {
	var inherit uintptr
	if bInheritHandle {
		inherit = 1
	}
	ret, _, _ := procOpenProcess.Call(uintptr(dwDesiredAccess), inherit, uintptr(dwProcessId))
	return ret
}

// SetConsoleCtrlHandler registers a console control handler callback.
// handlerRoutine must be created with syscall.NewCallback and have the
// signature: func(dwCtrlType uintptr) uintptr.
// If add is true, the handler is registered; if false, it is removed.
// Returns true on success.
//
// The caller must keep the handlerRoutine callback alive for the
// entire time it is registered — Windows holds a raw pointer to it.
func SetConsoleCtrlHandler(handlerRoutine uintptr, add bool) bool {
	var addVal uintptr
	if add {
		addVal = 1
	}
	ret, _, _ := procSetConsoleCtrlHandler.Call(handlerRoutine, addVal)
	return ret != 0
}
