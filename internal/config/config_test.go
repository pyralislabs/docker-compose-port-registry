package config

import (
	"testing"

	"github.com/pyralis-labs/compose-port-registry/internal/model"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				Roots:        []string{"."},
				Format:       "text",
				Range:        PortRange{Start: 4000, End: 4999},
				BackupSuffix: ".bak",
			},
			wantErr: false,
		},
		{
			name: "invalid format",
			cfg: &Config{
				Roots:        []string{"."},
				Format:       "xml",
				Range:        PortRange{Start: 4000, End: 4999},
				BackupSuffix: ".bak",
			},
			wantErr: true,
		},
		{
			name: "no roots no files",
			cfg: &Config{
				Format:       "text",
				Range:        PortRange{Start: 4000, End: 4999},
				BackupSuffix: ".bak",
			},
			wantErr: true,
		},
		{
			name: "reversed range",
			cfg: &Config{
				Roots:        []string{"."},
				Format:       "text",
				Range:        PortRange{Start: 5000, End: 4000},
				BackupSuffix: ".bak",
			},
			wantErr: true,
		},
		{
			name: "no backup without yes",
			cfg: &Config{
				Roots:        []string{"."},
				Format:       "text",
				Range:        PortRange{Start: 4000, End: 4999},
				NoBackup:     true,
				BackupSuffix: ".bak",
			},
			wantErr: true,
		},
		{
			name: "file and roots combined",
			cfg: &Config{
				Roots:        []string{"."},
				Files:        []string{"compose.yaml"},
				Format:       "text",
				Range:        PortRange{Start: 4000, End: 4999},
				BackupSuffix: ".bak",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.cfg)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestParsePortRange(t *testing.T) {
	tests := []struct {
		input    string
		wantOk   bool
		wantErr  bool
		wantLow  uint16
		wantHigh uint16
	}{
		{"4000-4999", true, false, 4000, 4999},
		{"80-80", true, false, 80, 80},
		{"invalid", false, true, 0, 0},
		{"1-65535", true, false, 1, 65535},
		{"0-100", false, true, 0, 0},
		{"100-50", false, true, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			pr, err := ParsePortRange(tt.input)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantOk {
				if pr.Start != tt.wantLow || pr.End != tt.wantHigh {
					t.Errorf("got %d-%d, want %d-%d", pr.Start, pr.End, tt.wantLow, tt.wantHigh)
				}
			}
		})
	}
}

func TestMapExitCode(t *testing.T) {
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
		t.Run("", func(t *testing.T) {
			got := MapExitCode(tt.errType)
			if got != tt.want {
				t.Errorf("MapExitCode(%d) = %d, want %d", tt.errType, got, tt.want)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Range.Start != 4000 || cfg.Range.End != 4999 {
		t.Errorf("expected range 4000-4999, got %d-%d", cfg.Range.Start, cfg.Range.End)
	}
	if cfg.Format != "text" {
		t.Errorf("expected format text, got %s", cfg.Format)
	}
}
