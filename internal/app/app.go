package app

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/pyralis-labs/compose-port-registry/internal/allocate"
	"github.com/pyralis-labs/compose-port-registry/internal/collision"
	"github.com/pyralis-labs/compose-port-registry/internal/compose"
	"github.com/pyralis-labs/compose-port-registry/internal/config"
	"github.com/pyralis-labs/compose-port-registry/internal/fix"
	"github.com/pyralis-labs/compose-port-registry/internal/model"
	"github.com/pyralis-labs/compose-port-registry/internal/report"
)

type App struct {
	Config *config.Config
	Env    map[string]string
	Stdout *os.File
	Stderr *os.File
}

func New(cfg *config.Config) *App {
	return &App{
		Config: cfg,
		Env:    captureEnv(),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

func (a *App) Run(ctx context.Context) int {
	if a.Config.ShowVersion {
		fmt.Fprintf(a.Stdout, "compose-port-registry %s\n", model.Version)
		return 0
	}

	if err := config.Validate(a.Config); err != nil {
		fmt.Fprintf(a.Stderr, "ERROR: %v\n", err)
		return config.MapExitCode(model.ErrInvalidConfig)
	}

	projects, allBindings, warnings, err := a.scanProjects(ctx)
	if err != nil {
		typedErr, ok := err.(*model.TypedError)
		if ok {
			fmt.Fprintf(a.Stderr, "ERROR: %v\n", typedErr)
			return config.MapExitCode(typedErr.Type)
		}
		fmt.Fprintf(a.Stderr, "ERROR: %v\n", err)
		return config.MapExitCode(model.ErrInternal)
	}

	collisionEngine := collision.NewEngine(allBindings)
	collisions := collisionEngine.Detect()

	var fixes []model.Fix

	if a.Config.Suggest || a.Config.Fix {
		allocator := allocate.NewAllocator(
			a.Config.Range.Start,
			a.Config.Range.End,
			allBindings,
		)
		allocResults := allocator.Allocate(collisions)

		planner := fix.NewPlanner(
			a.Config.BackupSuffix,
			a.Config.NoBackup,
			a.Config.DryRun,
		)

		fixAllocations := make([]fix.AllocationResult, 0, len(allocResults))
		for _, ar := range allocResults {
			far := fix.AllocationResult{
				Binding:   ar.Binding,
				Suggested: ar.Suggested,
				Exhausted: ar.Exhausted,
			}
			fixAllocations = append(fixAllocations, far)
		}

		fixPlan := planner.Plan(allBindings, fixAllocations)

		var err error
		yes := a.Config.Yes
		if !yes && a.Config.Fix && !a.Config.DryRun {
			fmt.Fprintf(a.Stderr, "WARNING: --fix requires --yes for non-interactive mutation\n")
			return config.MapExitCode(model.ErrFixRefused)
		}

		fixes, err = planner.Execute(fixPlan, yes && a.Config.Fix && !a.Config.DryRun)
		if err != nil {
			fmt.Fprintf(a.Stderr, "ERROR: fix execution failed: %v\n", err)
			return config.MapExitCode(model.ErrTransaction)
		}

		if a.Config.Fix && !a.Config.DryRun {
			for _, f := range fixes {
				if f.Status == model.FixRolledBack {
					fixPaths := make([]string, 0)
					for _, p := range projects {
						fixPaths = append(fixPaths, p.Files...)
					}
					if err := fix.RestoreBackups(a.Config.BackupSuffix, fixPaths); err != nil {
						fmt.Fprintf(a.Stderr, "ERROR: rollback failed: %v\n", err)
					}
					return config.MapExitCode(model.ErrTransaction)
				}
			}
		}
	}

	r := report.BuildReport(projects, collisions, warnings, fixes, a.Config.Roots)

	if len(collisions) > 0 {
		r.Summary.FixesApplied = model.CountFixesByStatus(fixes, model.FixApplied)
	}

	if err := report.RenderReport(a.Stdout, r, a.Config.Format); err != nil {
		fmt.Fprintf(a.Stderr, "ERROR: report rendering failed: %v\n", err)
		return config.MapExitCode(model.ErrInternal)
	}

	if len(collisions) > 0 {
		return config.MapExitCode(model.ErrCollision)
	}

	for _, f := range fixes {
		if f.Status == model.FixRefused || f.Status == model.FixRolledBack {
			return config.MapExitCode(model.ErrFixRefused)
		}
	}

	return config.MapExitCode(model.ErrCollision)
}

func (a *App) scanProjects(ctx context.Context) ([]model.Project, []model.Binding, []model.Warning, error) {
	var allProjects []model.Project
	var allBindings []model.Binding
	var allWarnings []model.Warning

	if len(a.Config.Files) > 0 {
		proj, bindings, warnings, err := a.loadExplicitProject(ctx)
		if err != nil {
			return nil, nil, nil, err
		}
		allProjects = append(allProjects, proj)
		allBindings = append(allBindings, bindings...)
		allWarnings = append(allWarnings, warnings...)
	} else {
		discovered, discWarnings, err := compose.DiscoverProjects(a.Config.Roots, a.Config.Excludes)
		if err != nil {
			return nil, nil, nil, err
		}
		allWarnings = append(allWarnings, discWarnings...)

		for _, dp := range discovered {
			proj, bindings, warnings, err := a.loadDiscoveredProject(ctx, dp)
			if err != nil {
				allWarnings = append(allWarnings, model.Warning{
					Message: fmt.Sprintf("failed to load project %s: %v", dp.Dir, err),
				})
				continue
			}
			allProjects = append(allProjects, proj)
			allBindings = append(allBindings, bindings...)
			allWarnings = append(allWarnings, warnings...)
		}
	}

	model.SortBindings(allBindings)

	sort.Slice(allProjects, func(i, j int) bool {
		return allProjects[i].ID < allProjects[j].ID
	})

	return allProjects, allBindings, allWarnings, nil
}

func (a *App) loadExplicitProject(ctx context.Context) (model.Project, []model.Binding, []model.Warning, error) {
	opts := compose.LoadOptions{
		ConfigFiles: a.Config.Files,
		ProjectDir:  a.Config.ProjectDir,
		EnvFiles:    a.Config.EnvFiles,
		Profiles:    a.Config.Profiles,
		Strict:      a.Config.Strict,
		Env:         a.Env,
	}

	result, err := compose.LoadProject(ctx, opts)
	if err != nil {
		return model.Project{}, nil, nil, err
	}

	bindings, loadWarnings := compose.NormalizePorts(
		result.Project.Services,
		result.Project.Name,
		result.Project.Files,
	)

	project := model.Project{
		ID:        result.Project.Name,
		Name:      result.Project.Name,
		Directory: result.Project.Directory,
		Files:     result.Project.Files,
		Bindings:  bindings,
	}

	return project, bindings, append(result.Warnings, loadWarnings...), nil
}

func (a *App) loadDiscoveredProject(ctx context.Context, dp compose.DiscoveredProject) (model.Project, []model.Binding, []model.Warning, error) {
	opts := compose.LoadOptions{
		ConfigFiles: []string{dp.File},
		ProjectDir:  dp.Dir,
		EnvFiles:    a.Config.EnvFiles,
		Profiles:    a.Config.Profiles,
		Strict:      a.Config.Strict,
		Env:         a.Env,
	}

	result, err := compose.LoadProject(ctx, opts)
	if err != nil {
		return model.Project{}, nil, nil, err
	}

	bindings, loadWarnings := compose.NormalizePorts(
		result.Project.Services,
		result.Project.Name,
		result.Project.Files,
	)

	projectID := result.Project.Name
	if projectID == "" {
		projectID = dp.Dir
	}

	project := model.Project{
		ID:        projectID,
		Name:      result.Project.Name,
		Directory: result.Project.Directory,
		Files:     result.Project.Files,
		Bindings:  bindings,
	}

	return project, bindings, append(result.Warnings, loadWarnings...), nil
}

func captureEnv() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				env[e[:i]] = e[i+1:]
				break
			}
		}
	}
	return env
}
