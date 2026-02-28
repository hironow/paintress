package session

import (
	"runtime"
	"strings"
)

// shellQuoteUnix wraps a string in single quotes with proper escaping for sh.
func shellQuoteUnix(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// shellQuoteCmd wraps a string in double quotes with proper escaping for cmd.exe.
func shellQuoteCmd(s string) string {
	s = strings.ReplaceAll(s, `"`, `""`)
	s = strings.ReplaceAll(s, `%`, `%%`)
	return `"` + s + `"`
}

// shellQuote quotes s for safe interpolation into shell commands.
func shellQuote(s string) string {
	if runtime.GOOS == "windows" {
		return shellQuoteCmd(s)
	}
	return shellQuoteUnix(s)
}
