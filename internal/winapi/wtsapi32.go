package winapi

import "golang.org/x/sys/windows"

var (
	modWtsapi32                          = windows.NewLazySystemDLL("wtsapi32.dll")
	procWTSRegisterSessionNotification   = modWtsapi32.NewProc("WTSRegisterSessionNotification")
	procWTSUnRegisterSessionNotification = modWtsapi32.NewProc("WTSUnRegisterSessionNotification")
)

// WTSRegisterSessionNotification registers a window to receive session change notifications.
func WTSRegisterSessionNotification(hWnd uintptr, dwFlags uint32) bool {
	ret, _, _ := procWTSRegisterSessionNotification.Call(hWnd, uintptr(dwFlags))
	return ret != 0
}

// WTSUnRegisterSessionNotification unregisters a window from session change notifications.
func WTSUnRegisterSessionNotification(hWnd uintptr) bool {
	ret, _, _ := procWTSUnRegisterSessionNotification.Call(hWnd)
	return ret != 0
}
