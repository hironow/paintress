package session

// export_test.go exposes unexported symbols for external tests (package session_test).
// This is a standard Go pattern used by the stdlib (e.g., net/http/export_test.go).

var ExportWatchFlag = watchFlag
var ExportWatchInbox = watchInbox
var ExportShellName = shellName
var ExportShellFlag = shellFlag
var ExportExtractValue = extractValue
