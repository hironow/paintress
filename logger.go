package paintress

import (
	"fmt"
	"os"
	"sync"
	"time"

	"golang.org/x/term"
)

const (
	colorReset  = "\033[0m"
	colorCyan   = "\033[36m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
)

// Exported color accessors — return ANSI codes only when stderr is a TTY.
var (
	ColorReset  = colorCode(colorReset)
	ColorCyan   = colorCode(colorCyan)
	ColorGreen  = colorCode(colorGreen)
	ColorYellow = colorCode(colorYellow)
	ColorRed    = colorCode(colorRed)
	ColorBlue   = colorCode(colorBlue)
	ColorPurple = colorCode(colorPurple)
)

func colorCode(code string) string {
	if term.IsTerminal(int(os.Stderr.Fd())) {
		return code
	}
	return ""
}

var (
	logMu   sync.Mutex
	logFile *os.File
)

func InitLogFile(path string) error {
	logMu.Lock()
	defer logMu.Unlock()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	logFile = f
	return nil
}

func CloseLogFile() {
	logMu.Lock()
	defer logMu.Unlock()
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

func logLine(prefix, color, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	ts := time.Now().Format("15:04:05")

	if os.Getenv("PAINTRESS_QUIET") == "" {
		// TTY-aware: use color only when stderr is a terminal
		if term.IsTerminal(int(os.Stderr.Fd())) {
			fmt.Fprintf(os.Stderr, "[%s] %s%s%s %s\n", ts, color, prefix, colorReset, msg)
		} else {
			fmt.Fprintf(os.Stderr, "[%s] %s %s\n", ts, prefix, msg)
		}
	}

	logMu.Lock()
	defer logMu.Unlock()
	if logFile != nil {
		fmt.Fprintf(logFile, "[%s] %s %s\n", ts, prefix, msg)
	}
}

func LogInfo(format string, args ...any) {
	logLine("INFO", colorCyan, format, args...)
}

func LogOK(format string, args ...any) {
	logLine(" OK ", colorGreen, format, args...)
}

func LogWarn(format string, args ...any) {
	logLine("WARN", colorYellow, format, args...)
}

func LogError(format string, args ...any) {
	logLine(" ERR", colorRed, format, args...)
}

func LogQA(format string, args ...any) {
	logLine(" QA ", colorPurple, format, args...)
}

func LogExp(format string, args ...any) {
	logLine(" EXP", colorBlue, format, args...)
}
