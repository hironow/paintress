package paintress

import "context"

// Approver requests human approval and blocks until a response or context expiry.
type Approver interface {
	RequestApproval(ctx context.Context, message string) (approved bool, err error)
}

// AutoApprover always approves without human interaction.
type AutoApprover struct{}

func (a *AutoApprover) RequestApproval(_ context.Context, _ string) (bool, error) {
	return true, nil
}
