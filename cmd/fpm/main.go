package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"awesomeProject/internal/sbpm"
)

func main() {
	// Basic subcommand parser
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	sub := os.Args[1]
	switch sub {
	case "detect":
		cmdDetect(os.Args[2:])
	case "plan":
		cmdPlan(os.Args[2:])
	case "install":
		cmdInstall(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n\n", sub)
		usage()
		os.Exit(2)
	}
}

func baseFlags(fs *flag.FlagSet) (verbose *bool, dryRun *bool, timeout *time.Duration, platformOverride *string) {
	verbose = fs.Bool("v", false, "verbose output")
	dryRun = fs.Bool("dry-run", false, "print actions without executing")
	timeout = fs.Duration("timeout", 15*time.Minute, "command timeout")
	platformOverride = fs.String("platform", "", "override runtime platform (windows|linux|macos)")
	return
}

func cmdDetect(args []string) {
	fs := flag.NewFlagSet("detect", flag.ExitOnError)
	verbose, _, _, platformOverride := baseFlags(fs)
	_ = fs.Parse(args)
	logger := setupLogger(*verbose)
	plat := sbpm.DetectPlatform(*platformOverride)
	logger.Info("detected platform", "os", plat.OS, "arch", plat.Arch)
	fmt.Printf("%s/%s\n", plat.OS, plat.Arch)
}

func cmdPlan(args []string) {
	fs := flag.NewFlagSet("plan", flag.ExitOnError)
	verbose, _, timeout, platformOverride := baseFlags(fs)
	configPath := fs.String("config", "", "path to configuration JSON (optional)")
	_ = fs.Parse(args)
	logger := setupLogger(*verbose)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	exec := &sbpm.DefaultExecutor{}
	plat := sbpm.DetectPlatform(*platformOverride)
	planner := sbpm.NewPlanner(exec, plat, logger)
	var cfg *sbpm.Config
	var err error
	if strings.TrimSpace(*configPath) != "" {
		cfg, err = sbpm.LoadConfig(*configPath)
		if err != nil {
			logger.Error("failed to load config", "err", err)
			os.Exit(1)
		}
	}
	plan, err := planner.Plan(ctx, cfg)
	if err != nil {
		logger.Error("planning failed", "err", err)
		os.Exit(1)
	}
	fmt.Println(plan.String())
}

func cmdInstall(args []string) {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	verbose, dryRun, timeout, platformOverride := baseFlags(fs)
	configPath := fs.String("config", "", "path to configuration JSON (optional)")
	_ = fs.Parse(args)
	logger := setupLogger(*verbose)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	exec := &sbpm.DefaultExecutor{}
	plat := sbpm.DetectPlatform(*platformOverride)
	planner := sbpm.NewPlanner(exec, plat, logger)
	var cfg *sbpm.Config
	var err error
	if strings.TrimSpace(*configPath) != "" {
		cfg, err = sbpm.LoadConfig(*configPath)
		if err != nil {
			logger.Error("failed to load config", "err", err)
			os.Exit(1)
		}
	}
	plan, err := planner.Plan(ctx, cfg)
	if err != nil {
		logger.Error("planning failed", "err", err)
		os.Exit(1)
	}
	if *dryRun {
		fmt.Println(plan.String())
		return
	}
	res := planner.Execute(ctx, plan)
	if res.Err != nil {
		logger.Error("execution failed", "err", res.Err)
		os.Exit(1)
	}
	logger.Info("execution completed", "actions", len(plan.Actions))
}

func usage() {
	fmt.Fprintf(os.Stderr, `fpm - Flutter Package Manager helper (skeleton)

Usage:
  fpm <subcommand> [options]

Subcommands:
  detect      Detect current platform
  plan        Produce an action plan (no changes)
  install     Execute the plan (use --dry-run to preview)

Global flags:
  -v              verbose
  --dry-run       print actions without executing (install)
  --timeout dur   timeout for operations
  --platform str  override runtime platform (windows|linux|macos)
`)
}

func setupLogger(verbose bool) *slog.Logger {
	lvl := new(slog.LevelVar)
	if verbose {
		lvl.Set(slog.LevelDebug)
	} else {
		lvl.Set(slog.LevelInfo)
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))
}
