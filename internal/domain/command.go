package domain

// RunExpeditionCommand represents the intent to run a paintress expedition.
// Independent of cobra — framework concerns are separated at the cmd layer.
// Fields are unexported; use NewRunExpeditionCommand to construct a valid instance.
type RunExpeditionCommand struct {
	repoPath RepoPath
}

// NewRunExpeditionCommand creates a RunExpeditionCommand from validated primitives.
func NewRunExpeditionCommand(repoPath RepoPath) RunExpeditionCommand {
	return RunExpeditionCommand{repoPath: repoPath}
}

// RepoPath returns the validated repository path.
func (c RunExpeditionCommand) RepoPath() RepoPath { return c.repoPath }

// InitCommand represents the intent to initialize a paintress project.
// Fields are unexported; use NewInitCommand to construct a valid instance.
type InitCommand struct {
	repoPath RepoPath
	team     Team
	project  Project
}

// NewInitCommand creates an InitCommand from validated primitives.
func NewInitCommand(repoPath RepoPath, team Team, project Project) InitCommand {
	return InitCommand{repoPath: repoPath, team: team, project: project}
}

// RepoPath returns the validated repository path.
func (c InitCommand) RepoPath() RepoPath { return c.repoPath }

// Team returns the team identifier.
func (c InitCommand) Team() Team { return c.team }

// Project returns the project name.
func (c InitCommand) Project() Project { return c.project }

// RebuildCommand represents the intent to rebuild projections from the event store.
// Fields are unexported; use NewRebuildCommand to construct a valid instance.
type RebuildCommand struct {
	repoPath RepoPath
}

// NewRebuildCommand creates a RebuildCommand from validated primitives.
func NewRebuildCommand(repoPath RepoPath) RebuildCommand {
	return RebuildCommand{repoPath: repoPath}
}

// RepoPath returns the validated repository path.
func (c RebuildCommand) RepoPath() RepoPath { return c.repoPath }

// ArchivePruneCommand represents the intent to prune old archive files.
// Fields are unexported; use NewArchivePruneCommand to construct a valid instance.
type ArchivePruneCommand struct {
	repoPath RepoPath
	days     Days
	execute  bool
}

// NewArchivePruneCommand creates an ArchivePruneCommand from validated primitives.
func NewArchivePruneCommand(repoPath RepoPath, days Days, execute bool) ArchivePruneCommand {
	return ArchivePruneCommand{repoPath: repoPath, days: days, execute: execute}
}

// RepoPath returns the validated repository path.
func (c ArchivePruneCommand) RepoPath() RepoPath { return c.repoPath }

// Days returns the validated retention day count.
func (c ArchivePruneCommand) Days() Days { return c.days }

// Execute returns whether to actually execute the pruning.
func (c ArchivePruneCommand) Execute() bool { return c.execute }
