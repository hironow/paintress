package session

// white-box-reason: bridge constructor: exposes unexported symbols for external test packages

// export_test.go exposes unexported symbols for external tests (package session_test).
// This is a standard Go pattern used by the stdlib (e.g., net/http/export_test.go).

import (
	"context"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

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

// ExportCheckGitRepo wraps checkGitRepo with a background context for tests.
func ExportCheckGitRepo(continent string) domain.DoctorCheck {
	return checkGitRepo(context.Background(), continent)
}

// ExportCheckGitRemote wraps checkGitRemote with a background context for tests.
func ExportCheckGitRemote(continent string) domain.DoctorCheck {
	return checkGitRemote(context.Background(), continent)
}

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

// ExportSetMaxWaitDuration overrides maxWaitDuration for external tests.
func ExportSetMaxWaitDuration(d time.Duration) func() {
	old := maxWaitDuration
	maxWaitDuration = d
	return func() { maxWaitDuration = old }
}

// ExportNewLocalGitExecutor returns a localGitExecutor as port.GitExecutor for external tests.
func ExportNewLocalGitExecutor() port.GitExecutor {
	return &localGitExecutor{}
}

// ExportInitGitRepoForWorktreeWithCommit wraps initGitRepoForWorktreeWithCommit for external tests.
func ExportInitGitRepoForWorktreeWithCommit(t *testing.T) string {
	return initGitRepoForWorktreeWithCommit(t)
}

// ExportCorrectionMetadataForReport wraps correctionMetadataForReport for external tests.
func ExportCorrectionMetadataForReport(report *domain.ExpeditionReport, expedition *Expedition) domain.CorrectionMetadata {
	return correctionMetadataForReport(report, expedition)
}

// ExportAnnotateReportDMail wraps annotateReportDMail for external tests.
func ExportAnnotateReportDMail(mail *domain.DMail, report *domain.ExpeditionReport, expedition *Expedition) {
	annotateReportDMail(mail, report, expedition)
}
