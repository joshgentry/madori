package engine

import (
	"sync"
	"time"
	"unsafe"

	"durablewindows/internal/logger"
	"durablewindows/internal/winapi"
)

// minimizeToTrayWindows tracks windows that were parked to tray
// (via Shift+minimize or crash-recovery restore).
var minimizeToTrayWindows = make(map[uintptr]bool)

// Shift-key tracking for Shift+minimize.  A WH_KEYBOARD_LL hook updates
// these on every Shift key event so onMinimizeStart can see the state
// even if the user released Shift before the out-of-context WinEvent fires.
var (
	shiftHeld        bool
	lastShiftUp      time.Time
	shiftGracePeriod = 300 * time.Millisecond
	shiftStateMu     sync.Mutex
	kbHookHandle     uintptr
	kbHookRunning    bool // guard against duplicate install
)

// SetShiftGracePeriod sets how long after Shift is released that a minimize
// is still treated as Shift+minimize (default 300 ms). Call before StartMinimizeToTray.
func SetShiftGracePeriod(d time.Duration) {
	shiftStateMu.Lock()
	shiftGracePeriod = d
	shiftStateMu.Unlock()
}

// isShiftDownOrRecent returns true if either Shift key is held right now,
// or was released within the grace period (WinEvent latency compensation).
func isShiftDownOrRecent() bool {
	shiftStateMu.Lock()
	defer shiftStateMu.Unlock()
	if shiftHeld {
		return true
	}
	return time.Since(lastShiftUp) < shiftGracePeriod
}

// keyboardHookProc is the WH_KEYBOARD_LL callback. It only tracks Shift key
// state; all other keys are passed through immediately.
func keyboardHookProc(nCode, wParam, lParam uintptr) uintptr {
	if int32(nCode) >= winapi.HC_ACTION {
		vkCode := (*winapi.KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam)).VkCode
		if vkCode == winapi.VK_LSHIFT || vkCode == winapi.VK_RSHIFT {
			shiftStateMu.Lock()
			if wParam == winapi.WM_KEYDOWN || wParam == winapi.WM_SYSKEYDOWN {
				shiftHeld = true
			} else {
				shiftHeld = false
				lastShiftUp = time.Now()
			}
			shiftStateMu.Unlock()
		}
	}
	return winapi.CallNextHookEx(kbHookHandle, nCode, wParam, lParam)
}

// Tray icon tracking for parked windows.
var (
	parkedIconUID            = make(map[uintptr]uint32) // hwnd -> tray icon UID
	parkedIconRev            = make(map[uint32]uintptr) // UID -> hwnd
	nextParkedIconUID uint32 = 100                      // must match FirstParkedIconUID in tray.go
)

// persistParkedWindows writes the current set of parked HWNDs to BoltDB so
// crashed sessions can recover. No-op if the store hasn't been initialised
// (one-shot CLI commands).
func persistParkedWindows() {
	if store == nil {
		return
	}
	hwnds := make([]uintptr, 0, len(minimizeToTrayWindows))
	for hwnd := range minimizeToTrayWindows {
		hwnds = append(hwnds, hwnd)
	}
	_ = store.SaveParkedWindows(hwnds)
}

// parkWindow hides the window and queues tray icon creation.
func parkWindow(p *Processor, hwnd uintptr) {
	winapi.ShowWindowAsync(hwnd, winapi.SW_HIDE)
	minimizeToTrayWindows[hwnd] = true
	persistParkedWindows()
	logger.Parking("minimized to tray", "%s", p.WindowDesc(hwnd))
	winapi.PostMessage(p.trayHWnd, winapi.WM_APP_PARKED, uintptr(hwnd), 0)
}

// StartMinimizeToTray installs the WH_KEYBOARD_LL hook that tracks Shift-key
// state for Shift+minimize parking. Must be called from a thread with a
// message pump (the tray app's main thread).
func (p *Processor) StartMinimizeToTray() {
	if kbHookRunning {
		return
	}
	kbHookHandle = winapi.SetWindowsHookExDirect(winapi.WH_KEYBOARD_LL, keyboardHookProc, 0, 0)
	kbHookRunning = true
	logger.Parking("shift-minimize-to-tray enabled", "WH_KEYBOARD_LL (handle=%d)", kbHookHandle)
}

// StopMinimizeToTray removes the keyboard hook.
func (p *Processor) StopMinimizeToTray() {
	if kbHookHandle != 0 {
		winapi.UnhookWindowsHookEx(kbHookHandle)
		kbHookHandle = 0
	}
	kbHookRunning = false
	logger.Parking("shift-minimize-to-tray disabled", "")
}

// SetTrayWindow stores the tray message window HWND so parked-window tray
// icons can route their callbacks to the correct window.
func (p *Processor) SetTrayWindow(hwnd uintptr) {
	p.trayHWnd = hwnd
}

// AddParkedTrayIcon creates a system-tray icon for a parked window.
func (p *Processor) AddParkedTrayIcon(hwnd uintptr) {
	uid := nextParkedIconUID
	nextParkedIconUID++
	parkedIconUID[hwnd] = uid
	parkedIconRev[uid] = hwnd

	title := GetWindowTitle(hwnd)
	if title == "" {
		title = GetWindowClassName(hwnd)
	}

	var hIcon uintptr
	if winapi.SendMessage(hwnd, winapi.WM_GETICON, winapi.ICON_SMALL, 0) != 0 {
		hIcon = winapi.SendMessage(hwnd, winapi.WM_GETICON, winapi.ICON_SMALL, 0)
	}
	if hIcon == 0 {
		hIcon = winapi.GetClassLongPtr(hwnd, winapi.GCLP_HICONSM)
	}
	if hIcon == 0 {
		hIcon = winapi.GetClassLongPtr(hwnd, winapi.GCLP_HICON)
	}

	nid := winapi.NOTIFYICONDATA{
		HWnd:             p.trayHWnd,
		UID:              uid,
		UFlags:           winapi.NIF_MESSAGE | winapi.NIF_TIP,
		UCallbackMessage: winapi.WM_TRAYICON,
		HIcon:            hIcon,
	}
	if hIcon != 0 {
		nid.UFlags |= winapi.NIF_ICON
	}
	copy16(nid.SzTip[:], title)
	winapi.ShellNotifyIcon(winapi.NIM_ADD, &nid)
	logger.Parking("parked icon added", "%s (uid=%d)", title, uid)
}

func (p *Processor) removeParkedTrayIcon(hwnd uintptr) {
	uid, ok := parkedIconUID[hwnd]
	if !ok {
		return
	}
	nid := winapi.NOTIFYICONDATA{
		HWnd: p.trayHWnd,
		UID:  uid,
	}
	winapi.ShellNotifyIcon(winapi.NIM_DELETE, &nid)
	delete(parkedIconUID, hwnd)
	delete(parkedIconRev, uid)
}

// FindParkedWindowByUID returns the HWND for a parked window given its tray icon UID.
func (p *Processor) FindParkedWindowByUID(uid uint32) uintptr {
	return parkedIconRev[uid]
}

// RestoreFromTray restores a window that was parked to tray.
func (p *Processor) RestoreFromTray(hwnd uintptr) {
	if !minimizeToTrayWindows[hwnd] {
		return
	}

	p.removeParkedTrayIcon(hwnd)

	winapi.ShowWindow(hwnd, winapi.SW_RESTORE)
	winapi.SetForegroundWindow(hwnd)

	delete(minimizeToTrayWindows, hwnd)
	persistParkedWindows()

	if metricsList, ok := p.monitorApplications[p.curDisplayKey][hwnd]; ok && len(metricsList) > 0 {
		p.restoreSingleWindow(hwnd, metricsList[len(metricsList)-1])
	}

	logger.Parking("restored from tray", "%s", p.WindowDesc(hwnd))
}

// restoreOrphanedParkedWindows loads the parked-window list from BoltDB and
// restores any windows that are still alive. This recovers from a crash where
// RestoreAllParked() never ran. On a clean shutdown the list is empty.
func (p *Processor) restoreOrphanedParkedWindows() {
	if store == nil {
		return
	}
	hwnds, err := store.LoadParkedWindows()
	if err != nil || len(hwnds) == 0 {
		return
	}
	for _, hwnd := range hwnds {
		if !winapi.IsWindow(hwnd) {
			continue
		}
		// Populate the in-memory map so RestoreFromTray's guard passes.
		// It will remove the entry and persist the updated list.
		minimizeToTrayWindows[hwnd] = true
		logger.Parking("orphaned park restored", "%s (crash recovery)", p.WindowDesc(hwnd))
		p.RestoreFromTray(hwnd)
	}
	// Clear the bucket now that we've restored everything.
	_ = store.SaveParkedWindows(nil)
}

// RestoreParkedWindowsCmd is the CLI one-shot (-restore_parked_windows):
// loads parked-window HWNDs from the database, restores any that are still
// alive, clears the bucket, and exits. Useful as a manual recovery tool.
func (p *Processor) RestoreParkedWindowsCmd() {
	p.restoreOrphanedParkedWindows()
}

// IsMinimizedToTray returns true if the window was parked to tray.
func (p *Processor) IsMinimizedToTray(hwnd uintptr) bool {
	return minimizeToTrayWindows[hwnd]
}

// GetMinimizedToTrayWindows returns all windows currently parked to tray.
func (p *Processor) GetMinimizedToTrayWindows() []uintptr {
	var result []uintptr
	for hwnd := range minimizeToTrayWindows {
		if winapi.IsWindow(hwnd) {
			result = append(result, hwnd)
		} else {
			delete(minimizeToTrayWindows, hwnd)
		}
	}
	return result
}

func copy16(dst []uint16, src string) {
	for i := 0; i < len(dst)-1 && i < len(src); i++ {
		dst[i] = uint16(src[i])
	}
}
