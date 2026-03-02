package paintress_test

import (
	"testing"

	"github.com/hironow/paintress"
)

func TestDMailIdempotencyKey_Deterministic(t *testing.T) {
	// given
	dm := paintress.DMail{
		Name:        "report-ISSUE-42",
		Kind:        "report",
		Description: "expedition completed",
		Body:        "Details here.\n",
	}

	// when
	key1 := paintress.DMailIdempotencyKey(dm)
	key2 := paintress.DMailIdempotencyKey(dm)

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
	dm1 := paintress.DMail{
		Name:        "report-ISSUE-42",
		Kind:        "report",
		Description: "expedition completed",
		Body:        "v1\n",
	}
	dm2 := paintress.DMail{
		Name:        "report-ISSUE-42",
		Kind:        "report",
		Description: "expedition completed",
		Body:        "v2\n",
	}

	// when
	key1 := paintress.DMailIdempotencyKey(dm1)
	key2 := paintress.DMailIdempotencyKey(dm2)

	// then
	if key1 == key2 {
		t.Error("different content should produce different keys")
	}
}

func TestDMailMarshal_IdempotencyKey(t *testing.T) {
	// given
	dm := paintress.DMail{
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
	parsed, err := paintress.ParseDMail(data)
	if err != nil {
		t.Fatalf("ParseDMail: %v", err)
	}
	key, ok := parsed.Metadata["idempotency_key"]
	if !ok {
		t.Fatal("expected idempotency_key in metadata")
	}
	expected := paintress.DMailIdempotencyKey(dm)
	if key != expected {
		t.Errorf("got %q, want %q", key, expected)
	}
}

func TestValidateDMail(t *testing.T) {
	tests := []struct {
		name    string
		dmail   paintress.DMail
		wantErr bool
	}{
		{
			name: "valid dmail",
			dmail: paintress.DMail{
				SchemaVersion: paintress.DMailSchemaVersion,
				Name:          "report-001",
				Kind:          "report",
				Description:   "test",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			dmail: paintress.DMail{
				SchemaVersion: paintress.DMailSchemaVersion,
				Kind:          "report",
				Description:   "test",
			},
			wantErr: true,
		},
		{
			name: "missing kind",
			dmail: paintress.DMail{
				SchemaVersion: paintress.DMailSchemaVersion,
				Name:          "report-001",
				Description:   "test",
			},
			wantErr: true,
		},
		{
			name: "missing description",
			dmail: paintress.DMail{
				SchemaVersion: paintress.DMailSchemaVersion,
				Name:          "report-001",
				Kind:          "report",
			},
			wantErr: true,
		},
		{
			name: "missing schema version",
			dmail: paintress.DMail{
				Name:        "report-001",
				Kind:        "report",
				Description: "test",
			},
			wantErr: true,
		},
		{
			name: "wrong schema version",
			dmail: paintress.DMail{
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
			err := paintress.ValidateDMail(tt.dmail)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDMail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDMailMarshal_ActionFieldRoundTrip(t *testing.T) {
	// given
	dm := paintress.DMail{
		Name:          "task-ISSUE-99",
		Kind:          "specification",
		Description:   "implement login feature",
		SchemaVersion: paintress.DMailSchemaVersion,
		Action:        "implement",
	}

	// when
	data, err := dm.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	parsed, err := paintress.ParseDMail(data)
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
	dm := paintress.DMail{
		Name:          "task-ISSUE-100",
		Kind:          "specification",
		Description:   "fix critical bug",
		SchemaVersion: paintress.DMailSchemaVersion,
		Priority:      3,
	}

	// when
	data, err := dm.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	parsed, err := paintress.ParseDMail(data)
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
	dm := paintress.DMail{
		Name:          "report-ISSUE-50",
		Kind:          "report",
		Description:   "simple report",
		SchemaVersion: paintress.DMailSchemaVersion,
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

func TestDMailMarshal_IdempotencyKey_PreservesExistingMetadata(t *testing.T) {
	// given
	dm := paintress.DMail{
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
	parsed, err := paintress.ParseDMail(data)
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
