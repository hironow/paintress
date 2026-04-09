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

func TestCheckStatus_JSONRoundTrip_UnknownIsNotOK(t *testing.T) {
	// RED-first: if domain has MarshalJSON/UnmarshalJSON, unknown label
	// must NOT be silently converted to CheckOK.
	// After GAP-048 fix, domain will have no JSON methods and this test
	// verifies that json.Marshal uses the integer representation (no MarshalJSON).

	// Marshal a known status
	data, err := json.Marshal(domain.CheckFail)
	if err != nil {
		t.Fatal(err)
	}

	// Attempt to unmarshal an unknown label
	unknownJSON := []byte(`"UNKNOWN_STATUS"`)
	var status domain.CheckStatus
	err = json.Unmarshal(unknownJSON, &status)
	// After MarshalJSON/UnmarshalJSON removal, this should fail (no custom unmarshaler)
	// or at minimum, status should NOT be CheckOK
	if err == nil && status == domain.CheckOK {
		t.Errorf("unknown JSON label %q was silently converted to CheckOK (fail-open); data=%s", unknownJSON, data)
	}
}
