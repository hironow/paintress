package domain_test

import (
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

func TestDMailIdempotencyKey_Deterministic(t *testing.T) {
	// given
	dm := domain.DMail{
		Name:        "report-ISSUE-42",
		Kind:        "report",
		Description: "expedition completed",
		Body:        "Details here.\n",
	}

	// when
	key1 := domain.DMailIdempotencyKey(dm)
	key2 := domain.DMailIdempotencyKey(dm)

	// then
	if key1 != key2 {
		t.Errorf("not deterministic: %q != %q", key1, key2)
	}
	if len(key1) != 64 {
		t.Errorf("expected 64-char hex, got %d: %q", len(key1), key1)
	}
}

func TestDMailIdempotencyKey_DifferentContent(t *testing.T) {
	// given
	dm1 := domain.DMail{
		Name:        "report-ISSUE-42",
		Kind:        "report",
		Description: "expedition completed",
		Body:        "v1\n",
	}
	dm2 := domain.DMail{
		Name:        "report-ISSUE-42",
		Kind:        "report",
		Description: "expedition completed",
		Body:        "v2\n",
	}

	// when
	key1 := domain.DMailIdempotencyKey(dm1)
	key2 := domain.DMailIdempotencyKey(dm2)

	// then
	if key1 == key2 {
		t.Error("different content should produce different keys")
	}
}

func TestDMailMarshal_IdempotencyKey(t *testing.T) {
	// given
	dm := domain.DMail{
		Name:        "report-ISSUE-42",
		Kind:        "report",
		Description: "expedition completed",
		Body:        "Details here.\n",
	}

	// when
	data, err := dm.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	// then
	parsed, err := domain.ParseDMail(data)
	if err != nil {
		t.Fatalf("ParseDMail: %v", err)
	}
	key, ok := parsed.Metadata["idempotency_key"]
	if !ok {
		t.Fatal("expected idempotency_key in metadata")
	}
	expected := domain.DMailIdempotencyKey(dm)
	if key != expected {
		t.Errorf("got %q, want %q", key, expected)
	}
}

func TestValidateDMail(t *testing.T) {
	tests := []struct {
		name    string
		dmail   domain.DMail
		wantErr bool
	}{
		{
			name: "valid dmail",
			dmail: domain.DMail{
				SchemaVersion: domain.DMailSchemaVersion,
				Name:          "report-001",
				Kind:          "report",
				Description:   "test",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			dmail: domain.DMail{
				SchemaVersion: domain.DMailSchemaVersion,
				Kind:          "report",
				Description:   "test",
			},
			wantErr: true,
		},
		{
			name: "missing kind",
			dmail: domain.DMail{
				SchemaVersion: domain.DMailSchemaVersion,
				Name:          "report-001",
				Description:   "test",
			},
			wantErr: true,
		},
		{
			name: "missing description",
			dmail: domain.DMail{
				SchemaVersion: domain.DMailSchemaVersion,
				Name:          "report-001",
				Kind:          "report",
			},
			wantErr: true,
		},
		{
			name: "missing schema version",
			dmail: domain.DMail{
				Name:        "report-001",
				Kind:        "report",
				Description: "test",
			},
			wantErr: true,
		},
		{
			name: "wrong schema version",
			dmail: domain.DMail{
				SchemaVersion: "99",
				Name:          "report-001",
				Kind:          "report",
				Description:   "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := domain.ValidateDMail(tt.dmail)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDMail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDMailMarshal_ActionFieldRoundTrip(t *testing.T) {
	// given
	dm := domain.DMail{
		Name:          "task-ISSUE-99",
		Kind:          "specification",
		Description:   "implement login feature",
		SchemaVersion: domain.DMailSchemaVersion,
		Action:        "implement",
	}

	// when
	data, err := dm.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	parsed, err := domain.ParseDMail(data)
	if err != nil {
		t.Fatalf("ParseDMail: %v", err)
	}

	// then
	if parsed.Action != "implement" {
		t.Errorf("Action round-trip: got %q, want %q", parsed.Action, "implement")
	}
}

func TestDMailMarshal_PriorityFieldRoundTrip(t *testing.T) {
	// given
	dm := domain.DMail{
		Name:          "task-ISSUE-100",
		Kind:          "specification",
		Description:   "fix critical bug",
		SchemaVersion: domain.DMailSchemaVersion,
		Priority:      3,
	}

	// when
	data, err := dm.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	parsed, err := domain.ParseDMail(data)
	if err != nil {
		t.Fatalf("ParseDMail: %v", err)
	}

	// then
	if parsed.Priority != 3 {
		t.Errorf("Priority round-trip: got %d, want %d", parsed.Priority, 3)
	}
}

func TestDMailMarshal_OmitEmptyActionAndPriority(t *testing.T) {
	// given — DMail with zero-value action and priority
	dm := domain.DMail{
		Name:          "report-ISSUE-50",
		Kind:          "report",
		Description:   "simple report",
		SchemaVersion: domain.DMailSchemaVersion,
	}

	// when
	data, err := dm.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	// then — action and priority should not appear in output
	s := string(data)
	if contains(s, "action:") {
		t.Error("empty action should be omitted from marshalled output")
	}
	if contains(s, "priority:") {
		t.Error("zero priority should be omitted from marshalled output")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && stringContains(s, substr)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestDMailMarshal_ContextRoundTrip(t *testing.T) {
	// given
	dm := domain.DMail{
		Name:          "report-ISSUE-55",
		Kind:          "report",
		Description:   "expedition with insight context",
		SchemaVersion: domain.DMailSchemaVersion,
		Context: &domain.InsightContext{
			Insights: []domain.InsightSummary{
				{Source: "paintress", Summary: "Lumina score improved after retry"},
				{Source: "amadeus", Summary: "ADR compliance at 95%"},
			},
		},
		Body: "Details here.\n",
	}

	// when
	data, err := dm.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	parsed, err := domain.ParseDMail(data)
	if err != nil {
		t.Fatalf("ParseDMail: %v", err)
	}

	// then
	if parsed.Context == nil {
		t.Fatal("expected non-nil Context after round-trip")
	}
	if len(parsed.Context.Insights) != 2 {
		t.Fatalf("expected 2 insights, got %d", len(parsed.Context.Insights))
	}
	if parsed.Context.Insights[0].Source != "paintress" {
		t.Errorf("insight[0].Source = %q, want %q", parsed.Context.Insights[0].Source, "paintress")
	}
	if parsed.Context.Insights[0].Summary != "Lumina score improved after retry" {
		t.Errorf("insight[0].Summary = %q, want %q", parsed.Context.Insights[0].Summary, "Lumina score improved after retry")
	}
	if parsed.Context.Insights[1].Source != "amadeus" {
		t.Errorf("insight[1].Source = %q, want %q", parsed.Context.Insights[1].Source, "amadeus")
	}
}

func TestDMailMarshal_NilContextOmitted(t *testing.T) {
	// given — DMail with nil Context
	dm := domain.DMail{
		Name:          "report-ISSUE-56",
		Kind:          "report",
		Description:   "no context",
		SchemaVersion: domain.DMailSchemaVersion,
	}

	// when
	data, err := dm.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	// then — context should not appear in output
	if contains(string(data), "context:") {
		t.Error("nil Context should be omitted from marshalled output")
	}
}

func TestNewReportDMail_InsightContext_Present(t *testing.T) {
	// given — report with an insight string
	report := &domain.ExpeditionReport{
		Expedition:  42,
		IssueID:     "MY-100",
		IssueTitle:  "Fix thing",
		MissionType: "fix",
		Status:      "success",
		Insight:     "retry reduced failures by 30%",
	}

	// when
	dm := domain.NewReportDMail(report)

	// then
	if dm.Context == nil {
		t.Fatal("expected non-nil Context when Insight is present")
	}
	if len(dm.Context.Insights) == 0 {
		t.Fatal("expected at least one InsightSummary in Context")
	}
	if dm.Context.Insights[0].Summary != "retry reduced failures by 30%" {
		t.Errorf("Summary = %q, want %q", dm.Context.Insights[0].Summary, "retry reduced failures by 30%")
	}
}

func TestNewReportDMail_InsightContext_Absent(t *testing.T) {
	// given — report with no insight string
	report := &domain.ExpeditionReport{
		Expedition:  43,
		IssueID:     "MY-101",
		IssueTitle:  "Fix other thing",
		MissionType: "fix",
		Status:      "success",
	}

	// when
	dm := domain.NewReportDMail(report)

	// then — backward-compatible: nil Context when Insight is empty
	if dm.Context != nil {
		t.Errorf("expected nil Context when Insight is absent, got %+v", dm.Context)
	}
}

func TestNewReportDMail_InsightContext_NilGuard(t *testing.T) {
	// given — report pointer itself; verify NewReportDMail does not panic on empty insight
	report := &domain.ExpeditionReport{
		Expedition:  44,
		IssueID:     "MY-102",
		IssueTitle:  "Another fix",
		MissionType: "fix",
		Status:      "failed",
		Insight:     "",
	}

	// when / then — must not panic
	dm := domain.NewReportDMail(report)
	if dm.Context != nil {
		t.Errorf("expected nil Context for empty Insight, got %+v", dm.Context)
	}
}

func TestDMailMarshal_IdempotencyKey_PreservesExistingMetadata(t *testing.T) {
	// given
	dm := domain.DMail{
		Name:        "report-ISSUE-42",
		Kind:        "report",
		Description: "test",
		Metadata:    map[string]string{"source": "expedition"},
	}

	// when
	data, err := dm.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	// then
	parsed, err := domain.ParseDMail(data)
	if err != nil {
		t.Fatalf("ParseDMail: %v", err)
	}
	if parsed.Metadata["source"] != "expedition" {
		t.Errorf("existing metadata lost: %v", parsed.Metadata)
	}
	if _, ok := parsed.Metadata["idempotency_key"]; !ok {
		t.Fatal("expected idempotency_key in metadata")
	}
}
