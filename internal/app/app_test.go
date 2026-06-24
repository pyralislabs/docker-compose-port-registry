package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pyralis-labs/compose-port-registry/internal/config"
	"github.com/pyralis-labs/compose-port-registry/internal/model"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func newTestApp(t *testing.T, cfg *config.Config) (*App, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	return &App{
		Config: cfg,
		Env:    map[string]string{},
		Stdout: stdout,
		Stderr: stderr,
	}, stdout, stderr
}

func TestRunNoCollisionsExitZero(t *testing.T) {
	tmpDir := t.TempDir()
	writeFile(t, filepath.Join(tmpDir, "compose.yaml"), "services:\n  web:\n    image: nginx\n    ports:\n      - \"8080:80\"\n  api:\n    image: api\n    ports:\n      - \"9090:3000\"\n")

	cfg := config.DefaultConfig()
	cfg.Roots = []string{tmpDir}
	cfg.Format = "text"

	a, _, _ := newTestApp(t, cfg)
	code := a.Run(context.Background())
	if code != 0 {
		t.Errorf("expected exit code 0 for no collisions, got %d", code)
	}
}

func TestRunCollisionExitOne(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()
	writeFile(t, filepath.Join(dirA, "compose.yaml"), "services:\n  api:\n    image: api\n    ports:\n      - \"8080:80\"\n")
	writeFile(t, filepath.Join(dirB, "compose.yaml"), "services:\n  web:\n    image: web\n    ports:\n      - \"8080:3000\"\n")

	cfg := config.DefaultConfig()
	cfg.Roots = []string{filepath.Dir(dirA), filepath.Dir(dirB)}
	cfg.Format = "text"

	a, _, _ := newTestApp(t, cfg)
	code := a.Run(context.Background())
	if code != 1 {
		t.Errorf("expected exit code 1 for collisions, got %d", code)
	}
}

func TestRunSuccessfulFixExitZero(t *testing.T) {
	tmpDir := t.TempDir()
	writeFile(t, filepath.Join(tmpDir, "proj-a", "compose.yaml"), "services:\n  api:\n    image: api\n    ports:\n      - \"8080:80\"\n")
	writeFile(t, filepath.Join(tmpDir, "proj-b", "compose.yaml"), "services:\n  web:\n    image: web\n    ports:\n      - \"8080:3000\"\n")

	cfg := config.DefaultConfig()
	cfg.Roots = []string{tmpDir}
	cfg.Format = "text"
	cfg.Fix = true
	cfg.Yes = true

	a, _, _ := newTestApp(t, cfg)
	code := a.Run(context.Background())
	if code != 0 {
		t.Errorf("expected exit code 0 after successful fix, got %d", code)
	}
}

func TestRunFixRequiresYesExitFour(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()
	writeFile(t, filepath.Join(dirA, "compose.yaml"), "services:\n  api:\n    image: api\n    ports:\n      - \"8080:80\"\n")
	writeFile(t, filepath.Join(dirB, "compose.yaml"), "services:\n  web:\n    image: web\n    ports:\n      - \"8080:3000\"\n")

	cfg := config.DefaultConfig()
	cfg.Roots = []string{filepath.Dir(dirA), filepath.Dir(dirB)}
	cfg.Format = "text"
	cfg.Fix = true

	a, _, _ := newTestApp(t, cfg)
	code := a.Run(context.Background())
	if code != 4 {
		t.Errorf("expected exit code 4 for fix without yes, got %d", code)
	}
}

func TestRunVersion(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ShowVersion = true

	a, stdout, _ := newTestApp(t, cfg)
	code := a.Run(context.Background())
	if code != 0 {
		t.Errorf("expected exit code 0 for version, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(model.Version)) {
		t.Errorf("expected version string in output: %s", stdout.String())
	}
}

func TestRunInvalidConfigExitTwo(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Format = "xml"

	a, _, _ := newTestApp(t, cfg)
	code := a.Run(context.Background())
	if code != 2 {
		t.Errorf("expected exit code 2 for invalid config, got %d", code)
	}
}

func TestRunJSONOutput(t *testing.T) {
	tmpDir := t.TempDir()
	writeFile(t, filepath.Join(tmpDir, "compose.yaml"), "services:\n  web:\n    image: nginx\n    ports:\n      - \"8080:80\"\n")

	cfg := config.DefaultConfig()
	cfg.Roots = []string{tmpDir}
	cfg.Format = "json"

	a, stdout, _ := newTestApp(t, cfg)
	code := a.Run(context.Background())
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	var r model.Report
	if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, stdout.String())
	}
	if r.SchemaVersion != "1" {
		t.Errorf("expected schema_version 1, got %s", r.SchemaVersion)
	}
	if r.ToolVersion != model.Version {
		t.Errorf("expected tool_version %s, got %s", model.Version, r.ToolVersion)
	}
}

func TestHasUnresolvedCollisions(t *testing.T) {
	mkBinding := func(id, svc string, port uint16) model.Binding {
		return model.Binding{
			ProjectID: id,
			Service:   svc,
			Protocol:  model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any, Canonical: "0.0.0.0"},
			Published: model.Interval{Start: port, End: port},
			Source:    model.SourceRef{File: id + ".yaml"},
		}
	}

	collisionAB := model.Collision{
		ID:        "collision:tcp:ipv4-any:8080",
		Protocol:  model.ProtocolTCP,
		HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any, Canonical: "0.0.0.0"},
		Published: model.Interval{Start: 8080, End: 8080},
		Bindings: []model.Binding{
			mkBinding("a", "api", 8080),
			mkBinding("b", "web", 8080),
		},
	}

	if hasUnresolvedCollisions(nil, nil) {
		t.Error("expected no unresolved with empty collisions")
	}

	if !hasUnresolvedCollisions([]model.Collision{collisionAB}, nil) {
		t.Error("expected unresolved without any fixes")
	}

	fixApplied := []model.Fix{
		{
			Binding: mkBinding("b", "web", 8080),
			Status:  model.FixApplied,
		},
	}
	if hasUnresolvedCollisions([]model.Collision{collisionAB}, fixApplied) {
		t.Error("expected resolved after applying fix to loser")
	}

	fixRefused := []model.Fix{
		{
			Binding: mkBinding("b", "web", 8080),
			Status:  model.FixRefused,
		},
	}
	if !hasUnresolvedCollisions([]model.Collision{collisionAB}, fixRefused) {
		t.Error("expected unresolved when fix is refused")
	}
}
