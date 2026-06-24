package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/pyralis-labs/compose-port-registry/internal/app"
	"github.com/pyralis-labs/compose-port-registry/internal/config"
	"github.com/pyralis-labs/compose-port-registry/internal/model"
)

func main() {
	os.Exit(run())
}

func run() int {
	cfg := parseFlags()

	application := app.New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	return application.Run(ctx)
}

func parseFlags() *config.Config {
	cfg := config.DefaultConfig()

	var showVersion bool
	var portRange string

	flag.BoolVar(&showVersion, "version", false, "print version")
	flag.StringVar(&portRange, "range", "4000-4999", "allocation port range (START-END)")
	flag.StringVar(&cfg.Format, "format", "text", "output format (text or json)")
	flag.BoolVar(&cfg.Suggest, "suggest", false, "include deterministic replacement suggestions")
	flag.BoolVar(&cfg.Fix, "fix", false, "apply supported suggestions")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "validate and report the edit plan without committing")
	flag.BoolVar(&cfg.Yes, "yes", false, "acknowledge mutation without interactive prompt")
	flag.BoolVar(&cfg.NoBackup, "no-backup", false, "disable backups; requires --yes")
	flag.BoolVar(&cfg.Strict, "strict", false, "treat unsupported constructs as errors")
	flag.StringVar(&cfg.BackupSuffix, "backup-suffix", ".port-registry.bak", "backup file suffix")
	flag.StringVar(&cfg.ProjectDir, "project-dir", "", "compose project directory for explicit files")

	var files multiFlag
	var envFiles multiFlag
	var profiles multiFlag
	var excludes multiFlag

	flag.Var(&files, "file", "explicit compose file; repeatable")
	flag.Var(&envFiles, "env-file", "interpolation environment file; repeatable")
	flag.Var(&profiles, "profile", "active compose profile; repeatable")
	flag.Var(&excludes, "exclude", "exclude path glob; repeatable")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: compose-port-registry [ROOT...] [flags]\n")
		fmt.Fprintf(os.Stderr, "       compose-port-registry --file PATH [--file PATH...] [flags]\n")
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if showVersion {
		cfg.ShowVersion = true
		fmt.Printf("compose-port-registry %s\n", model.Version)
		os.Exit(0)
	}

	if portRange != "" {
		pr, err := config.ParsePortRange(portRange)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: invalid --range: %v\n", err)
			os.Exit(config.MapExitCode(model.ErrInvalidConfig))
		}
		cfg.Range = pr
	}

	cfg.Files = files
	cfg.EnvFiles = envFiles
	cfg.Profiles = profiles
	cfg.Excludes = excludes
	cfg.Roots = flag.Args()

	return cfg
}

type multiFlag []string

func (m *multiFlag) String() string {
	return fmt.Sprintf("%v", *m)
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}
