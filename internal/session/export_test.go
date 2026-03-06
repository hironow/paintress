package session

// white-box-reason: bridge constructor: exposes unexported symbols for external test packages

// export_test.go exposes unexported symbols for external tests (package session_test).
// This is a standard Go pattern used by the stdlib (e.g., net/http/export_test.go).

var ExportWatchFlag = watchFlag
var ExportWatchInbox = watchInbox
var ExportShellName = shellName
var ExportShellFlag = shellFlag
var ExportExtractValue = extractValue
var ExportParseKV = parseKV
var ExportReconcileFlags = reconcileFlags
var ExportCheckClaudeAuth = checkClaudeAuth
var ExportCheckLinearMCP = checkLinearMCP
var ExportCheckContinent = checkContinent
var ExportCheckConfig = checkConfig
var ExportCheckGitRepo = checkGitRepo
var ExportCheckWritability = checkWritability
var ExportCheckSkills = checkSkills
var ExportCheckEventStore = checkEventStore
