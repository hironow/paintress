package verifier_test

import (
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/harness/verifier"
)

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
			err := verifier.ValidateDMail(tt.dmail)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDMail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
