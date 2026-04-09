package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

func TestCheckStatus_StatusLabel_AllKnown(t *testing.T) {
	known := map[domain.CheckStatus]string{
		domain.CheckOK:    "OK",
		domain.CheckFail:  "FAIL",
		domain.CheckSkip:  "SKIP",
		domain.CheckWarn:  "WARN",
		domain.CheckFixed: "FIX",
	}
	for status, want := range known {
		if got := status.StatusLabel(); got != want {
			t.Errorf("StatusLabel(%d) = %q, want %q", status, got, want)
		}
	}
}

func TestCheckStatus_StatusLabel_Unknown_IsNotOK(t *testing.T) {
	// RED-first: this test proves that unknown status does NOT silently
	// become "OK" (fail-open). GAP-ARCH-048 requires fail-closed behavior.
	unknown := domain.CheckStatus(99)
	label := unknown.StatusLabel()
	if label == "OK" {
		t.Error("unknown status must not map to OK (fail-open)")
	}
	if label != "????" {
		t.Errorf("unknown status label = %q, want %q", label, "????")
	}
}

func TestCheckStatus_JSONUnmarshal_UnknownLabelMustError(t *testing.T) {
	// CheckStatus has no custom UnmarshalJSON (transport removed in GAP-048).
	// Unknown string labels MUST produce an unmarshal error — they must never
	// be silently converted to any known status (CheckOK, CheckWarn, etc.).
	unknownLabels := []string{`"UNKNOWN"`, `"INVALID"`, `""`, `"ok"`, `"fail"`}
	for _, raw := range unknownLabels {
		var status domain.CheckStatus
		err := json.Unmarshal([]byte(raw), &status)
		if err == nil {
			t.Errorf("json.Unmarshal(%s) should error (no custom unmarshaler), but got status=%d", raw, status)
		}
	}
}

func TestCheckStatus_JSONMarshal_UsesInteger(t *testing.T) {
	// Without custom MarshalJSON, CheckStatus serializes as its integer value.
	// This ensures domain has no transport responsibility.
	data, err := json.Marshal(domain.CheckFail)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "1" {
		t.Errorf("json.Marshal(CheckFail) = %s, want \"1\" (integer, no custom MarshalJSON)", data)
	}
}
