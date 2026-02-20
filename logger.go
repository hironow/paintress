package paintress

import (
	"fmt"
	"os"
	"sync"
	"time"
)

const (
	ColorReset  = "\033[0m"
	ColorCyan   = "\033[36m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorRed    = "\033[31m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
)

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
	line := fmt.Sprintf("[%s] %s%s%s %s", ts, color, prefix, ColorReset, msg)
	if os.Getenv("PAINTRESS_QUIET") == "" {
		fmt.Println(line)
	}

	logMu.Lock()
	defer logMu.Unlock()
	if logFile != nil {
		fmt.Fprintf(logFile, "[%s] %s %s\n", ts, prefix, msg)
	}
}

func LogInfo(format string, args ...any) {
	logLine("INFO", ColorCyan, format, args...)
}

func LogOK(format string, args ...any) {
	logLine(" OK ", ColorGreen, format, args...)
}

func LogWarn(format string, args ...any) {
	logLine("WARN", ColorYellow, format, args...)
}

func LogError(format string, args ...any) {
	logLine(" ERR", ColorRed, format, args...)
}

func LogQA(format string, args ...any) {
	logLine(" QA ", ColorPurple, format, args...)
}

func LogExp(format string, args ...any) {
	logLine(" EXP", ColorBlue, format, args...)
}
