package winapi

import "golang.org/x/sys/windows"

var (
	modGdi32   = windows.NewLazySystemDLL("gdi32.dll")
	procBitBlt = modGdi32.NewProc("BitBlt")
)

const SRCCOPY = 0x00CC0020

// BitBlt performs a bit-block transfer from a source device context to a destination device context.
func BitBlt(hdcDest uintptr, xDest, yDest, cx, cy int32, hdcSrc uintptr, xSrc, ySrc int32, rop uint32) bool {
	ret, _, _ := procBitBlt.Call(
		uintptr(hdcDest),
		uintptr(xDest), uintptr(yDest), uintptr(cx), uintptr(cy),
		uintptr(hdcSrc),
		uintptr(xSrc), uintptr(ySrc),
		uintptr(rop),
	)
	return ret != 0
}
