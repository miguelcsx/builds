package text

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"builds/internal/analysis/performance"
	"builds/internal/models"
)

type Reporter struct {
	build    *models.Build
	analysis *performance.AnalysisResult
	outDir   string
}

func NewReporter(build *models.Build, analysis *performance.AnalysisResult, outDir string) *Reporter {
	return &Reporter{
		build:    build,
		analysis: analysis,
		outDir:   outDir,
	}
}

func (r *Reporter) Generate() error {
	if err := os.MkdirAll(r.outDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	reportPath := filepath.Join(r.outDir, fmt.Sprintf("build-%s.txt", r.build.ID))
	file, err := os.Create(reportPath)
	if err != nil {
		return fmt.Errorf("creating report file: %w", err)
	}
	defer file.Close()

	w := tabwriter.NewWriter(file, 0, 0, 2, ' ', 0)

	// Build Summary
	fmt.Fprintf(w, "Build Report\n")
	fmt.Fprintf(w, "============\n\n")
	fmt.Fprintf(w, "Build ID:\t%s\n", r.build.ID)
	fmt.Fprintf(w, "Status:\t%s\n", r.getStatus())
	fmt.Fprintf(w, "Start Time:\t%s\n", r.build.StartTime.Format(time.RFC3339))
	fmt.Fprintf(w, "End Time:\t%s\n", r.build.EndTime.Format(time.RFC3339))
	fmt.Fprintf(w, "Duration:\t%.2f seconds\n", r.build.Duration)
	if !r.build.Success {
		fmt.Fprintf(w, "Error:\t%s\n", r.build.Error)
	}
	fmt.Fprintf(w, "\n")

	// Environment Information
	fmt.Fprintf(w, "Environment Information\n")
	fmt.Fprintf(w, "=====================\n")
	fmt.Fprintf(w, "Operating System:\t%s\n", r.build.Environment.OS)
	fmt.Fprintf(w, "Architecture:\t%s\n", r.build.Environment.Arch)
	fmt.Fprintf(w, "Working Directory:\t%s\n", r.build.Environment.WorkingDir)
	if len(r.build.Environment.Variables) > 0 {
		fmt.Fprintf(w, "\nEnvironment Variables:\n")
		for k, v := range r.build.Environment.Variables {
			fmt.Fprintf(w, "  %s:\t%s\n", k, v)
		}
	}
	fmt.Fprintf(w, "\n")

	// Hardware Information
	fmt.Fprintf(w, "Hardware Information\n")
	fmt.Fprintf(w, "===================\n")

	// CPU Info
	fmt.Fprintf(w, "CPU:\n")
	fmt.Fprintf(w, "  Model:\t%s\n", r.build.Hardware.CPU.Model)
	fmt.Fprintf(w, "  Vendor:\t%s\n", r.build.Hardware.CPU.Vendor)
	fmt.Fprintf(w, "  Frequency:\t%.2f MHz\n", r.build.Hardware.CPU.Frequency)
	fmt.Fprintf(w, "  Cores:\t%d\n", r.build.Hardware.CPU.Cores)
	fmt.Fprintf(w, "  Threads:\t%d\n", r.build.Hardware.CPU.Threads)
	fmt.Fprintf(w, "  Cache Size:\t%d bytes\n", r.build.Hardware.CPU.CacheSize)

	// Memory Info
	fmt.Fprintf(w, "\nMemory:\n")
	fmt.Fprintf(w, "  Total:\t%s\n", formatBytes(r.build.Hardware.Memory.Total))
	fmt.Fprintf(w, "  Available:\t%s\n", formatBytes(r.build.Hardware.Memory.Available))
	fmt.Fprintf(w, "  Used:\t%s\n", formatBytes(r.build.Hardware.Memory.Used))
	fmt.Fprintf(w, "  Swap Total:\t%s\n", formatBytes(r.build.Hardware.Memory.SwapTotal))
	fmt.Fprintf(w, "  Swap Free:\t%s\n", formatBytes(r.build.Hardware.Memory.SwapFree))

	// GPU Info
	if len(r.build.Hardware.GPUs) > 0 {
		fmt.Fprintf(w, "\nGPUs:\n")
		for i, gpu := range r.build.Hardware.GPUs {
			fmt.Fprintf(w, "  GPU %d:\n", i+1)
			fmt.Fprintf(w, "    Model:\t%s\n", gpu.Model)
			fmt.Fprintf(w, "    Memory:\t%s\n", formatBytes(gpu.Memory))
			fmt.Fprintf(w, "    Driver:\t%s\n", gpu.Driver)
			fmt.Fprintf(w, "    Compute Capabilities:\t%s\n", gpu.ComputeCaps)
		}
	}
	fmt.Fprintf(w, "\n")

	// Compiler Information
	fmt.Fprintf(w, "Compiler Information\n")
	fmt.Fprintf(w, "===================\n")
	fmt.Fprintf(w, "Name:\t%s\n", r.build.Compiler.Name)
	fmt.Fprintf(w, "Version:\t%s\n", r.build.Compiler.Version)
	fmt.Fprintf(w, "Target:\t%s\n", r.build.Compiler.Target)

	// Language Info
	fmt.Fprintf(w, "\nLanguage:\n")
	fmt.Fprintf(w, "  Name:\t%s\n", r.build.Compiler.Language.Name)
	fmt.Fprintf(w, "  Version:\t%s\n", r.build.Compiler.Language.Version)
	fmt.Fprintf(w, "  Specification:\t%s\n", r.build.Compiler.Language.Specification)

	// Compiler Features
	fmt.Fprintf(w, "\nFeatures:\n")
	fmt.Fprintf(w, "  OpenMP Support:\t%v\n", r.build.Compiler.Features.SupportsOpenMP)
	fmt.Fprintf(w, "  GPU Support:\t%v\n", r.build.Compiler.Features.SupportsGPU)
	fmt.Fprintf(w, "  LTO Support:\t%v\n", r.build.Compiler.Features.SupportsLTO)
	fmt.Fprintf(w, "  PGO Support:\t%v\n", r.build.Compiler.Features.SupportsPGO)
	if len(r.build.Compiler.Features.Extensions) > 0 {
		fmt.Fprintf(w, "  Extensions:\t%s\n", strings.Join(r.build.Compiler.Features.Extensions, ", "))
	}

	// Compiler Options and Flags
	if len(r.build.Compiler.Options) > 0 {
		fmt.Fprintf(w, "\nOptions:\t%s\n", strings.Join(r.build.Compiler.Options, " "))
	}
	if len(r.build.Compiler.Optimizations) > 0 {
		fmt.Fprintf(w, "\nOptimizations:\n")
		for opt, enabled := range r.build.Compiler.Optimizations {
			fmt.Fprintf(w, "  %s:\t%v\n", opt, enabled)
		}
	}
	fmt.Fprintf(w, "\n")

	// Command Information
	fmt.Fprintf(w, "Command Information\n")
	fmt.Fprintf(w, "==================\n")
	fmt.Fprintf(w, "Executable:\t%s\n", r.build.Command.Executable)
	fmt.Fprintf(w, "Arguments:\t%s\n", strings.Join(r.build.Command.Arguments, " "))
	fmt.Fprintf(w, "Working Directory:\t%s\n", r.build.Command.WorkingDir)
	fmt.Fprintf(w, "\n")

	// Output Information
	fmt.Fprintf(w, "Output Information\n")
	fmt.Fprintf(w, "=================\n")
	fmt.Fprintf(w, "Exit Code:\t%d\n", r.build.Output.ExitCode)
	if len(r.build.Output.Warnings) > 0 {
		fmt.Fprintf(w, "\nWarnings:\n")
		for _, warning := range r.build.Output.Warnings {
			fmt.Fprintf(w, "  - %s\n", warning)
		}
	}
	if len(r.build.Output.Errors) > 0 {
		fmt.Fprintf(w, "\nErrors:\n")
		for _, err := range r.build.Output.Errors {
			fmt.Fprintf(w, "  - %s\n", err)
		}
	}
	if len(r.build.Output.Artifacts) > 0 {
		fmt.Fprintf(w, "\nArtifacts:\n")
		for _, artifact := range r.build.Output.Artifacts {
			fmt.Fprintf(w, "  - %s\n", artifact.Path)
			fmt.Fprintf(w, "    Type: %s\n", artifact.Type)
			fmt.Fprintf(w, "    Size: %s\n", formatBytes(artifact.Size))
			fmt.Fprintf(w, "    Hash: %s\n", artifact.Hash)
		}
	}
	fmt.Fprintf(w, "\n")

	// Resource Usage Information
	fmt.Fprintf(w, "Resource Usage\n")
	fmt.Fprintf(w, "==============\n")
	fmt.Fprintf(w, "Max Memory:\t%s\n", formatBytes(r.build.ResourceUsage.MaxMemory))
	fmt.Fprintf(w, "CPU Time:\t%.2f seconds\n", r.build.ResourceUsage.CPUTime)
	fmt.Fprintf(w, "Threads:\t%d\n", r.build.ResourceUsage.Threads)

	// IO Statistics
	fmt.Fprintf(w, "\nIO Statistics:\n")
	fmt.Fprintf(w, "  Read:\t%s (%d operations)\n", formatBytes(r.build.ResourceUsage.IO.ReadBytes), r.build.ResourceUsage.IO.ReadCount)
	fmt.Fprintf(w, "  Write:\t%s (%d operations)\n", formatBytes(r.build.ResourceUsage.IO.WriteBytes), r.build.ResourceUsage.IO.WriteCount)
	fmt.Fprintf(w, "\n")

	// Performance Information
	fmt.Fprintf(w, "Performance Information\n")
	fmt.Fprintf(w, "=====================\n")
	fmt.Fprintf(w, "Compile Time:\t%.2f seconds\n", r.build.Performance.CompileTime)
	fmt.Fprintf(w, "Link Time:\t%.2f seconds\n", r.build.Performance.LinkTime)
	fmt.Fprintf(w, "Optimize Time:\t%.2f seconds\n", r.build.Performance.OptimizeTime)

	if len(r.build.Performance.Phases) > 0 {
		fmt.Fprintf(w, "\nPhase Timings:\n")
		for phase, duration := range r.build.Performance.Phases {
			fmt.Fprintf(w, "  %s:\t%.2f seconds\n", phase, duration)
		}
	}
	fmt.Fprintf(w, "\n")

	// Analysis Results
	fmt.Fprintf(w, "Performance Analysis Results\n")
	fmt.Fprintf(w, "=========================\n")
	fmt.Fprintf(w, "Resource Efficiency:\t%.2f%%\n", r.analysis.ResourceEfficiency*100)

	if len(r.analysis.MemoryUsageProfile) > 0 {
		fmt.Fprintf(w, "\nMemory Usage Profile:\n")
		for metric, value := range r.analysis.MemoryUsageProfile {
			fmt.Fprintf(w, "  %s:\t%s\n", metric, formatBytes(value))
		}
	}

	if len(r.analysis.CompilationOverhead) > 0 {
		fmt.Fprintf(w, "\nCompilation Overhead:\n")
		for phase, duration := range r.analysis.CompilationOverhead {
			fmt.Fprintf(w, "  %s:\t%.2f seconds\n", phase, duration)
		}
	}

	if len(r.analysis.OptimizationMetrics) > 0 {
		fmt.Fprintf(w, "\nOptimization Metrics:\n")
		for metric, value := range r.analysis.OptimizationMetrics {
			fmt.Fprintf(w, "  %s:\t%d\n", metric, value)
		}
	}

	// Compiler Remarks
	if len(r.build.Remarks) > 0 {
		fmt.Fprintf(w, "\nCompiler Remarks:\n")
		remarksByType := make(map[string]int)
		for _, remark := range r.build.Remarks {
			remarksByType[remark.Type]++
		}
		for remarkType, count := range remarksByType {
			fmt.Fprintf(w, "  %s:\t%d remarks\n", remarkType, count)
		}

		// Print detailed remarks (limited to avoid excessive output)
		const maxDetailedRemarks = 10
		if len(r.build.Remarks) > 0 {
			fmt.Fprintf(w, "\nDetailed Remarks (up to %d):\n", maxDetailedRemarks)
			for i, remark := range r.build.Remarks {
				if i >= maxDetailedRemarks {
					fmt.Fprintf(w, "  ... and %d more remarks\n", len(r.build.Remarks)-maxDetailedRemarks)
					break
				}
				fmt.Fprintf(w, "  - %s [%s] in %s\n", remark.Message, remark.Type, remark.Function)
				if remark.Location.File != "" {
					fmt.Fprintf(w, "    at %s:%d:%d\n", remark.Location.File, remark.Location.Line, remark.Location.Column)
				}
			}
		}
	}
	fmt.Fprintf(w, "\n")

	// Bottlenecks and Recommendations
	if len(r.analysis.Bottlenecks) > 0 {
		fmt.Fprintf(w, "Performance Bottlenecks\n")
		fmt.Fprintf(w, "=====================\n")
		for _, b := range r.analysis.Bottlenecks {
			fmt.Fprintf(w, "- %s (Severity: %s)\n", b.Description, b.Severity)
			fmt.Fprintf(w, "  Impact: %.2f\n", b.Impact)
		}
		fmt.Fprintf(w, "\n")
	}

	if len(r.analysis.Recommendations) > 0 {
		fmt.Fprintf(w, "Recommendations\n")
		fmt.Fprintf(w, "===============\n")
		for _, rec := range r.analysis.Recommendations {
			fmt.Fprintf(w, "- %s\n", rec.Action)
			fmt.Fprintf(w, "  Category: %s\n", rec.Category)
			fmt.Fprintf(w, "  Impact: %s\n", rec.Impact)
			fmt.Fprintf(w, "  Details: %s\n", rec.Details)
			fmt.Fprintf(w, "\n")
		}
	}

	return w.Flush()
}

func (r *Reporter) getStatus() string {
	if r.build.Success {
		return "SUCCESS"
	}
	return "FAILED"
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
