package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pyralis-labs/compose-port-registry/internal/config"
	"github.com/pyralis-labs/compose-port-registry/internal/model"
)

func writeYAML(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestNewAppCapturesEnv(t *testing.T) {
	t.Setenv("TEST_PORT_REG", "8080")

	cfg := config.DefaultConfig()
	a := New(cfg)

	if a.Env["TEST_PORT_REG"] != "8080" {
		t.Errorf("expected env TEST_PORT_REG=8080, got %s", a.Env["TEST_PORT_REG"])
	}
}

func TestCaptureEnvHandlesMalformed(t *testing.T) {
	t.Setenv("VALID", "value")

	env := captureEnv()
	if env["VALID"] != "value" {
		t.Errorf("expected VALID=value, got %s", env["VALID"])
	}
}

func TestRunExplicitFile(t *testing.T) {
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	writeYAML(t, composeFile, `services:
  web:
    image: nginx
    ports:
      - "8080:80"
`)

	cfg := config.DefaultConfig()
	cfg.Files = []string{composeFile}
	cfg.ProjectDir = tmpDir
	cfg.Format = "text"

	a, _, _ := newTestApp(t, cfg)
	code := a.Run(context.Background())
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestRunExplicitFileCollision(t *testing.T) {
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	writeYAML(t, composeFile, `services:
  web:
    image: nginx
    ports:
      - "8080:80"
      - "8080:90"
`)

	cfg := config.DefaultConfig()
	cfg.Files = []string{composeFile}
	cfg.ProjectDir = tmpDir
	cfg.Format = "text"

	a, _, _ := newTestApp(t, cfg)
	code := a.Run(context.Background())
	if code != 1 {
		t.Errorf("expected exit code 1 (intra-project duplicate), got %d", code)
	}
}

func TestRunDiscoveryFailure(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Roots = []string{"/nonexistent/path/that/does/not/exist"}
	cfg.Format = "text"

	a, _, _ := newTestApp(t, cfg)
	code := a.Run(context.Background())
	if code != 3 {
		t.Errorf("expected exit code 3 (discovery failure), got %d", code)
	}
}

func TestRunInvalidRange(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Roots = []string{"/tmp"}
	cfg.Range = config.PortRange{Start: 5000, End: 4000}
	cfg.Format = "text"

	a, _, _ := newTestApp(t, cfg)
	code := a.Run(context.Background())
	if code != 2 {
		t.Errorf("expected exit code 2 (invalid config), got %d", code)
	}
}

func TestRunLoadProjectFailure(t *testing.T) {
	tmpDir := t.TempDir()
	badFile := filepath.Join(tmpDir, "compose.yaml")
	writeYAML(t, badFile, `not_valid: yaml: : :`)

	cfg := config.DefaultConfig()
	cfg.Files = []string{badFile}
	cfg.ProjectDir = tmpDir
	cfg.Format = "text"

	a, _, _ := newTestApp(t, cfg)
	code := a.Run(context.Background())
	if code == 0 {
		t.Error("expected non-zero exit for bad yaml")
	}
}

func TestRunDiscoveredProjectLoadError(t *testing.T) {
	tmpDir := t.TempDir()
	subdir := filepath.Join(tmpDir, "bad")
	os.MkdirAll(subdir, 0755)
	badFile := filepath.Join(subdir, "compose.yaml")
	writeYAML(t, badFile, `services:
  web:
    ports:
      - "not-a-port:80"
`)

	cfg := config.DefaultConfig()
	cfg.Roots = []string{tmpDir}
	cfg.Format = "text"

	a, stdout, _ := newTestApp(t, cfg)
	code := a.Run(context.Background())
	if code != 0 && code != 1 {
		t.Errorf("expected exit 0 (with warnings) or 1, got %d", code)
	}
	if !contains(stdout.String(), "WARNING") {
		t.Errorf("expected WARNING in stdout for failed load: %s", stdout.String())
	}
}

func TestRunSuggestedFixWithoutApply(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()
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
	cfg.Roots = []string{dirA, dirB}
	cfg.Suggest = true
	cfg.Format = "text"

	a, stdout, _ := newTestApp(t, cfg)
	code := a.Run(context.Background())
	if code != 1 {
		t.Errorf("expected exit code 1 (collision), got %d", code)
	}
	if !contains(stdout.String(), "PLANNED") {
		t.Errorf("expected PLANNED suggestion in output: %s", stdout.String())
	}
}

func TestRunInvalidFormat(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Roots = []string{"/tmp"}
	cfg.Format = "yaml"

	a, _, _ := newTestApp(t, cfg)
	code := a.Run(context.Background())
	if code != 2 {
		t.Errorf("expected exit code 2 for invalid format, got %d", code)
	}
}

func TestRunFileAndRootsCombined(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Roots = []string{"/tmp"}
	cfg.Files = []string{"/tmp/compose.yaml"}
	cfg.Format = "text"

	a, _, _ := newTestApp(t, cfg)
	code := a.Run(context.Background())
	if code != 2 {
		t.Errorf("expected exit code 2 (cannot combine file+roots), got %d", code)
	}
}

func TestBindingIdentityKeyStability(t *testing.T) {
	a := model.Binding{
		ProjectID: "p", Service: "s", Protocol: model.ProtocolTCP,
		HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any, Canonical: "0.0.0.0"},
		Published: model.Interval{Start: 8080, End: 8080},
		Source:    model.SourceRef{File: "/x.yaml"},
	}
	b := model.Binding{
		ProjectID: "p", Service: "s", Protocol: model.ProtocolTCP,
		HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any, Canonical: "0.0.0.0"},
		Published: model.Interval{Start: 8080, End: 8080},
		Source:    model.SourceRef{File: "/x.yaml"},
	}

	if bindingIdentityKey(a) != bindingIdentityKey(b) {
		t.Error("expected identical keys for identical bindings")
	}

	b.Published.Start = 9090
	if bindingIdentityKey(a) == bindingIdentityKey(b) {
		t.Error("expected different keys when port differs")
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
