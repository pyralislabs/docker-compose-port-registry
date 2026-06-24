package compose

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"

	"github.com/pyralis-labs/compose-port-registry/internal/model"
)

func TestLoadProjectShortSyntax(t *testing.T) {
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	if err := os.WriteFile(composeFile, []byte(`
services:
  web:
    image: nginx
    ports:
      - "8080:80"
      - "127.0.0.1:9090:3000"
      - "53:53/udp"
`), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	result, err := LoadProject(context.Background(), LoadOptions{
		ConfigFiles: []string{composeFile},
		ProjectDir:  tmpDir,
		Env:         map[string]string{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Warnings) != 0 {
		t.Errorf("unexpected warnings: %v", result.Warnings)
	}
	if _, ok := result.Project.Services["web"]; !ok {
		t.Fatal("expected web service")
	}
}

func TestLoadProjectWithEnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(envFile, []byte("WEB_PORT=8080\n"), 0644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	composeFile := filepath.Join(tmpDir, "compose.yaml")
	if err := os.WriteFile(composeFile, []byte(`
services:
  web:
    image: nginx
    ports:
      - "${WEB_PORT}:80"
`), 0644); err != nil {
		t.Fatalf("write compose: %v", err)
	}

	result, err := LoadProject(context.Background(), LoadOptions{
		ConfigFiles: []string{composeFile},
		ProjectDir:  tmpDir,
		Env:         map[string]string{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ports := result.Project.Services["web"].Ports
	if len(ports) == 0 || ports[0].Published != "8080" {
		t.Errorf("expected published port 8080 after interpolation, got %+v", ports)
	}
}

func TestLoadProjectExplicitEnvFileFlag(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, "custom.env")
	if err := os.WriteFile(envFile, []byte("PORT=5555\n"), 0644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	composeFile := filepath.Join(tmpDir, "compose.yaml")
	if err := os.WriteFile(composeFile, []byte(`
services:
  web:
    image: nginx
    ports:
      - "${PORT}:80"
`), 0644); err != nil {
		t.Fatalf("write compose: %v", err)
	}

	result, err := LoadProject(context.Background(), LoadOptions{
		ConfigFiles: []string{composeFile},
		ProjectDir:  tmpDir,
		EnvFiles:    []string{envFile},
		Env:         map[string]string{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Project.Services["web"].Ports[0].Published != "5555" {
		t.Errorf("expected port 5555 from custom env, got %s", result.Project.Services["web"].Ports[0].Published)
	}
}

func TestLoadProjectMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := LoadProject(context.Background(), LoadOptions{
		ProjectDir: tmpDir,
		Env:        map[string]string{},
	})
	if err == nil {
		t.Fatal("expected error for missing compose file")
	}
}

func TestLoadProjectEnvOverridePrecedence(t *testing.T) {
	tmpDir := t.TempDir()
	defaultEnv := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(defaultEnv, []byte("PORT=1111\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	composeFile := filepath.Join(tmpDir, "compose.yaml")
	if err := os.WriteFile(composeFile, []byte(`
services:
  web:
    image: nginx
    ports:
      - "${PORT}:80"
`), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	result, err := LoadProject(context.Background(), LoadOptions{
		ConfigFiles: []string{composeFile},
		ProjectDir:  tmpDir,
		Env:         map[string]string{"PORT": "9999"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := result.Project.Services["web"].Ports[0].Published; got != "9999" {
		t.Errorf("expected PORT=9999 from explicit env to win, got %s", got)
	}
}

func TestParsePublishedEdgeCases(t *testing.T) {
	tests := []struct {
		input   string
		want    model.Interval
		wantErr bool
	}{
		{"", model.Interval{Start: 0, End: 0}, false},
		{"80", model.Interval{Start: 80, End: 80}, false},
		{"8000-8005", model.Interval{Start: 8000, End: 8005}, false},
		{"0", model.Interval{}, true},
		{"abc", model.Interval{}, true},
		{"0-10", model.Interval{}, true},
		{"10-0", model.Interval{}, true},
	}
	for _, tt := range tests {
		got, err := parsePublished(tt.input)
		if tt.wantErr && err == nil {
			t.Errorf("parsePublished(%q) expected error, got %+v", tt.input, got)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("parsePublished(%q) unexpected error: %v", tt.input, err)
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("parsePublished(%q) = %+v, want %+v", tt.input, got, tt.want)
		}
	}
}

func TestPortToBindingAllProtocols(t *testing.T) {
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	if err := os.WriteFile(composeFile, []byte("placeholder"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		port types.ServicePortConfig
	}{
		{
			"tcp default",
			types.ServicePortConfig{Published: "8080", Target: 80},
		},
		{
			"udp explicit",
			types.ServicePortConfig{Published: "53", Target: 53, Protocol: "udp"},
		},
		{
			"uppercase UDP",
			types.ServicePortConfig{Published: "53", Target: 53, Protocol: "UDP"},
		},
		{
			"with host ip v4",
			types.ServicePortConfig{Published: "8080", Target: 80, HostIP: "127.0.0.1"},
		},
		{
			"with host ip v6",
			types.ServicePortConfig{Published: "8080", Target: 80, HostIP: "::1"},
		},
		{
			"range published",
			types.ServicePortConfig{Published: "8000-8005", Target: 80},
		},
		{
			"interpolated",
			types.ServicePortConfig{Published: "${PORT}", Target: 80},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binding, warn := portToBinding(tt.port, "svc", "proj", []string{composeFile})
			if binding.Service != "svc" {
				t.Errorf("expected service svc, got %s", binding.Service)
			}
			if binding.ProjectID != "proj" {
				t.Errorf("expected projectID proj, got %s", binding.ProjectID)
			}
			if binding.Source.File != composeFile {
				t.Errorf("expected source file %s, got %s", composeFile, binding.Source.File)
			}
			_ = warn
		})
	}
}

func TestPortToBindingInvalidPublished(t *testing.T) {
	port := types.ServicePortConfig{Published: "abc", Target: 80}
	binding, warn := portToBinding(port, "svc", "proj", []string{"/x/compose.yaml"})

	if warn == nil {
		t.Error("expected warning for invalid published")
	}
	if binding.Mutability != model.MutableInterpolation {
		t.Errorf("expected MutableInterpolation for invalid published, got %v", binding.Mutability)
	}
	if binding.Published.Start != 0 || binding.Published.End != 0 {
		t.Errorf("expected zero published interval, got %+v", binding.Published)
	}
}

func TestPortToBindingWithIPv6Host(t *testing.T) {
	port := types.ServicePortConfig{Published: "8080", Target: 80, HostIP: "::1"}
	binding, _ := portToBinding(port, "web", "proj", []string{"/x"})

	if binding.HostIP.Scope != model.HostIPv6Specific {
		t.Errorf("expected HostIPv6Specific scope, got %v", binding.HostIP.Scope)
	}
	if binding.HostIP.Canonical != "::1" {
		t.Errorf("expected canonical ::1, got %s", binding.HostIP.Canonical)
	}
}

func TestSourceFileEmpty(t *testing.T) {
	if got := sourceFile(nil); got != "" {
		t.Errorf("expected empty source for nil files, got %s", got)
	}
	if got := sourceFile([]string{"/a/b.yaml"}); got != "/a/b.yaml" {
		t.Errorf("expected /a/b.yaml, got %s", got)
	}
}

func TestBuildEnvMapPriority(t *testing.T) {
	tmpDir := t.TempDir()
	defaultEnv := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(defaultEnv, []byte("A=from_default\nB=from_default\n# comment\n\nC=from_default\n"), 0644); err != nil {
		t.Fatal(err)
	}

	customEnv := filepath.Join(tmpDir, "custom.env")
	if err := os.WriteFile(customEnv, []byte("A=from_custom\nD=from_custom\n"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("process env wins", func(t *testing.T) {
		got := buildEnvMap(map[string]string{"A": "from_process", "E": "from_process"}, nil, tmpDir)
		if got["A"] != "from_process" {
			t.Errorf("expected A from process, got %s", got["A"])
		}
		if got["B"] != "from_default" {
			t.Errorf("expected B from default, got %s", got["B"])
		}
		if got["C"] != "from_default" {
			t.Errorf("expected C from default, got %s", got["C"])
		}
	})

	t.Run("custom env file overrides default", func(t *testing.T) {
		got := buildEnvMap(map[string]string{}, []string{customEnv}, tmpDir)
		if got["A"] != "from_custom" {
			t.Errorf("expected A from custom, got %s", got["A"])
		}
		if got["D"] != "from_custom" {
			t.Errorf("expected D from custom, got %s", got["D"])
		}
	})

	t.Run("missing default env returns empty", func(t *testing.T) {
		emptyDir := t.TempDir()
		got := buildEnvMap(map[string]string{}, nil, emptyDir)
		if len(got) != 0 {
			t.Errorf("expected empty map, got %v", got)
		}
	})

	t.Run("missing custom env file ignored", func(t *testing.T) {
		got := buildEnvMap(map[string]string{}, []string{"/nonexistent.env"}, tmpDir)
		if _, ok := got["A"]; ok {
			t.Errorf("did not expect A from missing file, got %s", got["A"])
		}
	})
}

func TestResolveAbsoluteFiles(t *testing.T) {
	got := resolveAbsoluteFiles([]string{"compose.yaml", "/abs/path.yaml"}, "/project")
	want := []string{"/project/compose.yaml", "/abs/path.yaml"}
	if len(got) != len(want) {
		t.Fatalf("expected %d files, got %d", len(want), len(got))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("got %s, want %s", got[i], want[i])
		}
	}
}

func TestLoadProjectDefaultFileFound(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "docker-compose.yml"), []byte(`
services:
  web:
    image: nginx
    ports:
      - "8080:80"
`), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := LoadProject(context.Background(), LoadOptions{
		ProjectDir: tmpDir,
		Env:        map[string]string{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := result.Project.Services["web"]; !ok {
		t.Error("expected web service from default file lookup")
	}
}

func TestLoadProjectSetProjectName(t *testing.T) {
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	if err := os.WriteFile(composeFile, []byte(`
name: my-project
services:
  web:
    image: nginx
    ports:
      - "8080:80"
`), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := LoadProject(context.Background(), LoadOptions{
		ConfigFiles: []string{composeFile},
		ProjectDir:  tmpDir,
		Env:         map[string]string{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Project.Name != "my-project" {
		t.Errorf("expected project name my-project, got %s", result.Project.Name)
	}
}

func TestLoadProjectFilesAreAbsolute(t *testing.T) {
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	if err := os.WriteFile(composeFile, []byte(`
services:
  web:
    image: nginx
    ports:
      - "8080:80"
`), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := LoadProject(context.Background(), LoadOptions{
		ConfigFiles: []string{composeFile},
		ProjectDir:  tmpDir,
		Env:         map[string]string{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, f := range result.Project.Files {
		if !filepath.IsAbs(f) {
			t.Errorf("expected absolute file path, got %s", f)
		}
	}
}

func TestNormalizePortsProtocolCase(t *testing.T) {
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	if err := os.WriteFile(composeFile, []byte(`
services:
  dns:
    image: dns
    ports:
      - "53:53/udp"
      - "54:54/tcp"
`), 0644); err != nil {
		t.Fatal(err)
	}

	r, err := LoadProject(context.Background(), LoadOptions{
		ConfigFiles: []string{composeFile},
		ProjectDir:  tmpDir,
		Env:         map[string]string{},
	})
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	bindings, _ := NormalizePorts(r.Project.Services, r.Project.Name, r.Project.Files)
	if len(bindings) != 2 {
		t.Fatalf("expected 2 bindings, got %d", len(bindings))
	}

	protocols := make(map[model.Protocol]bool)
	for _, b := range bindings {
		protocols[b.Protocol] = true
	}
	if !protocols[model.ProtocolUDP] {
		t.Error("expected UDP binding")
	}
	if !protocols[model.ProtocolTCP] {
		t.Error("expected TCP binding")
	}
}

func TestNormalizePortsSorted(t *testing.T) {
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	if err := os.WriteFile(composeFile, []byte(`
services:
  zebra:
    image: z
    ports:
      - "9090:90"
  alpha:
    image: a
    ports:
      - "8080:80"
`), 0644); err != nil {
		t.Fatal(err)
	}

	r, err := LoadProject(context.Background(), LoadOptions{
		ConfigFiles: []string{composeFile},
		ProjectDir:  tmpDir,
		Env:         map[string]string{},
	})
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	bindings, _ := NormalizePorts(r.Project.Services, r.Project.Name, r.Project.Files)
	if len(bindings) < 2 {
		t.Fatalf("expected at least 2 bindings, got %d", len(bindings))
	}
	if bindings[0].Service > bindings[1].Service {
		t.Errorf("expected bindings sorted, got %s then %s", bindings[0].Service, bindings[1].Service)
	}
}
