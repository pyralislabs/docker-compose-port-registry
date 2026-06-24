package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pyralis-labs/compose-port-registry/internal/config"
)

func TestRunRollbackOnFixFailure(t *testing.T) {
	tmpDir := t.TempDir()
	dirA := filepath.Join(tmpDir, "a")
	dirB := filepath.Join(tmpDir, "b")
	if err := os.MkdirAll(dirA, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dirB, 0755); err != nil {
		t.Fatal(err)
	}

	writeYAML(t, filepath.Join(dirA, "compose.yaml"), `services:
  api:
    image: nginx
    ports:
      - "8080:80"
`)
	writeYAML(t, filepath.Join(dirB, "compose.yaml"), `services:
  web:
    image: nginx
    ports:
      - "8080:80"
`)

	if err := os.Chmod(filepath.Join(dirB, "compose.yaml"), 0444); err != nil {
		t.Skipf("can't make readonly file: %v", err)
	}
	defer os.Chmod(filepath.Join(dirB, "compose.yaml"), 0644)

	cfg := config.DefaultConfig()
	cfg.Roots = []string{tmpDir}
	cfg.Format = "text"
	cfg.Fix = true
	cfg.Yes = true

	a, _, _ := newTestApp(t, cfg)
	code := a.Run(context.Background())
	if code != 4 {
		t.Errorf("expected exit code 4 (rollback), got %d", code)
	}
}

func TestRunNoBackupRequiresYes(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Roots = []string{"/tmp"}
	cfg.NoBackup = true
	cfg.Format = "text"

	a, _, _ := newTestApp(t, cfg)
	code := a.Run(context.Background())
	if code != 2 {
		t.Errorf("expected exit code 2 for --no-backup without --yes, got %d", code)
	}
}

func TestRunDryRunWithoutFix(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Roots = []string{"/tmp"}
	cfg.DryRun = true
	cfg.Format = "text"

	a, _, _ := newTestApp(t, cfg)
	code := a.Run(context.Background())
	if code != 2 {
		t.Errorf("expected exit code 2 for --dry-run without --fix, got %d", code)
	}
}

func TestRunFixDryRunCollision(t *testing.T) {
	tmpDir := t.TempDir()
	dirA := filepath.Join(tmpDir, "a")
	dirB := filepath.Join(tmpDir, "b")
	if err := os.MkdirAll(dirA, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dirB, 0755); err != nil {
		t.Fatal(err)
	}

	writeYAML(t, filepath.Join(dirA, "compose.yaml"), `services:
  api:
    image: nginx
    ports:
      - "8080:80"
`)
	writeYAML(t, filepath.Join(dirB, "compose.yaml"), `services:
  web:
    image: nginx
    ports:
      - "8080:80"
`)

	cfg := config.DefaultConfig()
	cfg.Roots = []string{tmpDir}
	cfg.Format = "text"
	cfg.Fix = true
	cfg.DryRun = true

	a, stdout, _ := newTestApp(t, cfg)
	code := a.Run(context.Background())
	if code != 1 {
		t.Errorf("expected exit code 1 (collision, dry-run), got %d", code)
	}
	if !strings.Contains(stdout.String(), "PLANNED") {
		t.Errorf("expected PLANNED in output: %s", stdout.String())
	}
}

func TestRunFixDryRunNoBackup(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Roots = []string{"/tmp"}
	cfg.Fix = true
	cfg.DryRun = true
	cfg.NoBackup = true
	cfg.Yes = true
	cfg.Format = "text"

	a, _, _ := newTestApp(t, cfg)
	code := a.Run(context.Background())
	if code != 2 {
		t.Errorf("expected exit code 2 for --dry-run + --no-backup, got %d", code)
	}
}

func TestRunExcludeRemovesProject(t *testing.T) {
	tmpDir := t.TempDir()
	vendorDir := filepath.Join(tmpDir, "vendor")
	otherDir := filepath.Join(tmpDir, "src")

	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(otherDir, 0755); err != nil {
		t.Fatal(err)
	}

	writeYAML(t, filepath.Join(vendorDir, "compose.yaml"), `services:
  web:
    image: nginx
    ports:
      - "8080:80"
`)
	writeYAML(t, filepath.Join(otherDir, "compose.yaml"), `services:
  api:
    image: nginx
    ports:
      - "9090:80"
`)

	cfg := config.DefaultConfig()
	cfg.Roots = []string{tmpDir}
	cfg.Excludes = []string{vendorDir}
	cfg.Format = "text"

	a, _, _ := newTestApp(t, cfg)
	code := a.Run(context.Background())
	if code != 0 {
		t.Errorf("expected exit code 0 (no collision after exclude), got %d", code)
	}
}

func TestRunDuplicateIntraProject(t *testing.T) {
	tmpDir := t.TempDir()
	writeYAML(t, filepath.Join(tmpDir, "compose.yaml"), `services:
  web:
    image: nginx
    ports:
      - "8080:80"
  api:
    image: nginx
    ports:
      - "8080:90"
`)

	cfg := config.DefaultConfig()
	cfg.Roots = []string{tmpDir}
	cfg.Format = "text"

	a, _, _ := newTestApp(t, cfg)
	code := a.Run(context.Background())
	if code != 1 {
		t.Errorf("expected exit code 1 (intra-project collision), got %d", code)
	}
}
