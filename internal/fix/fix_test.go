package fix

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pyralis-labs/compose-port-registry/internal/model"
)

func TestPlanRefusedRange(t *testing.T) {
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
				Mutability: model.MutableRange,
			},
			Suggested: &model.Interval{Start: 4000, End: 4000},
		},
	}
	plan := planner.Plan(nil, allocations)
	if len(plan.Edits) != 1 {
		t.Fatalf("expected 1 edit, got %d", len(plan.Edits))
	}
	if !plan.Edits[0].Refused {
		t.Error("expected refused for range mutability")
	}
}

func TestPlanDryRunNoEdits(t *testing.T) {
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	os.WriteFile(composeFile, []byte("services:\n  web:\n    ports:\n      - \"8080:80\"\n"), 0644)

	planner := NewPlanner(".bak", false, true)
	allocations := []AllocationResult{
		{
			Binding: model.Binding{
				ProjectID:  "a",
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
	if len(plan.Edits) != 1 {
		t.Fatalf("expected 1 edit, got %d", len(plan.Edits))
	}
	if plan.Edits[0].Refused {
		t.Fatalf("unexpected refused: %s", plan.Edits[0].RefuseReason)
	}

	fixes, err := planner.Execute(plan, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fixes) != 1 {
		t.Fatalf("expected 1 fix, got %d", len(fixes))
	}
	if fixes[0].Status != model.FixPlanned {
		t.Errorf("expected planned status, got %v", fixes[0].Status)
	}

	original, _ := os.ReadFile(composeFile)
	if string(original) != "services:\n  web:\n    ports:\n      - \"8080:80\"\n" {
		t.Error("dry run modified the original file")
	}
}

func TestPlanAndExecute(t *testing.T) {
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	content := "services:\n  web:\n    ports:\n      - \"8080:80\"\n"
	os.WriteFile(composeFile, []byte(content), 0644)

	planner := NewPlanner(".bak", false, false)
	allocations := []AllocationResult{
		{
			Binding: model.Binding{
				ProjectID:  "a",
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
	if len(plan.Edits) != 1 {
		t.Fatalf("expected 1 edit, got %d", len(plan.Edits))
	}

	fixes, err := planner.Execute(plan, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fixes) != 1 {
		t.Fatalf("expected 1 fix, got %d", len(fixes))
	}
	if fixes[0].Status != model.FixApplied {
		t.Errorf("expected applied, got %v", fixes[0].Status)
	}

	backupFile := composeFile + ".bak"
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		t.Error("backup not created")
	}
}

func TestBindingWithNoSource(t *testing.T) {
	planner := NewPlanner(".bak", false, true)
	allocations := []AllocationResult{
		{
			Binding: model.Binding{
				ProjectID:  "a",
				Service:    "web",
				Protocol:   model.ProtocolTCP,
				HostIP:     model.HostScopeInfo{Scope: model.HostAnyUnspecified},
				Published:  model.Interval{Start: 8080, End: 8080},
				Target:     model.Interval{Start: 80, End: 80},
				Mutability: model.Mutable,
			},
			Suggested: &model.Interval{Start: 4000, End: 4000},
		},
	}
	plan := planner.Plan(nil, allocations)
	if len(plan.Edits) != 1 {
		t.Fatalf("expected 1 edit, got %d", len(plan.Edits))
	}
	if plan.Edits[0].Refused {
		t.Error("expected editable binding not to be refused")
	}
}

func TestRestoreBackups(t *testing.T) {
	tmpDir := t.TempDir()
	origFile := filepath.Join(tmpDir, "test.txt")
	backupFile := origFile + ".bak"
	os.WriteFile(origFile, []byte("modified"), 0644)
	os.WriteFile(backupFile, []byte("original"), 0644)

	err := RestoreBackups(".bak", []string{origFile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(origFile)
	if string(data) != "original" {
		t.Errorf("expected 'original', got %s", string(data))
	}
}

func TestIsWritable(t *testing.T) {
	tmpDir := t.TempDir()
	writeFile := filepath.Join(tmpDir, "writable.txt")
	os.WriteFile(writeFile, []byte("test"), 0644)
	if !isWritable(writeFile) {
		t.Error("expected writable file to be writable")
	}
}

func TestEditBeforeSorting(t *testing.T) {
	a := Edit{Binding: model.Binding{ProjectID: "b", Service: "web"}}
	b := Edit{Binding: model.Binding{ProjectID: "a", Service: "web"}}
	if !editBefore(b, a) {
		t.Error("expected a < b")
	}
}
