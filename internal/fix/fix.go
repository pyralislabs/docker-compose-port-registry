package fix

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/pyralis-labs/compose-port-registry/internal/model"
)

type Plan struct {
	Edits []Edit
	Files map[string]*FileState
}

type Edit struct {
	Binding      model.Binding
	NewPort      uint16
	OldValue     string
	NewValue     string
	Refused      bool
	RefuseReason string
}

type FileState struct {
	Path         string
	OriginalHash string
	Mode         os.FileMode
}

type Planner struct {
	BackupSuffix string
	NoBackup     bool
	DryRun       bool
}

func NewPlanner(backupSuffix string, noBackup, dryRun bool) *Planner {
	return &Planner{
		BackupSuffix: backupSuffix,
		NoBackup:     noBackup,
		DryRun:       dryRun,
	}
}

func (p *Planner) Plan(bindings []model.Binding, allocations []AllocationResult) *Plan {
	plan := &Plan{
		Edits: make([]Edit, 0),
		Files: make(map[string]*FileState),
	}

	for _, alloc := range allocations {
		if alloc.Exhausted || alloc.Suggested == nil {
			continue
		}

		b := alloc.Binding
		edit := Edit{
			Binding: b,
			NewPort: alloc.Suggested.Start,
		}

		if b.HostIP.Scope != model.HostAnyUnspecified && b.HostIP.Canonical != "" {
			edit.OldValue = fmt.Sprintf("%s:%d:%d", b.HostIP.Canonical, b.Published.Start, b.Target.Start)
			edit.NewValue = fmt.Sprintf("%s:%d:%d", b.HostIP.Canonical, edit.NewPort, b.Target.Start)
		} else {
			edit.OldValue = fmt.Sprintf("%d:%d", b.Published.Start, b.Target.Start)
			edit.NewValue = fmt.Sprintf("%d:%d", edit.NewPort, b.Target.Start)
		}

		if b.Mutability != model.Mutable {
			edit.Refused = true
			edit.RefuseReason = b.Mutability.String()
			plan.Edits = append(plan.Edits, edit)
			continue
		}

		if b.Source.File != "" {
			if _, exists := plan.Files[b.Source.File]; !exists {
				info, err := os.Stat(b.Source.File)
				if err != nil {
					edit.Refused = true
					edit.RefuseReason = fmt.Sprintf("cannot stat file: %v", err)
					plan.Edits = append(plan.Edits, edit)
					continue
				}
				hash, err := hashFile(b.Source.File)
				if err != nil {
					edit.Refused = true
					edit.RefuseReason = fmt.Sprintf("cannot hash file: %v", err)
					plan.Edits = append(plan.Edits, edit)
					continue
				}
				plan.Files[b.Source.File] = &FileState{
					Path:         b.Source.File,
					OriginalHash: hash,
					Mode:         info.Mode(),
				}
			}
		}

		plan.Edits = append(plan.Edits, edit)
	}

	sort.Slice(plan.Edits, func(i, j int) bool {
		return editBefore(plan.Edits[i], plan.Edits[j])
	})

	return plan
}

type AllocationResult struct {
	Binding   model.Binding
	Suggested *model.Interval
	Exhausted bool
}

func (p *Planner) Execute(plan *Plan, yes bool) ([]model.Fix, error) {
	var fixes []model.Fix

	for _, edit := range plan.Edits {
		if edit.Refused {
			fixes = append(fixes, model.Fix{
				Binding:  edit.Binding,
				OldValue: edit.OldValue,
				Status:   model.FixRefused,
				Reason:   edit.RefuseReason,
			})
			continue
		}

		if !yes {
			fixes = append(fixes, model.Fix{
				Binding:  edit.Binding,
				OldValue: edit.OldValue,
				NewValue: edit.NewValue,
				Status:   model.FixPlanned,
			})
			continue
		}

		err := p.applyEdit(edit)
		if err != nil {
			fixes = append(fixes, model.Fix{
				Binding:  edit.Binding,
				OldValue: edit.OldValue,
				Status:   model.FixRolledBack,
				Reason:   err.Error(),
			})
			continue
		}

		fixes = append(fixes, model.Fix{
			Binding:  edit.Binding,
			OldValue: edit.OldValue,
			NewValue: edit.NewValue,
			Status:   model.FixApplied,
		})
	}

	model.SortFixes(fixes)
	return fixes, nil
}

func (p *Planner) applyEdit(edit Edit) error {
	srcFile := edit.Binding.Source.File
	if srcFile == "" {
		return fmt.Errorf("no source file for binding")
	}

	data, err := os.ReadFile(srcFile)
	if err != nil {
		return fmt.Errorf("cannot read file: %w", err)
	}

	if !isWritable(srcFile) {
		return fmt.Errorf("file is not writable")
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("cannot parse YAML: %w", err)
	}

	modified, err := findAndReplaceScalar(&doc, edit.OldValue, edit.NewValue)
	if err != nil {
		return fmt.Errorf("cannot locate scalar to edit: %w", err)
	}
	if !modified {
		return fmt.Errorf("scalar %q not found in file", edit.OldValue)
	}

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return fmt.Errorf("cannot marshal YAML: %w", err)
	}

	if p.DryRun {
		return nil
	}

	tmpFile, err := writeTemporary(srcFile, out)
	if err != nil {
		return fmt.Errorf("cannot write temporary file: %w", err)
	}

	if !p.NoBackup {
		backupPath := srcFile + p.BackupSuffix
		if _, err := os.Stat(backupPath); err == nil {
			os.Remove(tmpFile)
			return fmt.Errorf("backup already exists: %s", backupPath)
		}
		if err := copyFile(srcFile, backupPath); err != nil {
			os.Remove(tmpFile)
			return fmt.Errorf("cannot create backup: %w", err)
		}
	}

	if err := os.Rename(tmpFile, srcFile); err != nil {
		if !p.NoBackup {
			os.Remove(srcFile + p.BackupSuffix)
		}
		return fmt.Errorf("cannot replace file: %w", err)
	}

	return nil
}

func findAndReplaceScalar(doc *yaml.Node, oldValue, newValue string) (bool, error) {
	return findAndReplaceInNode(doc, oldValue, newValue)
}

func findAndReplaceInNode(node *yaml.Node, oldValue, newValue string) (bool, error) {
	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			if ok, err := findAndReplaceInNode(child, oldValue, newValue); err != nil {
				return false, err
			} else if ok {
				return true, nil
			}
		}
	case yaml.MappingNode:
		for i := 0; i < len(node.Content)-1; i += 2 {
			if node.Content[i].Value == "ports" {
				if ok, err := findAndReplaceInNode(node.Content[i+1], oldValue, newValue); err != nil {
					return false, err
				} else if ok {
					return true, nil
				}
			}
		}
		for i := 0; i < len(node.Content)-1; i += 2 {
			if ok, err := findAndReplaceInNode(node.Content[i+1], oldValue, newValue); err != nil {
				return false, err
			} else if ok {
				return true, nil
			}
		}
	case yaml.SequenceNode:
		for _, child := range node.Content {
			if child.Kind == yaml.ScalarNode && child.Value == oldValue {
				child.Value = newValue
				return true, nil
			}
		}
		for _, child := range node.Content {
			if ok, err := findAndReplaceInNode(child, oldValue, newValue); err != nil {
				return false, err
			} else if ok {
				return true, nil
			}
		}
	}
	return false, nil
}

func writeTemporary(originalPath string, data []byte) (string, error) {
	dir := filepath.Dir(originalPath)
	tmpFile := filepath.Join(dir, ".port-registry-"+filepath.Base(originalPath)+".tmp")
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return "", err
	}
	return tmpFile, nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}

func isWritable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().Perm()&0222 != 0
}

func RestoreBackups(backupSuffix string, filePaths []string) error {
	var lastErr error
	for _, fp := range filePaths {
		backupPath := fp + backupSuffix
		if _, err := os.Stat(backupPath); err != nil {
			lastErr = fmt.Errorf("backup not found: %s", backupPath)
			continue
		}
		if err := os.Rename(backupPath, fp); err != nil {
			lastErr = fmt.Errorf("cannot restore %s: %w", fp, err)
		}
	}
	return lastErr
}

func editBefore(a, b Edit) bool {
	if a.Binding.ProjectID != b.Binding.ProjectID {
		return a.Binding.ProjectID < b.Binding.ProjectID
	}
	if a.Binding.Service != b.Binding.Service {
		return a.Binding.Service < b.Binding.Service
	}
	return a.Binding.Source.File < b.Binding.Source.File
}
