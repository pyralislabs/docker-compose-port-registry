package fix

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pyralis-labs/compose-port-registry/internal/model"
)

func TestApplyEditRollbackOnRenameFailure(t *testing.T) {
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	if err := os.WriteFile(composeFile, []byte("services:\n  web:\n    ports:\n      - \"8080:80\"\n"), 0444); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(composeFile, 0644)

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
		t.Errorf("expected FixRolledBack due to read-only file, got %v", fixes[0].Status)
	}
}

func TestApplyEditTargetValueNotInFile(t *testing.T) {
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	if err := os.WriteFile(composeFile, []byte("services:\n  web:\n    ports:\n      - \"9999:80\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

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
		t.Errorf("expected FixRolledBack (scalar not found), got %v", fixes[0].Status)
	}
}

func TestApplyEditInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	if err := os.WriteFile(composeFile, []byte("invalid: : : yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	planner := NewPlanner(".bak", false, false)
	allocations := []AllocationResult{
		{
			Binding: model.Binding{
				Mutability: model.Mutable,
				Source:     model.SourceRef{File: composeFile},
			},
			Suggested: &model.Interval{Start: 4000, End: 4000},
		},
	}
	plan := planner.Plan(nil, allocations)

	fixes, _ := planner.Execute(plan, true)
	if fixes[0].Status != model.FixRolledBack {
		t.Errorf("expected FixRolledBack for invalid YAML, got %v", fixes[0].Status)
	}
}

func TestApplyEditDryRunDoesNotWrite(t *testing.T) {
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	original := "services:\n  web:\n    ports:\n      - \"8080:80\"\n"
	if err := os.WriteFile(composeFile, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	planner := NewPlanner(".bak", false, true)
	allocations := []AllocationResult{
		{
			Binding: model.Binding{
				Mutability: model.Mutable,
				Published:  model.Interval{Start: 8080, End: 8080},
				Target:     model.Interval{Start: 80, End: 80},
				Source:     model.SourceRef{File: composeFile},
			},
			Suggested: &model.Interval{Start: 4000, End: 4000},
		},
	}
	plan := planner.Plan(nil, allocations)

	fixes, _ := planner.Execute(plan, true)
	if fixes[0].Status != model.FixApplied {
		t.Errorf("expected FixApplied in dry-run mode, got %v", fixes[0].Status)
	}

	data, _ := os.ReadFile(composeFile)
	if string(data) != original {
		t.Error("dry-run should not modify file")
	}

	if _, err := os.Stat(composeFile + ".bak"); !os.IsNotExist(err) {
		t.Error("dry-run should not create backup")
	}
}

func TestPlanAndExecuteBackupSuffixCustom(t *testing.T) {
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	if err := os.WriteFile(composeFile, []byte("services:\n  web:\n    ports:\n      - \"8080:80\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	planner := NewPlanner(".custom.bak", false, false)
	allocations := []AllocationResult{
		{
			Binding: model.Binding{
				Mutability: model.Mutable,
				Published:  model.Interval{Start: 8080, End: 8080},
				Target:     model.Interval{Start: 80, End: 80},
				Source:     model.SourceRef{File: composeFile},
			},
			Suggested: &model.Interval{Start: 4000, End: 4000},
		},
	}
	plan := planner.Plan(nil, allocations)
	fixes, _ := planner.Execute(plan, true)
	if fixes[0].Status != model.FixApplied {
		t.Errorf("expected FixApplied, got %v", fixes[0].Status)
	}

	if _, err := os.Stat(composeFile + ".custom.bak"); err != nil {
		t.Errorf("expected custom backup suffix file: %v", err)
	}
}

func TestExecuteSortsFixesByProjectService(t *testing.T) {
	planner := NewPlanner(".bak", false, false)
	plan := &Plan{
		Edits: []Edit{
			{
				Binding:  model.Binding{ProjectID: "z", Service: "web", Source: model.SourceRef{File: "/x"}},
				NewValue: "z",
			},
			{
				Binding:  model.Binding{ProjectID: "a", Service: "z", Source: model.SourceRef{File: "/x"}},
				NewValue: "a",
			},
			{
				Binding:  model.Binding{ProjectID: "a", Service: "a", Source: model.SourceRef{File: "/a"}},
				NewValue: "aa",
			},
		},
	}

	fixes, _ := planner.Execute(plan, false)
	if fixes[0].Binding.ProjectID != "a" {
		t.Errorf("expected first fix project a, got %s", fixes[0].Binding.ProjectID)
	}
}
