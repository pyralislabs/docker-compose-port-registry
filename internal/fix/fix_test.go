package fix

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pyralis-labs/compose-port-registry/internal/model"
)

func TestPlanRefusedOverrideFile(t *testing.T) {
	planner := NewPlanner(".bak", false, true)
	allocations := []AllocationResult{
		{
			Binding: model.Binding{
				ProjectID:  "a",
				Service:    "web",
				Protocol:   model.ProtocolTCP,
				HostIP:     model.HostScopeInfo{Scope: model.HostIPv4Any},
				Published:  model.Interval{Start: 8080, End: 8080},
				Target:     model.Interval{Start: 80, End: 80},
				Mutability: model.MutableOverride,
			},
			Suggested: &model.Interval{Start: 4000, End: 4000},
		},
	}
	plan := planner.Plan(nil, allocations)
	if !plan.Edits[0].Refused {
		t.Error("expected refused for MutableOverride")
	}
	if plan.Edits[0].RefuseReason != "refused:override" {
		t.Errorf("expected refused:override reason, got %s", plan.Edits[0].RefuseReason)
	}
}

func TestPlanRefusedLongSyntax(t *testing.T) {
	planner := NewPlanner(".bak", false, true)
	allocations := []AllocationResult{
		{
			Binding: model.Binding{
				Mutability: model.MutableLongSyntax,
				Source:     model.SourceRef{File: "/nope.yaml"},
			},
			Suggested: &model.Interval{Start: 4000, End: 4000},
		},
	}
	plan := planner.Plan(nil, allocations)
	if !plan.Edits[0].Refused {
		t.Error("expected refused for MutableLongSyntax")
	}
}

func TestPlanRefusedStatFailure(t *testing.T) {
	planner := NewPlanner(".bak", false, true)
	allocations := []AllocationResult{
		{
			Binding: model.Binding{
				Mutability: model.Mutable,
				Source:     model.SourceRef{File: "/nonexistent/file.yaml"},
			},
			Suggested: &model.Interval{Start: 4000, End: 4000},
		},
	}
	plan := planner.Plan(nil, allocations)
	if !plan.Edits[0].Refused {
		t.Error("expected refused for stat failure")
	}
	if plan.Edits[0].RefuseReason == "" {
		t.Error("expected non-empty refuse reason")
	}
}

func TestPlanSkipsExhaustedAllocations(t *testing.T) {
	planner := NewPlanner(".bak", false, true)
	allocations := []AllocationResult{
		{
			Binding: model.Binding{
				Mutability: model.Mutable,
				Source:     model.SourceRef{File: "/nonexistent.yaml"},
			},
			Exhausted: true,
		},
		{
			Binding: model.Binding{
				Mutability: model.Mutable,
				Source:     model.SourceRef{File: "/nonexistent2.yaml"},
			},
			Suggested: nil,
		},
	}
	plan := planner.Plan(nil, allocations)
	if len(plan.Edits) != 0 {
		t.Errorf("expected 0 edits (exhausted/nil skipped), got %d", len(plan.Edits))
	}
}

func TestPlanWithHostIPEdit(t *testing.T) {
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	os.WriteFile(composeFile, []byte("services:\n  web:\n    ports:\n      - \"127.0.0.1:8080:80\"\n"), 0644)

	planner := NewPlanner(".bak", false, true)
	allocations := []AllocationResult{
		{
			Binding: model.Binding{
				ProjectID:  "p",
				Service:    "web",
				Protocol:   model.ProtocolTCP,
				HostIP:     model.HostScopeInfo{Scope: model.HostIPv4Specific, Canonical: "127.0.0.1"},
				Published:  model.Interval{Start: 8080, End: 8080},
				Target:     model.Interval{Start: 80, End: 80},
				Source:     model.SourceRef{File: composeFile},
				Mutability: model.Mutable,
			},
			Suggested: &model.Interval{Start: 4000, End: 4000},
		},
	}
	plan := planner.Plan(nil, allocations)
	if len(plan.Edits) != 1 {
		t.Fatalf("expected 1 edit, got %d", len(plan.Edits))
	}
	if plan.Edits[0].OldValue != "127.0.0.1:8080:80" {
		t.Errorf("expected old value '127.0.0.1:8080:80', got %s", plan.Edits[0].OldValue)
	}
	if plan.Edits[0].NewValue != "127.0.0.1:4000:80" {
		t.Errorf("expected new value '127.0.0.1:4000:80', got %s", plan.Edits[0].NewValue)
	}
}

func TestExecutePlannedWithoutApply(t *testing.T) {
	planner := NewPlanner(".bak", false, false)
	plan := &Plan{
		Edits: []Edit{
			{
				Binding:  model.Binding{ProjectID: "p", Service: "web"},
				OldValue: "8080:80",
				NewValue: "4000:80",
			},
		},
	}

	fixes, err := planner.Execute(plan, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fixes[0].Status != model.FixPlanned {
		t.Errorf("expected FixPlanned, got %v", fixes[0].Status)
	}
}

func TestExecuteRefusedPasses(t *testing.T) {
	planner := NewPlanner(".bak", false, false)
	plan := &Plan{
		Edits: []Edit{
			{
				Binding:      model.Binding{ProjectID: "p", Service: "web"},
				OldValue:     "8080:80",
				NewValue:     "4000:80",
				Refused:      true,
				RefuseReason: "test reason",
			},
		},
	}

	fixes, _ := planner.Execute(plan, true)
	if fixes[0].Status != model.FixRefused {
		t.Errorf("expected FixRefused, got %v", fixes[0].Status)
	}
	if fixes[0].Reason != "test reason" {
		t.Errorf("expected test reason, got %s", fixes[0].Reason)
	}
}

func TestApplyEditSuccessful(t *testing.T) {
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	os.WriteFile(composeFile, []byte("services:\n  web:\n    ports:\n      - \"8080:80\"\n"), 0644)

	planner := NewPlanner(".bak", false, false)
	allocations := []AllocationResult{
		{
			Binding: model.Binding{
				ProjectID:  "p",
				Service:    "web",
				Protocol:   model.ProtocolTCP,
				HostIP:     model.HostScopeInfo{Scope: model.HostAnyUnspecified},
				Published:  model.Interval{Start: 8080, End: 8080},
				Target:     model.Interval{Start: 80, End: 80},
				Source:     model.SourceRef{File: composeFile},
				Mutability: model.Mutable,
			},
			Suggested: &model.Interval{Start: 4000, End: 4000},
		},
	}
	plan := planner.Plan(nil, allocations)

	fixes, err := planner.Execute(plan, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fixes[0].Status != model.FixApplied {
		t.Errorf("expected FixApplied, got %v", fixes[0].Status)
	}

	data, _ := os.ReadFile(composeFile)
	if !contains(string(data), "4000:80") {
		t.Errorf("expected new port in file, got: %s", string(data))
	}

	if _, err := os.Stat(composeFile + ".bak"); err != nil {
		t.Errorf("expected backup to exist: %v", err)
	}
}

func TestApplyEditNoBackup(t *testing.T) {
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	os.WriteFile(composeFile, []byte("services:\n  web:\n    ports:\n      - \"8080:80\"\n"), 0644)

	planner := NewPlanner(".bak", true, false)
	allocations := []AllocationResult{
		{
			Binding: model.Binding{
				ProjectID:  "p",
				Service:    "web",
				Protocol:   model.ProtocolTCP,
				HostIP:     model.HostScopeInfo{Scope: model.HostAnyUnspecified},
				Published:  model.Interval{Start: 8080, End: 8080},
				Target:     model.Interval{Start: 80, End: 80},
				Source:     model.SourceRef{File: composeFile},
				Mutability: model.Mutable,
			},
			Suggested: &model.Interval{Start: 4000, End: 4000},
		},
	}
	plan := planner.Plan(nil, allocations)

	fixes, _ := planner.Execute(plan, true)
	if fixes[0].Status != model.FixApplied {
		t.Errorf("expected FixApplied, got %v", fixes[0].Status)
	}

	if _, err := os.Stat(composeFile + ".bak"); !os.IsNotExist(err) {
		t.Errorf("expected no backup with NoBackup=true, got: %v", err)
	}
}

func TestApplyEditExistingBackupRefused(t *testing.T) {
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	os.WriteFile(composeFile, []byte("services:\n  web:\n    ports:\n      - \"8080:80\"\n"), 0644)

	os.WriteFile(composeFile+".bak", []byte("existing"), 0644)

	planner := NewPlanner(".bak", false, false)
	allocations := []AllocationResult{
		{
			Binding: model.Binding{
				ProjectID:  "p",
				Service:    "web",
				Protocol:   model.ProtocolTCP,
				HostIP:     model.HostScopeInfo{Scope: model.HostAnyUnspecified},
				Published:  model.Interval{Start: 8080, End: 8080},
				Target:     model.Interval{Start: 80, End: 80},
				Source:     model.SourceRef{File: composeFile},
				Mutability: model.Mutable,
			},
			Suggested: &model.Interval{Start: 4000, End: 4000},
		},
	}
	plan := planner.Plan(nil, allocations)

	fixes, _ := planner.Execute(plan, true)
	if fixes[0].Status != model.FixRolledBack {
		t.Errorf("expected FixRolledBack, got %v", fixes[0].Status)
	}

	original, _ := os.ReadFile(composeFile)
	if string(original) != "services:\n  web:\n    ports:\n      - \"8080:80\"\n" {
		t.Errorf("original file should not be modified")
	}
}

func TestApplyEditMissingFileRefused(t *testing.T) {
	planner := NewPlanner(".bak", false, false)
	edit := Edit{
		Binding:  model.Binding{Source: model.SourceRef{File: ""}},
		OldValue: "8080:80",
		NewValue: "4000:80",
	}

	err := planner.applyEdit(edit)
	if err == nil {
		t.Error("expected error for missing source file")
	}
}

func TestApplyEditUnreadableFile(t *testing.T) {
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	os.WriteFile(composeFile, []byte("services:\n  web:\n    ports:\n      - \"8080:80\"\n"), 0000)
	defer os.Chmod(composeFile, 0644)

	planner := NewPlanner(".bak", false, false)
	edit := Edit{
		Binding:  model.Binding{Source: model.SourceRef{File: composeFile}},
		OldValue: "8080:80",
		NewValue: "4000:80",
	}

	err := planner.applyEdit(edit)
	if err == nil {
		t.Error("expected error for unreadable file")
	}
}

func TestFindAndReplaceScalarPortsList(t *testing.T) {
	doc := yamlParse("services:\n  web:\n    ports:\n      - \"8080:80\"\n      - \"9090:90\"\n")

	modified, err := findAndReplaceInNode(doc, "8080:80", "4000:80")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !modified {
		t.Error("expected modification")
	}

	rendered := yamlRender(doc)
	if !contains(rendered, "4000:80") || contains(rendered, "8080:80") {
		t.Errorf("expected 8080:80 replaced, got: %s", rendered)
	}
}

func TestFindAndReplaceScalarNotFound(t *testing.T) {
	doc := yamlParse("services:\n  web:\n    image: nginx\n")

	modified, err := findAndReplaceInNode(doc, "8080:80", "4000:80")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if modified {
		t.Error("expected no modification")
	}
}

func TestFindAndReplaceScalarNestedPorts(t *testing.T) {
	doc := yamlParse("services:\n  web:\n    image: nginx\n    ports:\n      - \"8080:80\"\n")

	modified, _ := findAndReplaceInNode(doc, "8080:80", "4000:80")
	if !modified {
		t.Error("expected modification in nested ports")
	}
}

func TestEditBeforeSortProjectAndFile(t *testing.T) {
	a := Edit{Binding: model.Binding{ProjectID: "z", Source: model.SourceRef{File: "a"}}}
	b := Edit{Binding: model.Binding{ProjectID: "a", Source: model.SourceRef{File: "z"}}}
	c := Edit{Binding: model.Binding{ProjectID: "a", Source: model.SourceRef{File: "a"}}}

	if !editBefore(c, b) {
		t.Error("expected c < b (same project, a < z file)")
	}
	if !editBefore(b, a) {
		t.Error("expected b < a (a < z project)")
	}
}

func TestWriteTemporaryAndCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	os.WriteFile(composeFile, []byte(""), 0644)

	tmpFile, err := writeTemporary(composeFile, []byte("new content"))
	if err != nil {
		t.Fatalf("writeTemporary: %v", err)
	}

	data, _ := os.ReadFile(tmpFile)
	if string(data) != "new content" {
		t.Errorf("expected tmp file content, got %s", string(data))
	}

	os.Remove(tmpFile)
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src.txt")
	dst := filepath.Join(tmpDir, "dst.txt")
	os.WriteFile(src, []byte("content"), 0644)

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	data, _ := os.ReadFile(dst)
	if string(data) != "content" {
		t.Errorf("expected copied content, got %s", string(data))
	}
}

func TestCopyFileMissingSrc(t *testing.T) {
	err := copyFile("/nonexistent/path.txt", "/tmp/dst.txt")
	if err == nil {
		t.Error("expected error for missing source")
	}
}

func TestHashFile(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "f.txt")
	os.WriteFile(f, []byte("hello"), 0644)

	h, err := hashFile(f)
	if err != nil {
		t.Fatalf("hashFile: %v", err)
	}
	if len(h) != 64 {
		t.Errorf("expected 64-char hex hash, got %d chars", len(h))
	}
}

func TestHashFileMissing(t *testing.T) {
	_, err := hashFile("/nonexistent/path.txt")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestIsWritableReadOnly(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "f.txt")
	os.WriteFile(f, []byte(""), 0000)
	defer os.Chmod(f, 0644)

	if isWritable(f) {
		t.Error("expected read-only file to not be writable")
	}
}

func TestIsWritableMissing(t *testing.T) {
	if isWritable("/nonexistent/file.txt") {
		t.Error("expected missing file to not be writable")
	}
}

func TestRestoreBackupsMissingBackup(t *testing.T) {
	err := RestoreBackups(".bak", []string{"/nonexistent/file.yaml"})
	if err == nil {
		t.Error("expected error for missing backup")
	}
}

func TestRestoreBackupsRenameError(t *testing.T) {
	tmpDir := t.TempDir()
	original := filepath.Join(tmpDir, "locked.yaml")
	backup := original + ".bak"

	os.WriteFile(original, []byte("orig"), 0644)
	os.WriteFile(backup, []byte("backup"), 0644)

	if err := os.Chmod(tmpDir, 0555); err != nil {
		t.Skipf("can't chmod: %v", err)
	}
	defer os.Chmod(tmpDir, 0755)

	err := RestoreBackups(".bak", []string{original})
	_ = err
}
