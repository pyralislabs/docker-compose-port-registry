package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/pyralis-labs/compose-port-registry/internal/app"
	"github.com/pyralis-labs/compose-port-registry/internal/config"
	"github.com/pyralis-labs/compose-port-registry/internal/model"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	cfg, versionOut, err := parseFlags(args, stderr)
	if err != nil {
		return config.MapExitCode(model.ErrInvalidConfig)
	}
	if versionOut != "" {
		fmt.Fprint(stdout, versionOut)
		return 0
	}

	application := app.New(cfg)
	application.Stdout = stdout
	application.Stderr = stderr

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	return application.Run(ctx)
}

func parseFlags(args []string, stderr io.Writer) (*config.Config, string, error) {
	cfg := config.DefaultConfig()

	fs := flag.NewFlagSet("compose-port-registry", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var showVersion bool
	var portRange string

	fs.BoolVar(&showVersion, "version", false, "print version")
	fs.StringVar(&portRange, "range", "4000-4999", "allocation port range (START-END)")
	fs.StringVar(&cfg.Format, "format", "text", "output format (text or json)")
	fs.BoolVar(&cfg.Suggest, "suggest", false, "include deterministic replacement suggestions")
	fs.BoolVar(&cfg.Fix, "fix", false, "apply supported suggestions")
	fs.BoolVar(&cfg.DryRun, "dry-run", false, "validate and report the edit plan without committing")
	fs.BoolVar(&cfg.Yes, "yes", false, "acknowledge mutation without interactive prompt")
	fs.BoolVar(&cfg.NoBackup, "no-backup", false, "disable backups; requires --yes")
	fs.BoolVar(&cfg.Strict, "strict", false, "treat unsupported constructs as errors")
	fs.StringVar(&cfg.BackupSuffix, "backup-suffix", ".port-registry.bak", "backup file suffix")
	fs.StringVar(&cfg.ProjectDir, "project-dir", "", "compose project directory for explicit files")

	var files multiFlag
	var envFiles multiFlag
	var profiles multiFlag
	var excludes multiFlag

	fs.Var(&files, "file", "explicit compose file; repeatable")
	fs.Var(&envFiles, "env-file", "interpolation environment file; repeatable")
	fs.Var(&profiles, "profile", "active compose profile; repeatable")
	fs.Var(&excludes, "exclude", "exclude path glob; repeatable")

	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: compose-port-registry [ROOT...] [flags]\n")
		fmt.Fprintf(stderr, "       compose-port-registry --file PATH [--file PATH...] [flags]\n")
		fmt.Fprintf(stderr, "\nFlags:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return nil, "", err
	}

	if showVersion {
		cfg.ShowVersion = true
		return cfg, fmt.Sprintf("compose-port-registry %s\n", model.Version), nil
	}

	if portRange != "" {
		pr, err := config.ParsePortRange(portRange)
		if err != nil {
			fmt.Fprintf(stderr, "ERROR: invalid --range: %v\n", err)
			return nil, "", err
		}
		cfg.Range = pr
	}

	cfg.Files = files
	cfg.EnvFiles = envFiles
	cfg.Profiles = profiles
	cfg.Excludes = excludes
	cfg.Roots = fs.Args()

	return cfg, "", nil
}

type multiFlag []string

func (m *multiFlag) String() string {
	return fmt.Sprintf("%v", *m)
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}
