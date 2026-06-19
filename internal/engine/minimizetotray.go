package engine

import (
	"durablewindows/internal/logger"
	"durablewindows/internal/winapi"
)

// minimizeToTrayWindows tracks windows that were parked to tray
// (via Shift+minimize or crash-recovery restore).
var minimizeToTrayWindows = make(map[uintptr]bool)

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

// StartMinimizeToTray is a no-op — parking is now driven by
// EVENT_SYSTEM_MINIMIZESTART in onMinimizeStart (processor.go).
// Kept for API compatibility with tray.go.
func (p *Processor) StartMinimizeToTray() {
	logger.Parking("shift-minimize-to-tray enabled", "")
}

// StopMinimizeToTray is a no-op.
func (p *Processor) StopMinimizeToTray() {
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
