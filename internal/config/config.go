package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pyralis-labs/compose-port-registry/internal/model"
)

type Config struct {
	Roots        []string
	Files        []string
	ProjectDir   string
	EnvFiles     []string
	Profiles     []string
	Range        PortRange
	Excludes     []string
	Format       string
	Suggest      bool
	Fix          bool
	DryRun       bool
	BackupSuffix string
	NoBackup     bool
	Yes          bool
	Strict       bool
	ShowVersion  bool
}

type PortRange struct {
	Start uint16
	End   uint16
}

func Validate(cfg *Config) error {
	if cfg.Format != "text" && cfg.Format != "json" {
		return fmt.Errorf("invalid format %q: must be text or json", cfg.Format)
	}
	if cfg.Range.Start == 0 || cfg.Range.End == 0 {
		return fmt.Errorf("port range must be between 1 and 65535")
	}
	if cfg.Range.Start > cfg.Range.End {
		return fmt.Errorf("port range start %d must not exceed end %d", cfg.Range.Start, cfg.Range.End)
	}
	if len(cfg.Roots) == 0 && len(cfg.Files) == 0 {
		return fmt.Errorf("at least one root path or --file must be specified")
	}
	if cfg.Fix && cfg.DryRun && !cfg.Fix {
		return fmt.Errorf("--dry-run requires --fix")
	}
	if cfg.NoBackup && !cfg.Yes {
		return fmt.Errorf("--no-backup requires --yes")
	}
	if cfg.DryRun && cfg.NoBackup {
		return fmt.Errorf("--dry-run and --no-backup are incompatible")
	}
	if cfg.BackupSuffix == "" {
		return fmt.Errorf("--backup-suffix must not be empty")
	}
	if len(cfg.Files) > 0 && len(cfg.Roots) > 0 {
		return fmt.Errorf("--file and root paths cannot be combined")
	}
	return nil
}

func DefaultConfig() *Config {
	return &Config{
		Range:        PortRange{Start: 4000, End: 4999},
		Format:       "text",
		BackupSuffix: ".port-registry.bak",
		Yes:          false,
	}
}

func ParsePortRange(s string) (PortRange, error) {
	s = strings.TrimSpace(s)
	parts := strings.SplitN(s, "-", 2)
	if len(parts) != 2 {
		return PortRange{}, fmt.Errorf("port range must be in START-END format (e.g., 4000-4999)")
	}
	start, err := strconv.ParseUint(parts[0], 10, 16)
	if err != nil {
		return PortRange{}, fmt.Errorf("invalid port range start %q", parts[0])
	}
	end, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return PortRange{}, fmt.Errorf("invalid port range end %q", parts[1])
	}
	pr := PortRange{Start: uint16(start), End: uint16(end)}
	if pr.Start == 0 || pr.End == 0 {
		return PortRange{}, fmt.Errorf("port range values must be 1-65535")
	}
	if pr.Start > pr.End {
		return PortRange{}, fmt.Errorf("port range start %d must not exceed end %d", pr.Start, pr.End)
	}
	return pr, nil
}

func MapExitCode(errType model.ErrorType) int {
	switch errType {
	case model.ErrInvalidConfig:
		return 2
	case model.ErrDiscovery:
		return 3
	case model.ErrLoad:
		return 3
	case model.ErrUnsupported:
		return 3
	case model.ErrIndeterminate:
		return 3
	case model.ErrCollision:
		return 1
	case model.ErrAllocationExhausted:
		return 1
	case model.ErrFixRefused:
		return 4
	case model.ErrTransaction:
		return 4
	case model.ErrInternal:
		return 5
	default:
		return 5
	}
}
