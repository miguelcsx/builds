package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"builds/internal/analysis/performance"
	"builds/internal/collectors/compiler"
	"builds/internal/collectors/environment"
	"builds/internal/collectors/hardware"
	"builds/internal/collectors/kernel"
	"builds/internal/collectors/remarks"
	"builds/internal/collectors/resource"
	"builds/internal/models"
	"builds/internal/reporters"
	"builds/pkg/config"

	"github.com/google/uuid"
)

var (
	configFile = flag.String("config", "", "Path to configuration file")
	outputDir  = flag.String("output", "", "Output directory for reports")
	format     = flag.String("format", "txt", "Report format (txt, json, or html)")
	verbose    = flag.Bool("verbose", false, "Enable verbose output")
	version    = flag.Bool("version", false, "Show version information")
)

const buildVersion = "0.1.0"

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <build command>\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nExample:")
		fmt.Fprintln(os.Stderr, "  builds -format json clang -O2 -g -fopenmp test.c")
	}

	flag.Parse()

	if *version {
		fmt.Printf("builds version %s\n", buildVersion)
		return
	}

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	// Load configuration
	cfg, err := loadConfig(*configFile)
	if err != nil {
		log.Printf("Warning: using default configuration: %v", err)
		cfg = config.DefaultConfig()
	}

	// Create build context
	buildCtx := createBuildContext(cfg, flag.Args())
	if buildCtx == nil {
		log.Printf("Error: No build command provided")
		flag.Usage()
		os.Exit(1)
	}

	// Execute build and collect information
	build, err := executeBuild(buildCtx, flag.Args())

	// Analyze results
	analyzer := performance.NewAnalyzer(build)
	analysis, err := analyzer.Analyze()
	if err != nil {
		log.Printf("Analysis failed: %v", err)
	}

	// Generate reports
	outDir := *outputDir
	if outDir == "" {
		outDir = cfg.ReportDir
	}

	reporter, err := reporters.NewReporter(reporters.Options{
		OutputDir: outDir,
		Format:    *format,
		Build:     build,
		Analysis:  analysis,
	})
	if err != nil {
		log.Printf("Failed to create reporter: %v", err)
		os.Exit(1)
	}

	if err := reporter.Generate(); err != nil {
		log.Printf("Failed to generate report: %v", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("Build completed. Reports generated in: %s\n", outDir)
	}
}

func loadConfig(path string) (*config.Config, error) {
	if path == "" {
		return config.DefaultConfig(), nil
	}
	return config.LoadConfig(path)
}

func createBuildContext(cfg *config.Config, args []string) *models.BuildContext {
	if len(args) == 0 {
		return nil
	}

	return &models.BuildContext{
		Context:   context.Background(),
		BuildID:   uuid.New().String(),
		OutputDir: cfg.BuildDir,
		Compiler:  args[0],  // First argument is the compiler
		Args:      args[1:], // Rest of the arguments
		Config: &models.CollectorConfig{
			Enabled:     true,
			Timeout:     300,
			MaxAttempts: 3,
		},
	}
}

func executeBuild(buildCtx *models.BuildContext, args []string) (*models.Build, error) {
	build := &models.Build{
		ID:        buildCtx.BuildID,
		StartTime: time.Now(),
		Command: models.Command{
			Executable: buildCtx.Compiler,
			Arguments:  buildCtx.Args,
			WorkingDir: buildCtx.OutputDir,
		},
	}

	// Initialize collectors
	factory := models.NewCollectorFactory()
	setupCollectors(factory, buildCtx)

	// Create command with enhanced flags for collection
	cmdArgs := enhanceBuildFlags(buildCtx.Args)
	cmd := exec.CommandContext(buildCtx.Context, buildCtx.Compiler, cmdArgs...)

	// Capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return build, fmt.Errorf("getting stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return build, fmt.Errorf("getting stderr pipe: %w", err)
	}

	// Start collectors
	for name, collector := range factory.GetCollectors() {
		if err := collector.Initialize(buildCtx.Context); err != nil {
			log.Printf("Warning: failed to initialize %s collector: %v", name, err)
			continue
		}
	}

	// Start build
	if err := cmd.Start(); err != nil {
		return build, fmt.Errorf("starting build: %w", err)
	}

	// Setup output collection
	var stdoutBuf, stderrBuf bytes.Buffer
	go io.Copy(&stdoutBuf, stdout)
	go io.Copy(&stderrBuf, stderr)

	// Wait for build to complete
	err = cmd.Wait()
	build.EndTime = time.Now()
	build.Duration = build.EndTime.Sub(build.StartTime).Seconds()
	build.Success = err == nil
	if !build.Success {
		build.Error = err.Error()
	}

	// Store the build output
	build.Output = models.Output{
		Stdout:    stdoutBuf.String(),
		Stderr:    stderrBuf.String(),
		ExitCode:  int32(cmd.ProcessState.ExitCode()),
		Artifacts: []models.Artifact{}, // Will be populated by collectors
	}

	// Run collectors
	for name, collector := range factory.GetCollectors() {
		if err := collector.Collect(buildCtx.Context); err != nil {
			log.Printf("Warning: collection failed for %s: %v", name, err)
			continue
		}

		// Store collected data in build info
		if data := collector.GetData(); data != nil {
			storeBuildData(build, name, data)
		}
	}

	return build, err
}

func setupCollectors(factory *models.CollectorFactory, ctx *models.BuildContext) {
	factory.RegisterCollector("environment", environment.NewCollector())
	factory.RegisterCollector("hardware", hardware.NewCollector())
	factory.RegisterCollector("compiler", compiler.NewCollector(ctx))
	factory.RegisterCollector("kernel", kernel.NewCollector(ctx, os.Stderr))
	factory.RegisterCollector("remarks", remarks.NewCollector(ctx))
	factory.RegisterCollector("resource", resource.NewCollector(ctx))
}

func enhanceBuildFlags(args []string) []string {
	enhanced := make([]string, len(args))
	copy(enhanced, args)

	// Add flags for remark collection
	if !hasFlag(enhanced, "-Rpass=.*") {
		enhanced = append(enhanced, "-Rpass=.*")
	}
	if !hasFlag(enhanced, "-Rpass-missed=.*") {
		enhanced = append(enhanced, "-Rpass-missed=.*")
	}
	if !hasFlag(enhanced, "-Rpass-analysis=.*") {
		enhanced = append(enhanced, "-Rpass-analysis=.*")
	}

	// Disable optimization record files
	if !hasFlag(enhanced, "-fno-save-optimization-record") {
		enhanced = append(enhanced, "-fno-save-optimization-record")
	}

	enhanced = removeFlag(enhanced, "-fsave-optimization-record")
	return enhanced
}

func removeFlag(args []string, flag string) []string {
	result := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] != flag {
			result = append(result, args[i])
		}
	}
	return result
}

func hasFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag || strings.HasPrefix(arg, flag) {
			return true
		}
	}
	return false
}

func storeBuildData(build *models.Build, collectorName string, data interface{}) {
	switch collectorName {
	case "environment":
		if info, ok := data.(models.Environment); ok {
			build.Environment = info
		}
	case "hardware":
		if info, ok := data.(models.Hardware); ok {
			build.Hardware = info
		}
	case "compiler":
		if info, ok := data.(models.Compiler); ok {
			build.Compiler = info
		}
	case "kernel", "remarks":
		if remarks, ok := data.([]models.CompilerRemark); ok {
			build.Remarks = append(build.Remarks, remarks...)
		}
	case "resource":
		if info, ok := data.(models.ResourceUsage); ok {
			build.ResourceUsage = info
		}
	}
}
