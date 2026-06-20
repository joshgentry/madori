package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc/eventlog"
)

// Level is a log severity level used as the first parameter of every
// logging function. The -log_level CLI flag sets the minimum level;
// messages below it are suppressed.
type Level int

const (
	LevelTrace Level = iota // most verbose
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError // least verbose
)

// levelChar maps a Level to its single-character prefix for the output format.
var levelChar = map[Level]string{
	LevelTrace: "T",
	LevelDebug: "D",
	LevelInfo:  "I",
	LevelWarn:  "W",
	LevelError: "E",
}

// levelColor maps a Level to its ANSI color for the prefix letter.
var levelColor = map[Level]string{
	LevelTrace: ColorMagenta, // purple
	LevelDebug: ColorGray,    // gray
	LevelInfo:  ColorGreen,   // green
	LevelWarn:  ColorYellow,  // yellow
	LevelError: ColorRed,     // red
}

// EventLog IDs
const (
	EventIDInfo  = 9990
	EventIDError = 9999
)

// Category is a bitmask of log topics that can be toggled via -log.
type Category uint8

const (
	CatFiltered    Category = 1 << iota // filtered_events
	CatAutoCapture                      // automatic_capture_restore
	CatSnapshot                         // snapshot_capture_restore (ON by default)
	CatWindowEvent                      // window_events
	CatTray                             // tray_interaction
	CatParking                          // window_parking
)

var categoryNames = map[string]Category{
	"filtered_events":           CatFiltered,
	"automatic_capture_restore": CatAutoCapture,
	"snapshot_capture_restore":  CatSnapshot,
	"window_events":             CatWindowEvent,
	"tray_interaction":          CatTray,
	"window_parking":            CatParking,
}

// ANSI color escape codes for per-category console output.
const (
	ColorReset   = "\033[0m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorWhite   = "\033[37m"
	ColorGray    = "\033[90m"
)

// categoryDisplay maps each Category to its display name and ANSI color.
type categoryDisplay struct {
	Name  string
	Color string
}

var categoryDisplayMap = map[Category]categoryDisplay{
	CatFiltered:    {"Filtered", ColorGray},
	CatAutoCapture: {"AutoCapture", ColorCyan},
	CatSnapshot:    {"Snapshot", ColorGreen},
	CatWindowEvent: {"WindowEvent", ColorYellow},
	CatTray:        {"Tray", ColorMagenta},
	CatParking:     {"Parking", ColorBlue},
}

var (
	mu                sync.Mutex
	el                *eventlog.Log
	silent            bool
	debugLog          *log.Logger
	enabledCategories Category

	// CurrentLevel is the minimum severity emitted. Messages with a
	// lower level are filtered before formatting or I/O. Set directly
	// after parsing the -log_level flag.
	CurrentLevel Level = LevelTrace
)

// enableVirtualTerminal enables ANSI/VT escape code processing on the
// Windows console so color codes render correctly. No-op on older Windows.
func enableVirtualTerminal() {
	handle, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE)
	if err != nil {
		return
	}
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return
	}
	mode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	windows.SetConsoleMode(handle, mode)
}

// Init initializes the Windows EventLog logger and enables ANSI
// color support on the console.
func Init(serviceName string) error {
	enableVirtualTerminal()

	var err error
	el, err = eventlog.Open(serviceName)
	if err != nil {
		// Fall back to file-based logging
		f, ferr := os.OpenFile("madori.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if ferr == nil {
			debugLog = log.New(f, "", 0)
		}
		return err
	}
	return nil
}

// Close closes the event log handle.
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if el != nil {
		el.Close()
		el = nil
	}
}

// SetSilent enables or disables silent mode (no event log output).
func SetSilent(s bool) {
	silent = s
}

// SetCategories parses a comma-separated list of category names and enables
// only those categories. An empty string resets to defaults (only CatSnapshot).
// Invalid names are silently ignored.
func SetCategories(csv string) {
	enabledCategories = 0
	if csv == "" {
		enabledCategories = CatSnapshot
		return
	}
	for _, name := range strings.Split(csv, ",") {
		name = strings.TrimSpace(name)
		if name == "all" {
			enabledCategories = ^Category(0)
			return
		}
		if cat, ok := categoryNames[name]; ok {
			enabledCategories |= cat
		}
	}
}

// DisableCategories parses a comma-separated list of category names and
// clears the corresponding bits from the enabled set. Call after
// SetCategories to subtract specific categories (e.g. after "-log all").
// Invalid names are silently ignored.
func DisableCategories(csv string) {
	for _, name := range strings.Split(csv, ",") {
		name = strings.TrimSpace(name)
		if cat, ok := categoryNames[name]; ok {
			enabledCategories &^= cat
		}
	}
}

// enabled returns true if the given category is enabled for logging.
func (c Category) enabled() bool {
	return enabledCategories&c != 0
}

// logCategory formats and writes a message with a category label, an optional
// event prefix, and color. All category-gated log functions funnel through this.
func logCategory(level Level, cat Category, event, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	disp := categoryDisplayMap[cat]
	writeLog(level, disp.Name, disp.Color, EventIDInfo, event, msg)
}

// Info logs an informational message (always visible, no category gate).
// Pass event="" to omit the event prefix.
func Info(level Level, event, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	writeLog(level, "Info", ColorWhite, EventIDInfo, event, msg)
}

// Error logs an error message.
// Pass event="" to omit the event prefix.
func Error(level Level, event, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	writeLog(level, "Error", ColorRed, EventIDError, event, msg)
}

// Event logs a significant event (same as Info but semantically different).
// Pass event="" to omit the event prefix.
func Event(level Level, event, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	writeLog(level, "Event", ColorWhite, EventIDInfo, event, msg)
}

// Trace logs a debug/trace message.
// Pass event="" to omit the event prefix.
func Trace(level Level, event, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	writeLog(level, "Trace", ColorGray, EventIDInfo, event, msg)
}

// Filtered logs messages about windows that were examined but not tracked
// (wrong class, cloaked, too small, overlay, ghost frame, etc).
// Category: filtered_events (default OFF).
// Pass event="" to omit the event prefix.
func Filtered(level Level, event, format string, args ...interface{}) {
	if !CatFiltered.enabled() {
		return
	}
	logCategory(level, CatFiltered, event, format, args...)
}

// AutoCapture logs automatic capture/restore cycle messages: capture timer,
// session lock/unlock, sleep/resume, auto-save, restore lifecycle.
// Category: automatic_capture_restore (default OFF).
// Pass event="" to omit the event prefix.
func AutoCapture(level Level, event, format string, args ...interface{}) {
	if !CatAutoCapture.enabled() {
		return
	}
	logCategory(level, CatAutoCapture, event, format, args...)
}

// Snapshot logs user-initiated snapshot capture/restore, individual window
// restores, off-screen fixes, minimize/restore-to-tray.
// Category: snapshot_capture_restore (default ON).
// Pass event="" to omit the event prefix.
func Snapshot(level Level, event, format string, args ...interface{}) {
	if !CatSnapshot.enabled() {
		return
	}
	logCategory(level, CatSnapshot, event, format, args...)
}

// WindowEvent logs non-filtered window lifecycle events: create, show, hide,
// kill, foreground change, move-size start/end, minimize start/end.
// Category: window_events (default OFF).
// Pass event="" to omit the event prefix.
func WindowEvent(level Level, event, format string, args ...interface{}) {
	if !CatWindowEvent.enabled() {
		return
	}
	logCategory(level, CatWindowEvent, event, format, args...)
}

// Tray logs system-tray interaction messages: clicks, double-clicks, timers,
// startup sequence (RegisterClassEx, CreateWindowEx, etc.).
// Category: tray_interaction (default OFF).
// Pass event="" to omit the event prefix.
func Tray(level Level, event, format string, args ...interface{}) {
	if !CatTray.enabled() {
		return
	}
	logCategory(level, CatTray, event, format, args...)
}

// Parking logs window-parking (minimize-to-tray) messages: Shift+minimize
// interception, park/unpark, parked tray icon management.
// Category: window_parking (default OFF).
// Pass event="" to omit the event prefix.
func Parking(level Level, event, format string, args ...interface{}) {
	if !CatParking.enabled() {
		return
	}
	logCategory(level, CatParking, event, format, args...)
}

// writeLog is the single point through which all console and event-log output
// flows. When event is non-empty it is formatted as "event --" and placed
// between the category label and the message.
func writeLog(level Level, categoryName, catColor string, eventID uint32, event, msg string) {
	if level < CurrentLevel {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	// Console / file output: "L HH:MM:SS [Category] event : message"
	ts := time.Now().Format("15:04:05")
	lvlPrefix := fmt.Sprintf("%s%s%s", levelColor[level], levelChar[level], ColorReset)
	var formatted string
	if event != "" {
		formatted = fmt.Sprintf("%s %s %s[%s]%s %s : %s",
			lvlPrefix, ts, catColor, categoryName, ColorReset, event, msg)
		// Avoid trailing strangeness due to component formatting
		formatted = strings.TrimSpace(formatted)
		formatted = strings.TrimSuffix(formatted, " :")
	} else {
		formatted = fmt.Sprintf("%s %s %s[%s]%s %s",
			lvlPrefix, ts, catColor, categoryName, ColorReset, msg)
	}

	if debugLog != nil {
		debugLog.Println(formatted)
	} else if !silent {
		fmt.Println(formatted)
	}

	if silent {
		return
	}

	if el != nil {
		plain := msg
		if event != "" {
			plain = fmt.Sprintf("%s : %s", event, msg)
		}
		switch eventID {
		case EventIDError:
			el.Error(eventID, fmt.Sprintf("[%s] %s", categoryName, plain))
		default:
			el.Info(eventID, fmt.Sprintf("[%s] %s", categoryName, plain))
		}
	}
}
