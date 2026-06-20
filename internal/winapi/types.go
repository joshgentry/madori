package winapi

import (
	"unsafe"
)

// POINT represents a Windows POINT structure.
type POINT struct {
	X int32
	Y int32
}

func (p POINT) Equals(other POINT) bool {
	return p.X == other.X && p.Y == other.Y
}

// RECT represents a Windows RECT structure.
type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

func (r RECT) Width() int32  { return r.Right - r.Left }
func (r RECT) Height() int32 { return r.Bottom - r.Top }

func (r RECT) Equals(other RECT) bool {
	return r.Left == other.Left && r.Top == other.Top &&
		r.Right == other.Right && r.Bottom == other.Bottom
}

// WINDOWPLACEMENT represents the Windows WINDOWPLACEMENT structure.
type WINDOWPLACEMENT struct {
	Length         uint32
	Flags          uint32
	ShowCmd        uint32
	MinPosition    POINT
	MaxPosition    POINT
	NormalPosition RECT
	RectDevice     RECT
}

// DefaultWINDOWPLACEMENT returns a WINDOWPLACEMENT with Length set correctly.
func DefaultWINDOWPLACEMENT() WINDOWPLACEMENT {
	return WINDOWPLACEMENT{
		Length: uint32(unsafe.Sizeof(WINDOWPLACEMENT{})),
	}
}

// CURSORINFO represents the Windows CURSORINFO structure.
type CURSORINFO struct {
	CbSize      uint32
	Flags       uint32
	HCursor     uintptr
	PTScreenPos POINT
}

// Display represents a monitor/display device.
type Display struct {
	DeviceName string
	Position   RECT
	Flags      uint32
}

// MOUSEHOOKSTRUCT represents low-level mouse hook info.
type MOUSEHOOKSTRUCT struct {
	PT           POINT
	HWnd         uintptr
	WHitTestCode uint32
	DWExtraInfo  uintptr
}

// CWPRETSTRUCT represents the structure passed to WH_CALLWNDPROCRET hook procs.
type CWPRETSTRUCT struct {
	LResult uintptr
	LParam  uintptr
	WParam  uintptr
	Message uint32
	HWnd    uintptr
}

// KBDLLHOOKSTRUCT represents the structure passed to WH_KEYBOARD_LL hook procs.
type KBDLLHOOKSTRUCT struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DWExtraInfo uintptr
}
