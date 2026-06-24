package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"

	"github.com/pyralis-labs/compose-port-registry/internal/model"
)

const maxFileSize = 10 * 1024 * 1024 // 10 MiB

type LoadOptions struct {
	ConfigFiles []string
	ProjectDir  string
	EnvFiles    []string
	Profiles    []string
	Strict      bool
	Env         map[string]string
}

type LoadResult struct {
	Project  *ProjectData
	Warnings []model.Warning
}

type ProjectData struct {
	Name      string
	Directory string
	Files     []string
	Services  map[string]types.ServiceConfig
}

func LoadProject(ctx context.Context, opts LoadOptions) (*LoadResult, error) {
	projectDir := opts.ProjectDir
	if projectDir == "" && len(opts.ConfigFiles) > 0 {
		projectDir = filepath.Dir(opts.ConfigFiles[0])
	}
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, model.NewPathError("load", projectDir, model.ErrLoad, err)
	}

	envMap := buildEnvMap(opts.Env, opts.EnvFiles, absDir)

	composeFiles := opts.ConfigFiles
	if len(composeFiles) == 0 {
		if f, err := findDefaultFile(absDir); err == nil {
			composeFiles = []string{f}
		} else {
			return nil, model.NewPathError("load", absDir, model.ErrLoad, fmt.Errorf("no compose file found in %s", absDir))
		}
	}

	absFiles := resolveAbsoluteFiles(composeFiles, projectDir)

	composeProject, err := loadComposeFiles(ctx, absFiles, envMap, opts.Profiles)
	if err != nil {
		return nil, model.NewPathError("load", strings.Join(absFiles, ", "), model.ErrLoad, err)
	}

	var warnings []model.Warning
	for sname, svc := range composeProject.Services {
		for _, port := range svc.Ports {
			if port.HostIP != "" {
				info := model.ParseHostScope(port.HostIP)
				if info.Scope == model.HostUnresolved {
					warnings = append(warnings, model.Warning{
						Message: fmt.Sprintf("service %q: unresolvable host IP %q", sname, port.HostIP),
					})
				}
			}
		}
	}

	services := make(map[string]types.ServiceConfig)
	for k, v := range composeProject.Services {
		services[k] = v
	}

	return &LoadResult{
		Project: &ProjectData{
			Name:      composeProject.Name,
			Directory: absDir,
			Files:     absFiles,
			Services:  services,
		},
		Warnings: warnings,
	}, nil
}

func findDefaultFile(dir string) (string, error) {
	for _, name := range conventionalFilenames {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", os.ErrNotExist
}

func resolveAbsoluteFiles(files []string, baseDir string) []string {
	abs := make([]string, len(files))
	for i, f := range files {
		if filepath.IsAbs(f) {
			abs[i] = f
		} else {
			abs[i] = filepath.Join(baseDir, f)
		}
	}
	return abs
}

func loadComposeFiles(ctx context.Context, files []string, env map[string]string, profiles []string) (*types.Project, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("no compose files specified")
	}

	for _, f := range files {
		info, err := os.Stat(f)
		if err == nil && info.Size() > maxFileSize {
			return nil, fmt.Errorf("compose file %s exceeds maximum size of 10 MiB (%d bytes)", f, info.Size())
		}
	}

	absDir := filepath.Dir(files[0])

	envMapping := make(types.Mapping)
	for k, v := range env {
		envMapping[k] = v
	}

	cfg := types.ConfigDetails{
		ConfigFiles: make([]types.ConfigFile, len(files)),
		Environment: envMapping,
		WorkingDir:  absDir,
	}
	for i, f := range files {
		cfg.ConfigFiles[i] = types.ConfigFile{Filename: f}
	}

	proj, err := loader.LoadWithContext(ctx, cfg, func(options *loader.Options) {
		options.SkipConsistencyCheck = true
		options.SetProjectName(filepath.Base(absDir), false)
		if len(profiles) > 0 {
			options.Profiles = profiles
		}
	})
	if err != nil {
		return nil, err
	}

	return proj, nil
}

func buildEnvMap(env map[string]string, envFiles []string, projectDir string) map[string]string {
	result := make(map[string]string)

	for k, v := range env {
		result[k] = v
	}

	if len(envFiles) == 0 {
		defaultEnv := filepath.Join(projectDir, ".env")
		if data, err := os.ReadFile(defaultEnv); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				if k, v, ok := strings.Cut(line, "="); ok {
					if _, exists := result[k]; !exists {
						result[strings.TrimSpace(k)] = strings.TrimSpace(v)
					}
				}
			}
		}
	} else {
		for _, ef := range envFiles {
			data, err := os.ReadFile(ef)
			if err != nil {
				continue
			}
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				if k, v, ok := strings.Cut(line, "="); ok {
					result[strings.TrimSpace(k)] = strings.TrimSpace(v)
				}
			}
		}
	}

	return result
}

func NormalizePorts(services map[string]types.ServiceConfig, projectName string, files []string) ([]model.Binding, []model.Warning) {
	var bindings []model.Binding
	var warnings []model.Warning

	for svcName, svc := range services {
		for _, port := range svc.Ports {
			binding, warn := portToBinding(port, svcName, projectName, files)
			if warn != nil {
				warnings = append(warnings, *warn)
			}
			bindings = append(bindings, binding)
		}
	}

	model.SortBindings(bindings)
	return bindings, warnings
}

func parsePublished(published string) (model.Interval, error) {
	if published == "" {
		return model.Interval{Start: 0, End: 0}, nil
	}
	if strings.Contains(published, "-") {
		parts := strings.SplitN(published, "-", 2)
		start, err1 := strconv.ParseUint(parts[0], 10, 16)
		end, err2 := strconv.ParseUint(parts[1], 10, 16)
		if err1 != nil || err2 != nil || start == 0 || end == 0 {
			return model.Interval{}, fmt.Errorf("invalid published range: %s", published)
		}
		return model.NewInterval(uint16(start), uint16(end))
	}
	port, err := strconv.ParseUint(published, 10, 16)
	if err != nil || port == 0 {
		return model.Interval{}, fmt.Errorf("invalid published port: %s", published)
	}
	return model.IntervalFromPort(uint16(port)), nil
}

func portToBinding(port types.ServicePortConfig, svcName, projectName string, files []string) (model.Binding, *model.Warning) {
	protocol := model.ProtocolTCP
	if strings.EqualFold(port.Protocol, "udp") {
		protocol = model.ProtocolUDP
	}

	hostIP := model.ParseHostScope(port.HostIP)

	publishedInterval, err := parsePublished(port.Published)
	if err != nil {
		return model.Binding{
				ProjectID:  projectName,
				Service:    svcName,
				Protocol:   protocol,
				HostIP:     hostIP,
				Published:  model.Interval{Start: 0, End: 0},
				Target:     model.IntervalFromPort(uint16(port.Target)),
				Source:     model.SourceRef{File: sourceFile(files)},
				Mutability: model.MutableInterpolation,
			}, &model.Warning{
				Message: fmt.Sprintf("service %q: invalid published port %q: %v", svcName, port.Published, err),
			}
	}

	mutability := model.Mutable
	if strings.Contains(port.Published, "-") {
		mutability = model.MutableRange
	} else if strings.Contains(port.Published, "${") || strings.Contains(port.Published, "$") {
		mutability = model.MutableInterpolation
	}

	return model.Binding{
		ProjectID:  projectName,
		Service:    svcName,
		Protocol:   protocol,
		HostIP:     hostIP,
		Published:  publishedInterval,
		Target:     model.IntervalFromPort(uint16(port.Target)),
		Source:     model.SourceRef{File: sourceFile(files)},
		Mutability: mutability,
	}, nil
}

func sourceFile(files []string) string {
	if len(files) > 0 {
		return files[0]
	}
	return ""
}
