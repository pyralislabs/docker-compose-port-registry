package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFlagsDefaults(t *testing.T) {
	cfg, version, err := parseFlags([]string{}, os.Stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != "" {
		t.Errorf("expected no version output, got %s", version)
	}
	if cfg.Format != "text" {
		t.Errorf("expected format text, got %s", cfg.Format)
	}
	if cfg.Range.Start != 4000 || cfg.Range.End != 4999 {
		t.Errorf("expected range 4000-4999, got %d-%d", cfg.Range.Start, cfg.Range.End)
	}
	if cfg.ShowVersion {
		t.Error("expected ShowVersion false by default")
	}
}

func TestParseFlagsVersion(t *testing.T) {
	cfg, version, err := parseFlags([]string{"--version"}, os.Stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.ShowVersion {
		t.Error("expected ShowVersion true")
	}
	if !strings.Contains(version, "compose-port-registry") {
		t.Errorf("expected version string, got %s", version)
	}
}

func TestParseFlagsCustomRange(t *testing.T) {
	cfg, _, err := parseFlags([]string{"--range", "5000-5999"}, os.Stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Range.Start != 5000 || cfg.Range.End != 5999 {
		t.Errorf("expected range 5000-5999, got %d-%d", cfg.Range.Start, cfg.Range.End)
	}
}

func TestParseFlagsInvalidRange(t *testing.T) {
	_, _, err := parseFlags([]string{"--range", "invalid"}, os.Stderr)
	if err == nil {
		t.Error("expected error for invalid range")
	}
}

func TestParseFlagsRepeatedFile(t *testing.T) {
	cfg, _, err := parseFlags([]string{"--file", "a.yaml", "--file", "b.yaml"}, os.Stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Files) != 2 || cfg.Files[0] != "a.yaml" || cfg.Files[1] != "b.yaml" {
		t.Errorf("expected [a.yaml, b.yaml], got %v", cfg.Files)
	}
}

func TestParseFlagsRepeatedEnvFile(t *testing.T) {
	cfg, _, err := parseFlags([]string{"--env-file", "a.env", "--env-file", "b.env"}, os.Stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.EnvFiles) != 2 {
		t.Errorf("expected 2 env files, got %d", len(cfg.EnvFiles))
	}
}

func TestParseFlagsRepeatedProfile(t *testing.T) {
	cfg, _, err := parseFlags([]string{"--profile", "dev", "--profile", "debug"}, os.Stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Profiles) != 2 || cfg.Profiles[0] != "dev" || cfg.Profiles[1] != "debug" {
		t.Errorf("expected [dev, debug], got %v", cfg.Profiles)
	}
}

func TestParseFlagsRepeatedExclude(t *testing.T) {
	cfg, _, err := parseFlags([]string{"--exclude", "vendor", "--exclude", "node_modules"}, os.Stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Excludes) != 2 {
		t.Errorf("expected 2 excludes, got %d", len(cfg.Excludes))
	}
}

func TestParseFlagsBoolFlags(t *testing.T) {
	cfg, _, err := parseFlags([]string{
		"--suggest", "--fix", "--dry-run", "--yes", "--no-backup", "--strict",
	}, os.Stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	flags := []struct {
		name string
		got  bool
	}{
		{"Suggest", cfg.Suggest},
		{"Fix", cfg.Fix},
		{"DryRun", cfg.DryRun},
		{"Yes", cfg.Yes},
		{"NoBackup", cfg.NoBackup},
		{"Strict", cfg.Strict},
	}
	for _, f := range flags {
		if !f.got {
			t.Errorf("expected %s true", f.name)
		}
	}
}

func TestParseFlagsStringFlags(t *testing.T) {
	cfg, _, err := parseFlags([]string{
		"--format", "json",
		"--backup-suffix", ".bak",
		"--project-dir", "/x",
	}, os.Stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Format != "json" {
		t.Errorf("expected format json, got %s", cfg.Format)
	}
	if cfg.BackupSuffix != ".bak" {
		t.Errorf("expected backup suffix .bak, got %s", cfg.BackupSuffix)
	}
	if cfg.ProjectDir != "/x" {
		t.Errorf("expected project dir /x, got %s", cfg.ProjectDir)
	}
}

func TestParseFlagsPositionalArgs(t *testing.T) {
	cfg, _, err := parseFlags([]string{"root1", "root2", "root3"}, os.Stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Roots) != 3 {
		t.Errorf("expected 3 roots, got %d", len(cfg.Roots))
	}
}

func TestParseFlagsUnknownFlag(t *testing.T) {
	_, _, err := parseFlags([]string{"--unknown-flag"}, os.Stderr)
	if err == nil {
		t.Error("expected error for unknown flag")
	}
}

func TestMultiFlagString(t *testing.T) {
	var m multiFlag
	m = append(m, "a", "b", "c")
	got := m.String()
	if got != "[a b c]" {
		t.Errorf("expected [a b c], got %s", got)
	}
}

func TestMultiFlagSet(t *testing.T) {
	var m multiFlag
	if err := m.Set("x"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := m.Set("y"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m) != 2 {
		t.Errorf("expected 2 values, got %d", len(m))
	}
}

func TestRunWithVersionFlag(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := run([]string{"--version"}, stdout, stderr)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "compose-port-registry") {
		t.Errorf("expected version in stdout: %s", stdout.String())
	}
}

func TestRunWithInvalidRange(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := run([]string{"--range", "bad", "/tmp"}, stdout, stderr)
	if code != 2 {
		t.Errorf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "invalid --range") {
		t.Errorf("expected invalid range error: %s", stderr.String())
	}
}

func TestRunWithValidPath(t *testing.T) {
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	os.WriteFile(composeFile, []byte("services:\n  web:\n    image: nginx\n    ports:\n      - \"8080:80\"\n"), 0644)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := run([]string{"--format", "json", tmpDir}, stdout, stderr)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d\nstderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "schema_version") {
		t.Errorf("expected JSON output: %s", stdout.String())
	}
}

func TestRunWithUnknownFlag(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := run([]string{"--unknown"}, stdout, stderr)
	if code != 2 {
		t.Errorf("expected exit code 2, got %d", code)
	}
}
