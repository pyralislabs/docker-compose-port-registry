package compose

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverProjects(t *testing.T) {
	tmpDir := t.TempDir()

	createDir(t, tmpDir, "proj-a")
	createFile(t, tmpDir, "proj-a/compose.yaml", "services:\n  web:\n    image: nginx\n")

	createDir(t, tmpDir, "proj-b")
	createFile(t, tmpDir, "proj-b/docker-compose.yml", "services:\n  web:\n    image: nginx\n")

	createDir(t, tmpDir, ".hidden")
	createFile(t, tmpDir, ".hidden/compose.yaml", "services:\n  web:\n    image: nginx\n")

	createDir(t, tmpDir, ".git")
	createFile(t, tmpDir, ".git/compose.yaml", "services:\n  web:\n    image: nginx\n")

	projects, warnings, err := DiscoverProjects([]string{tmpDir}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}

	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}
}

func TestDiscoverProjectsAmbiguity(t *testing.T) {
	tmpDir := t.TempDir()

	createDir(t, tmpDir, "proj")
	createFile(t, tmpDir, "proj/compose.yaml", "services:\n  web:\n    image: nginx\n")
	createFile(t, tmpDir, "proj/docker-compose.yml", "services:\n  web:\n    image: nginx\n")

	projects, warnings, err := DiscoverProjects([]string{tmpDir}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}

	if len(warnings) == 0 {
		t.Error("expected warning for ambiguity, got none")
	}
}

func TestDiscoverProjectsExclude(t *testing.T) {
	tmpDir := t.TempDir()

	createDir(t, tmpDir, "vendor")
	createFile(t, tmpDir, "vendor/compose.yaml", "services:\n  web:\n    image: nginx\n")
	createDir(t, tmpDir, "src")
	createFile(t, tmpDir, "src/compose.yaml", "services:\n  web:\n    image: nginx\n")

	projects, _, err := DiscoverProjects([]string{tmpDir}, []string{filepath.Join(tmpDir, "vendor")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("expected 1 project (excluded vendor), got %d", len(projects))
	}
}

func TestShouldSkipDir(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{".git", true},
		{"node_modules", true},
		{"vendor", true},
		{".hidden", true},
		{"src", false},
		{"project", false},
	}
	for _, tt := range tests {
		if got := shouldSkipDir(tt.name); got != tt.want {
			t.Errorf("shouldSkipDir(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIsConventionalFile(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"compose.yaml", true},
		{"compose.yml", true},
		{"docker-compose.yaml", true},
		{"docker-compose.yml", true},
		{"docker-compose.yaml", true},
		{"random.txt", false},
		{"Compose.yaml", false},
	}
	for _, tt := range tests {
		if got := isConventionalFile(tt.name); got != tt.want {
			t.Errorf("isConventionalFile(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func createDir(t *testing.T, base, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(base, dir), 0755); err != nil {
		t.Fatalf("failed to create dir %s: %v", dir, err)
	}
}

func createFile(t *testing.T, base, path, content string) {
	t.Helper()
	fullPath := filepath.Join(base, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("failed to create parent dir: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}

func TestResolveOverrides(t *testing.T) {
	files := ResolveOverrides("/project", []string{"compose.yaml", "compose.dev.yaml"})
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	expected := filepath.FromSlash("/project/compose.yaml")
	if files[0] != expected {
		t.Errorf("expected %s, got %s", expected, files[0])
	}
}

func TestFindDefaultFile(t *testing.T) {
	tmpDir := t.TempDir()
	createFile(t, tmpDir, "compose.yaml", "")

	f, err := findDefaultFile(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Base(f) != "compose.yaml" {
		t.Errorf("expected compose.yaml, got %s", f)
	}

	emptyDir := t.TempDir()
	_, err = findDefaultFile(emptyDir)
	if err == nil {
		t.Error("expected error for empty directory")
	}
}
