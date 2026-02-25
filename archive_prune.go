package paintress

// PruneResult holds the outcome of an archive prune operation.
type PruneResult struct {
	Candidates []string // basenames of files older than threshold
	Deleted    int      // number of files actually removed (0 in dry-run)
}
