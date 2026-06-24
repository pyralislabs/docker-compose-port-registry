package config

import (
	"testing"

	"github.com/pyralis-labs/compose-port-registry/internal/model"
)

func TestValidateEmptyBackupSuffix(t *testing.T) {
	cfg := &Config{
		Roots:        []string{"."},
		Format:       "text",
		Range:        PortRange{Start: 4000, End: 4999},
		BackupSuffix: "",
	}
	if err := Validate(cfg); err == nil {
		t.Error("expected error for empty backup suffix")
	}
}

func TestValidateDryRunRequiresFix(t *testing.T) {
	cfg := &Config{
		Roots:        []string{"."},
		Format:       "text",
		Range:        PortRange{Start: 4000, End: 4999},
		BackupSuffix: ".bak",
		DryRun:       true,
		Fix:          false,
	}
	if err := Validate(cfg); err == nil {
		t.Error("expected error when --dry-run without --fix")
	}
}

func TestValidateDryRunIncompatibleWithNoBackup(t *testing.T) {
	cfg := &Config{
		Roots:        []string{"."},
		Format:       "text",
		Range:        PortRange{Start: 4000, End: 4999},
		BackupSuffix: ".bak",
		DryRun:       true,
		Fix:          true,
		NoBackup:     true,
		Yes:          true,
	}
	if err := Validate(cfg); err == nil {
		t.Error("expected error for --dry-run + --no-backup combination")
	}
}

func TestValidateZeroPortRange(t *testing.T) {
	cfg := &Config{
		Roots:        []string{"."},
		Format:       "text",
		Range:        PortRange{Start: 0, End: 4999},
		BackupSuffix: ".bak",
	}
	if err := Validate(cfg); err == nil {
		t.Error("expected error for zero port range start")
	}
}

func TestParsePortRangeMaxBoundary(t *testing.T) {
	pr, err := ParsePortRange("65535-65535")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pr.Start != 65535 || pr.End != 65535 {
		t.Errorf("expected 65535-65535, got %d-%d", pr.Start, pr.End)
	}
}

func TestParsePortRangeInvalidStart(t *testing.T) {
	_, err := ParsePortRange("abc-100")
	if err == nil {
		t.Error("expected error for non-numeric start")
	}
}

func TestParsePortRangeInvalidEnd(t *testing.T) {
	_, err := ParsePortRange("100-xyz")
	if err == nil {
		t.Error("expected error for non-numeric end")
	}
}

func TestMapExitCodeDefault(t *testing.T) {
	if got := MapExitCode(model.ErrorType(999)); got != 5 {
		t.Errorf("expected default exit code 5, got %d", got)
	}
}

func TestMapExitCodeAllErrors(t *testing.T) {
	tests := []struct {
		errType model.ErrorType
		want    int
	}{
		{model.ErrInvalidConfig, 2},
		{model.ErrDiscovery, 3},
		{model.ErrLoad, 3},
		{model.ErrUnsupported, 3},
		{model.ErrIndeterminate, 3},
		{model.ErrCollision, 1},
		{model.ErrAllocationExhausted, 1},
		{model.ErrFixRefused, 4},
		{model.ErrTransaction, 4},
		{model.ErrInternal, 5},
	}
	for _, tt := range tests {
		if got := MapExitCode(tt.errType); got != tt.want {
			t.Errorf("MapExitCode(%v) = %d, want %d", tt.errType, got, tt.want)
		}
	}
}
