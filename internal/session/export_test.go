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
var ExportCheckGitRemote = checkGitRemote
var ExportCheckWritability = checkWritability
var ExportCheckSkills = checkSkills
var ExportCheckEventStore = checkEventStore
var ExportCheckClaudeInference = checkClaudeInference
var ExportCheckGHScopes = checkGHScopes
var ExportCheckContextBudget = CheckContextBudget
var ExportExtractStreamResult = ExtractStreamResult
var ExportCheckSkillsRefToolchain = checkSkillsRefToolchain

// NewCmdApproverForTest creates a CmdApprover with a test command factory.
func NewCmdApproverForTest(cmdTemplate string, factory cmdFactoryFunc) *CmdApprover {
	return &CmdApprover{cmdTemplate: cmdTemplate, cmdFactory: factory}
}

// NewLocalNotifierForTest creates a LocalNotifier with test overrides.
func NewLocalNotifierForTest(osName string, factory cmdFactoryFunc) *LocalNotifier {
	return &LocalNotifier{forceOS: osName, cmdFactory: factory}
}

// NewCmdNotifierForTest creates a CmdNotifier with a test command factory.
func NewCmdNotifierForTest(cmdTemplate string, factory cmdFactoryFunc) *CmdNotifier {
	return &CmdNotifier{cmdTemplate: cmdTemplate, cmdFactory: factory}
}

// ExportBuildIsolationFlags exposes buildIsolationFlags for contract testing.
func ExportBuildIsolationFlags(cfg EnterConfig) []string { return buildIsolationFlags(cfg) }
