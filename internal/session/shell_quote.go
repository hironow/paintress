package session

import (
	"runtime"
	"strings"
)

// ShellQuoteUnix wraps a string in single quotes with proper escaping for sh.
func ShellQuoteUnix(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// ShellQuoteCmd wraps a string in double quotes with proper escaping for cmd.exe.
func ShellQuoteCmd(s string) string {
	s = strings.ReplaceAll(s, `"`, `""`)
	s = strings.ReplaceAll(s, `%`, `%%`)
	return `"` + s + `"`
}

// ShellQuote quotes s for safe interpolation into shell commands.
func ShellQuote(s string) string {
	if runtime.GOOS == "windows" {
		return ShellQuoteCmd(s)
	}
	return ShellQuoteUnix(s)
}
