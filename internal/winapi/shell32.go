package winapi

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modShell32 = windows.NewLazySystemDLL("shell32.dll")

	procShellNotifyIconW             = modShell32.NewProc("Shell_NotifyIconW")
	procSHAppBarMessage              = modShell32.NewProc("SHAppBarMessage")
	procSHQueryUserNotificationState = modShell32.NewProc("SHQueryUserNotificationState")
)

// --- NOTIFYICONDATA ---

// NOTIFYICONDATA represents the Windows NOTIFYICONDATAW structure.
type NOTIFYICONDATA struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GUIDItem         windows.GUID
	HBalloonIcon     uintptr
}

// --- Shell_NotifyIcon constants ---

const (
	NIM_ADD        = 0x00000000
	NIM_MODIFY     = 0x00000001
	NIM_DELETE     = 0x00000002
	NIM_SETVERSION = 0x00000004
)

const (
	NIF_MESSAGE = 0x00000001
	NIF_ICON    = 0x00000002
	NIF_TIP     = 0x00000004
	NIF_STATE   = 0x00000008
	NIF_INFO    = 0x00000010
	NIF_GUID    = 0x00000020
)

// NOTIFYICON_VERSION_4 enables alpha-blended tray icons, GUID items,
// and balloon icon customization (Windows Vista+).
const NOTIFYICON_VERSION_4 = 4

const (
	NIIF_NONE       = 0x00000000
	NIIF_INFO       = 0x00000001
	NIIF_WARNING    = 0x00000002
	NIIF_ERROR      = 0x00000003
	NIIF_USER       = 0x00000004
	NIIF_LARGE_ICON = 0x00000020
)

const (
	NIS_HIDDEN     = 0x00000001
	NIS_SHAREDICON = 0x00000002
)

func ShellNotifyIcon(dwMessage uint32, nid *NOTIFYICONDATA) bool {
	nid.CbSize = uint32(unsafe.Sizeof(NOTIFYICONDATA{}))
	ret, _, _ := procShellNotifyIconW.Call(uintptr(dwMessage), uintptr(unsafe.Pointer(nid)))
	return ret != 0
}

// --- APP_BAR_DATA ---

// APP_BAR_DATA represents the Windows APPBARDATA structure.
type APP_BAR_DATA struct {
	CbSize           uint32
	HWnd             uintptr
	UCallbackMessage int32
	UEdge            int32
	RC               RECT
	LParam           uintptr
}

// --- SHAppBarMessage constants ---

const (
	ABM_NEW            = 0x00
	ABM_REMOVE         = 0x01
	ABM_QUERYPOS       = 0x02
	ABM_SETPOS         = 0x03
	ABM_GETSTATE       = 0x04
	ABM_GETTASKBARPOS  = 0x05
	ABM_GETAUTOHIDEBAR = 0x07
	ABM_SETAUTOHIDEBAR = 0x08
	ABM_SETSTATE       = 0x0A
)

const (
	ABE_LEFT   = 0
	ABE_TOP    = 1
	ABE_RIGHT  = 2
	ABE_BOTTOM = 3
)

const (
	ABS_AUTOHIDE    = 0x01
	ABS_ALWAYSONTOP = 0x02
)

func SHAppBarMessage(dwMessage uint32, pData *APP_BAR_DATA) uintptr {
	ret, _, _ := procSHAppBarMessage.Call(uintptr(dwMessage), uintptr(unsafe.Pointer(pData)))
	return ret
}

// --- SHQueryUserNotificationState ---

const (
	QUNS_NOT_PRESENT             = 1
	QUNS_BUSY                    = 2
	QUNS_RUNNING_D3D_FULL_SCREEN = 3
	QUNS_PRESENTATION_MODE       = 4
	QUNS_ACCEPTS_NOTIFICATIONS   = 5
	QUNS_QUIET_TIME              = 6
)

func SHQueryUserNotificationState(pquns *uint32) int32 {
	ret, _, _ := procSHQueryUserNotificationState.Call(uintptr(unsafe.Pointer(pquns)))
	return int32(ret)
}
