package compose

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pyralis-labs/compose-port-registry/internal/model"
)

const maxWalkDepth = 20

var conventionalFilenames = []string{
	"compose.yaml",
	"compose.yml",
	"docker-compose.yaml",
	"docker-compose.yml",
}

type DiscoveredProject struct {
	Dir  string
	File string
}

func DiscoverProjects(roots []string, excludes []string) ([]DiscoveredProject, []model.Warning, error) {
	var projects []DiscoveredProject
	var warnings []model.Warning
	seen := make(map[string]bool)

	excludeSet := buildExcludeSet(excludes)

	for _, root := range roots {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return nil, warnings, model.NewPathError("discover", root, model.ErrDiscovery, err)
		}
		info, err := os.Stat(absRoot)
		if err != nil {
			return nil, warnings, model.NewPathError("discover", absRoot, model.ErrDiscovery, err)
		}
		if !info.IsDir() {
			return nil, warnings, model.NewPathError("discover", absRoot, model.ErrDiscovery, os.ErrInvalid)
		}

		startDepth := strings.Count(absRoot, string(filepath.Separator))
		err = filepath.Walk(absRoot, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			depth := strings.Count(path, string(filepath.Separator)) - startDepth
			if depth > maxWalkDepth {
				return filepath.SkipDir
			}
			if fi.IsDir() {
				base := filepath.Base(path)
				if shouldSkipDir(base) || excludeSet[path] {
					return filepath.SkipDir
				}
				return nil
			}
			fname := filepath.Base(path)
			if !isConventionalFile(fname) {
				return nil
			}
			dir := filepath.Dir(path)
			if seen[dir] {
				warnings = append(warnings, model.Warning{
					Message: "multiple conventional compose files in " + dir + "; use explicit --file to disambiguate",
				})
				return nil
			}
			seen[dir] = true
			projects = append(projects, DiscoveredProject{Dir: dir, File: path})
			return nil
		})
		if err != nil {
			return nil, warnings, model.NewPathError("discover", absRoot, model.ErrDiscovery, err)
		}
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].File < projects[j].File
	})

	return projects, warnings, nil
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", ".svn", ".hg", "node_modules", "vendor", ".tox", "__pycache__":
		return true
	default:
		return strings.HasPrefix(name, ".")
	}
}

func isConventionalFile(name string) bool {
	for _, f := range conventionalFilenames {
		if name == f {
			return true
		}
	}
	return false
}

func buildExcludeSet(excludes []string) map[string]bool {
	set := make(map[string]bool)
	for _, e := range excludes {
		if abs, err := filepath.Abs(e); err == nil {
			set[abs] = true
		}
	}
	return set
}

func ResolveOverrides(projectDir string, files []string) []string {
	if len(files) == 0 {
		return nil
	}
	resolved := make([]string, len(files))
	for i, f := range files {
		if filepath.IsAbs(f) {
			resolved[i] = f
		} else {
			resolved[i] = filepath.Join(projectDir, f)
		}
	}
	return resolved
}
