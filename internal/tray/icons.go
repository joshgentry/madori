package tray

import (
	"madori"
	"madori/internal/winapi"
)

// loadIcons loads the embedded icon data and returns HICON handles for each state.
func loadIcons() (idle, busy uintptr) {
	idle = loadICO(madori.IdleIcoData)
	busy = loadICO(madori.BusyIcoData)
	if busy == 0 {
		busy = idle
	}
	return
}

// loadICO loads a Windows .ico file from raw bytes and returns an HICON.
func loadICO(data []byte) uintptr {
	if len(data) < 22 {
		return 0
	}

	// Parse .ico header: type must be 1 (icon)
	if data[2] != 1 || data[3] != 0 {
		return 0
	}
	count := int(data[4]) | int(data[5])<<8
	if count == 0 {
		return 0
	}

	// Find the largest icon entry by dimensions
	type icoEntry struct {
		offset uint32
	}
	var best icoEntry
	bestSize := uint32(0)

	for i := 0; i < count; i++ {
		entryOff := 6 + i*16
		if entryOff+16 > len(data) {
			break
		}
		w := data[entryOff]
		h := data[entryOff+1]
		offset := uint32(data[entryOff+12]) | uint32(data[entryOff+13])<<8 |
			uint32(data[entryOff+14])<<16 | uint32(data[entryOff+15])<<24

		dim := uint32(w) * uint32(h)
		if w == 0 {
			dim = 256 * 256 // 0 means 256
		}
		if dim > bestSize {
			bestSize = dim
			best = icoEntry{offset: offset}
		}
	}

	if best.offset == 0 || int(best.offset) >= len(data) {
		return 0
	}

	iconData := data[best.offset:]
	return winapi.CreateIconFromResourceEx(
		iconData, uint32(len(iconData)),
		true,
		0x00030000,
		0, 0,
		0, // LR_DEFAULTSIZE
	)
}

// initIcons loads all icons into the TrayApp.
func (t *TrayApp) initIcons() {
	t.idleIcon, t.busyIcon = loadIcons()
}
