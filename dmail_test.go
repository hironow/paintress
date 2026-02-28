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
