// internal/reporters/text/reporter.go
package text

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

type remarkStats struct {
	TotalRemarks  int
	ByType        map[string]int
	ByPass        map[string]int
	ByFunction    map[string]int
	Optimizations struct {
		Passed int
		Missed int
		Total  int
	}
	InliningStats struct {
		Successful int
		Failed     int
		Total      int
	}
	KernelStats struct {
		TotalAccesses    int
		TotalThreadLimit int
		TotalDirectCalls int
		TotalAllocas     int
	}
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
	return r.GenerateToWriter(w)
}

func (r *Reporter) GenerateToWriter(w *tabwriter.Writer) error {
	// Generate each section
	sections := []func(*tabwriter.Writer) error{
		r.generateBuildSummary,
		r.generateEnvironmentInfo,
		r.generateHardwareInfo,
		r.generateCompilerInfo,
		r.generateCommandInfo,
		r.generateOutputInfo,
		r.generateResourceUsage,
		r.generatePerformanceInfo,
		r.generateAnalysisResults,
		r.generateOptimizationRemarks,
		r.generateBottlenecks,
	}

	for _, section := range sections {
		if err := section(w); err != nil {
			return err
		}
		fmt.Fprintf(w, "\n")
	}

	return w.Flush()
}

func (r *Reporter) generateOptimizationRemarks(w *tabwriter.Writer) error {
	if len(r.build.Remarks) == 0 {
		return nil
	}

	fmt.Fprintf(w, "Compiler Optimization Remarks\n")
	fmt.Fprintf(w, "===========================\n\n")

	// Calculate statistics
	stats := r.calculateRemarkStats()

	// Print Summary Statistics
	fmt.Fprintf(w, "Summary Statistics\n")
	fmt.Fprintf(w, "-----------------\n")
	fmt.Fprintf(w, "Total Remarks:\t%d\n", stats.TotalRemarks)

	if stats.Optimizations.Total > 0 {
		successRate := float64(stats.Optimizations.Passed) / float64(stats.Optimizations.Total) * 100
		fmt.Fprintf(w, "Optimization Success Rate:\t%.1f%% (%d/%d)\n",
			successRate, stats.Optimizations.Passed, stats.Optimizations.Total)
	}

	if stats.InliningStats.Total > 0 {
		inlineRate := float64(stats.InliningStats.Successful) / float64(stats.InliningStats.Total) * 100
		fmt.Fprintf(w, "Inlining Success Rate:\t%.1f%% (%d/%d)\n",
			inlineRate, stats.InliningStats.Successful, stats.InliningStats.Total)
	}

	// Print Distribution by Type
	fmt.Fprintf(w, "\nDistribution by Type\n")
	fmt.Fprintf(w, "-------------------\n")
	r.printSortedMap(w, stats.ByType, stats.TotalRemarks)

	// Print Distribution by Pass
	fmt.Fprintf(w, "\nDistribution by Pass\n")
	fmt.Fprintf(w, "------------------\n")
	r.printSortedMap(w, stats.ByPass, stats.TotalRemarks)

	// Print Top Functions
	fmt.Fprintf(w, "\nTop Functions by Remark Count\n")
	fmt.Fprintf(w, "--------------------------\n")
	r.printTopItems(w, stats.ByFunction, 10)

	// Print Detailed Remarks
	fmt.Fprintf(w, "\nDetailed Remarks\n")
	fmt.Fprintf(w, "----------------\n")

	// Group remarks by pass
	remarksByPass := r.groupRemarksByPass()

	// Sort passes alphabetically
	var passes []string
	for pass := range remarksByPass {
		passes = append(passes, pass)
	}
	sort.Strings(passes)

	for _, pass := range passes {
		remarks := remarksByPass[pass]
		fmt.Fprintf(w, "\nPass: %s (%d remarks)\n", pass, len(remarks))
		fmt.Fprintf(w, "%s\n\n", strings.Repeat("-", len(pass)+20))

		for _, remark := range remarks {
			r.printRemark(w, remark)
		}
	}

	return nil
}

func (r *Reporter) generateBuildSummary(w *tabwriter.Writer) error {
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
	return nil
}

func (r *Reporter) generateEnvironmentInfo(w *tabwriter.Writer) error {
	fmt.Fprintf(w, "Environment Information\n")
	fmt.Fprintf(w, "=====================\n")
	fmt.Fprintf(w, "Operating System:\t%s\n", r.build.Environment.OS)
	fmt.Fprintf(w, "Architecture:\t%s\n", r.build.Environment.Arch)
	fmt.Fprintf(w, "Working Directory:\t%s\n", r.build.Environment.WorkingDir)
	if len(r.build.Environment.Variables) > 0 {
		fmt.Fprintf(w, "\nEnvironment Variables:\n")
		vars := make([]string, 0, len(r.build.Environment.Variables))
		for k := range r.build.Environment.Variables {
			vars = append(vars, k)
		}
		sort.Strings(vars)
		for _, k := range vars {
			fmt.Fprintf(w, "  %s:\t%s\n", k, r.build.Environment.Variables[k])
		}
	}
	return nil
}

func (r *Reporter) generateHardwareInfo(w *tabwriter.Writer) error {
	fmt.Fprintf(w, "Hardware Information\n")
	fmt.Fprintf(w, "===================\n")

	fmt.Fprintf(w, "\nCPU:\n")
	fmt.Fprintf(w, "  Model:\t%s\n", r.build.Hardware.CPU.Model)
	fmt.Fprintf(w, "  Vendor:\t%s\n", r.build.Hardware.CPU.Vendor)
	fmt.Fprintf(w, "  Frequency:\t%.2f MHz\n", r.build.Hardware.CPU.Frequency)
	fmt.Fprintf(w, "  Cores:\t%d\n", r.build.Hardware.CPU.Cores)
	fmt.Fprintf(w, "  Threads:\t%d\n", r.build.Hardware.CPU.Threads)
	fmt.Fprintf(w, "  Cache Size:\t%s\n", formatBytes(r.build.Hardware.CPU.CacheSize))

	fmt.Fprintf(w, "\nMemory:\n")
	fmt.Fprintf(w, "  Total:\t%s\n", formatBytes(r.build.Hardware.Memory.Total))
	fmt.Fprintf(w, "  Available:\t%s\n", formatBytes(r.build.Hardware.Memory.Available))
	fmt.Fprintf(w, "  Used:\t%s\n", formatBytes(r.build.Hardware.Memory.Used))
	fmt.Fprintf(w, "  Swap Total:\t%s\n", formatBytes(r.build.Hardware.Memory.SwapTotal))
	fmt.Fprintf(w, "  Swap Free:\t%s\n", formatBytes(r.build.Hardware.Memory.SwapFree))

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
	return nil
}

func (r *Reporter) generateCompilerInfo(w *tabwriter.Writer) error {
	fmt.Fprintf(w, "Compiler Information\n")
	fmt.Fprintf(w, "===================\n")
	fmt.Fprintf(w, "Name:\t%s\n", r.build.Compiler.Name)
	fmt.Fprintf(w, "Version:\t%s\n", r.build.Compiler.Version)
	fmt.Fprintf(w, "Target:\t%s\n", r.build.Compiler.Target)

	fmt.Fprintf(w, "\nLanguage:\n")
	fmt.Fprintf(w, "  Name:\t%s\n", r.build.Compiler.Language.Name)
	fmt.Fprintf(w, "  Version:\t%s\n", r.build.Compiler.Language.Version)
	fmt.Fprintf(w, "  Specification:\t%s\n", r.build.Compiler.Language.Specification)

	fmt.Fprintf(w, "\nFeatures:\n")
	fmt.Fprintf(w, "  OpenMP Support:\t%v\n", r.build.Compiler.Features.SupportsOpenMP)
	fmt.Fprintf(w, "  GPU Support:\t%v\n", r.build.Compiler.Features.SupportsGPU)
	fmt.Fprintf(w, "  LTO Support:\t%v\n", r.build.Compiler.Features.SupportsLTO)
	fmt.Fprintf(w, "  PGO Support:\t%v\n", r.build.Compiler.Features.SupportsPGO)
	if len(r.build.Compiler.Features.Extensions) > 0 {
		fmt.Fprintf(w, "  Extensions:\t%s\n", strings.Join(r.build.Compiler.Features.Extensions, ", "))
	}

	if len(r.build.Compiler.Options) > 0 {
		fmt.Fprintf(w, "\nOptions:\t%s\n", strings.Join(r.build.Compiler.Options, " "))
	}

	if len(r.build.Compiler.Optimizations) > 0 {
		fmt.Fprintf(w, "\nOptimizations:\n")
		for opt, enabled := range r.build.Compiler.Optimizations {
			fmt.Fprintf(w, "  %s:\t%v\n", opt, enabled)
		}
	}
	return nil
}

func (r *Reporter) generateCommandInfo(w *tabwriter.Writer) error {
	fmt.Fprintf(w, "Command Information\n")
	fmt.Fprintf(w, "==================\n")
	fmt.Fprintf(w, "Executable:\t%s\n", r.build.Command.Executable)
	fmt.Fprintf(w, "Working Directory:\t%s\n", r.build.Command.WorkingDir)

	if len(r.build.Command.Arguments) > 0 {
		fmt.Fprintf(w, "Arguments:\t%s\n", strings.Join(r.build.Command.Arguments, " "))
	}

	if len(r.build.Command.Env) > 0 {
		fmt.Fprintf(w, "\nEnvironment Overrides:\n")
		vars := make([]string, 0, len(r.build.Command.Env))
		for k := range r.build.Command.Env {
			vars = append(vars, k)
		}
		sort.Strings(vars)
		for _, k := range vars {
			fmt.Fprintf(w, "  %s:\t%s\n", k, r.build.Command.Env[k])
		}
	}
	return nil
}

func (r *Reporter) generateOutputInfo(w *tabwriter.Writer) error {
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
	return nil
}

func (r *Reporter) generateResourceUsage(w *tabwriter.Writer) error {
	fmt.Fprintf(w, "Resource Usage\n")
	fmt.Fprintf(w, "==============\n")
	fmt.Fprintf(w, "Max Memory:\t%s\n", formatBytes(r.build.ResourceUsage.MaxMemory))
	fmt.Fprintf(w, "CPU Time:\t%.2f seconds\n", r.build.ResourceUsage.CPUTime)
	fmt.Fprintf(w, "Threads:\t%d\n", r.build.ResourceUsage.Threads)

	fmt.Fprintf(w, "\nIO Statistics:\n")
	fmt.Fprintf(w, "  Read:\t%s (%d operations)\n",
		formatBytes(r.build.ResourceUsage.IO.ReadBytes),
		r.build.ResourceUsage.IO.ReadCount)
	fmt.Fprintf(w, "  Write:\t%s (%d operations)\n",
		formatBytes(r.build.ResourceUsage.IO.WriteBytes),
		r.build.ResourceUsage.IO.WriteCount)
	return nil
}

func (r *Reporter) generatePerformanceInfo(w *tabwriter.Writer) error {
	fmt.Fprintf(w, "Performance Information\n")
	fmt.Fprintf(w, "=====================\n")
	fmt.Fprintf(w, "Compile Time:\t%.2f seconds\n", r.build.Performance.CompileTime)
	fmt.Fprintf(w, "Link Time:\t%.2f seconds\n", r.build.Performance.LinkTime)
	fmt.Fprintf(w, "Optimize Time:\t%.2f seconds\n", r.build.Performance.OptimizeTime)

	if len(r.build.Performance.Phases) > 0 {
		fmt.Fprintf(w, "\nPhase Timings:\n")
		phases := make([]string, 0, len(r.build.Performance.Phases))
		for phase := range r.build.Performance.Phases {
			phases = append(phases, phase)
		}
		sort.Strings(phases)

		for _, phase := range phases {
			fmt.Fprintf(w, "  %s:\t%.2f seconds\n", phase, r.build.Performance.Phases[phase])
		}
	}
	return nil
}

func (r *Reporter) generateAnalysisResults(w *tabwriter.Writer) error {
	fmt.Fprintf(w, "Performance Analysis Results\n")
	fmt.Fprintf(w, "=========================\n")
	fmt.Fprintf(w, "Resource Efficiency:\t%.2f%%\n", r.analysis.ResourceEfficiency*100)

	if len(r.analysis.MemoryUsageProfile) > 0 {
		fmt.Fprintf(w, "\nMemory Usage Profile:\n")
		metrics := make([]string, 0, len(r.analysis.MemoryUsageProfile))
		for metric := range r.analysis.MemoryUsageProfile {
			metrics = append(metrics, metric)
		}
		sort.Strings(metrics)
		for _, metric := range metrics {
			fmt.Fprintf(w, "  %s:\t%s\n", metric, formatBytes(r.analysis.MemoryUsageProfile[metric]))
		}
	}

	if len(r.analysis.CompilationOverhead) > 0 {
		fmt.Fprintf(w, "\nCompilation Overhead:\n")
		phases := make([]string, 0, len(r.analysis.CompilationOverhead))
		for phase := range r.analysis.CompilationOverhead {
			phases = append(phases, phase)
		}
		sort.Strings(phases)
		for _, phase := range phases {
			fmt.Fprintf(w, "  %s:\t%.2f seconds\n", phase, r.analysis.CompilationOverhead[phase])
		}
	}

	if len(r.analysis.OptimizationMetrics) > 0 {
		fmt.Fprintf(w, "\nOptimization Metrics:\n")
		metrics := make([]string, 0, len(r.analysis.OptimizationMetrics))
		for metric := range r.analysis.OptimizationMetrics {
			metrics = append(metrics, metric)
		}
		sort.Strings(metrics)
		for _, metric := range metrics {
			fmt.Fprintf(w, "  %s:\t%d\n", metric, r.analysis.OptimizationMetrics[metric])
		}
	}
	return nil
}

func (r *Reporter) generateBottlenecks(w *tabwriter.Writer) error {
	if len(r.analysis.Bottlenecks) > 0 {
		fmt.Fprintf(w, "Performance Bottlenecks\n")
		fmt.Fprintf(w, "=====================\n")
		for _, b := range r.analysis.Bottlenecks {
			fmt.Fprintf(w, "- %s (Severity: %s)\n", b.Description, b.Severity)
			fmt.Fprintf(w, "  Impact: %.2f\n", b.Impact)
		}
	}

	if len(r.analysis.Recommendations) > 0 {
		fmt.Fprintf(w, "\nRecommendations\n")
		fmt.Fprintf(w, "===============\n")
		for _, rec := range r.analysis.Recommendations {
			fmt.Fprintf(w, "- %s\n", rec.Action)
			fmt.Fprintf(w, "  Category: %s\n", rec.Category)
			fmt.Fprintf(w, "  Impact: %s\n", rec.Impact)
			fmt.Fprintf(w, "  Details: %s\n\n", rec.Details)
		}
	}
	return nil
}

func (r *Reporter) printTopItems(w *tabwriter.Writer, m map[string]int, limit int) {
	type kv struct {
		Key   string
		Value int
	}

	var items []kv
	for k, v := range m {
		items = append(items, kv{k, v})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Value > items[j].Value
	})

	count := 0
	for _, item := range items {
		if count >= limit {
			break
		}
		fmt.Fprintf(w, "  %s:\t%d remarks\n", item.Key, item.Value)
		count++
	}
}

func (r *Reporter) groupRemarksByPass() map[string][]models.CompilerRemark {
	result := make(map[string][]models.CompilerRemark)
	for _, remark := range r.build.Remarks {
		result[remark.Pass] = append(result[remark.Pass], remark)
	}
	return result
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

func (r *Reporter) getStatus() string {
	if r.build.Success {
		return "SUCCESS"
	}
	return "FAILED"
}

func (r *Reporter) printRemark(w *tabwriter.Writer, remark models.CompilerRemark) {
	// Print base information
	fmt.Fprintf(w, "[%s] %s\n", remark.Type, remark.Message)

	if remark.Function != "" {
		fmt.Fprintf(w, "  Function:\t%s\n", remark.Function)
	}

	if remark.Location.File != "" {
		fmt.Fprintf(w, "  Location:\t%s:%d:%d\n",
			remark.Location.File,
			remark.Location.Line,
			remark.Location.Column)
	}

	// Print Args if available
	if len(remark.Args.Strings) > 0 {
		fmt.Fprintf(w, "  Arguments:\n")
		for _, arg := range remark.Args.Strings {
			fmt.Fprintf(w, "    - %s\n", arg)
		}
	}

	// Print specific argument details
	if remark.Args.Callee != "" {
		fmt.Fprintf(w, "  Callee:\t%s\n", remark.Args.Callee)
	}
	if remark.Args.Caller != "" {
		fmt.Fprintf(w, "  Caller:\t%s\n", remark.Args.Caller)
	}
	if remark.Args.Reason != "" {
		fmt.Fprintf(w, "  Reason:\t%s\n", remark.Args.Reason)
	}
	if remark.Args.Cost != "" {
		fmt.Fprintf(w, "  Cost:\t%s\n", remark.Args.Cost)
	}

	// Print kernel info if available
	if remark.KernelInfo != nil {
		fmt.Fprintf(w, "  Kernel Info:\n")
		if remark.KernelInfo.ThreadLimit > 0 {
			fmt.Fprintf(w, "    Thread Limit:\t%d\n", remark.KernelInfo.ThreadLimit)
		}
		if remark.KernelInfo.DirectCalls > 0 {
			fmt.Fprintf(w, "    Direct Calls:\t%d\n", remark.KernelInfo.DirectCalls)
		}
		if len(remark.KernelInfo.Callees) > 0 {
			fmt.Fprintf(w, "    Callees:\t%s\n", strings.Join(remark.KernelInfo.Callees, ", "))
		}
		if remark.KernelInfo.AllocasCount > 0 {
			fmt.Fprintf(w, "    Stack Allocations:\t%d (size: %d bytes)\n",
				remark.KernelInfo.AllocasCount,
				remark.KernelInfo.AllocasStaticSize)
		}
		if len(remark.KernelInfo.MemoryAccesses) > 0 {
			fmt.Fprintf(w, "    Memory Accesses:\n")
			for _, access := range remark.KernelInfo.MemoryAccesses {
				fmt.Fprintf(w, "      - %s %s access", access.Type, access.AddressSpace)
				if access.AccessPattern != "" {
					fmt.Fprintf(w, " (%s)", access.AccessPattern)
				}
				fmt.Fprintf(w, "\n")
				if access.Instruction != "" {
					fmt.Fprintf(w, "        Instruction: %s\n", access.Instruction)
				}
			}
		}
		if remark.KernelInfo.NumInstructions > 0 {
			fmt.Fprintf(w, "    Instructions:\t%d\n", remark.KernelInfo.NumInstructions)
		}
	}

	fmt.Fprintf(w, "\n")
}

func (r *Reporter) calculateRemarkStats() remarkStats {
	stats := remarkStats{
		ByType:     make(map[string]int),
		ByPass:     make(map[string]int),
		ByFunction: make(map[string]int),
	}

	for _, remark := range r.build.Remarks {
		stats.TotalRemarks++
		stats.ByType[remark.Type]++
		stats.ByPass[remark.Pass]++
		if remark.Function != "" {
			stats.ByFunction[remark.Function]++
		}

		// Track optimization statistics
		switch remark.Type {
		case "Passed":
			stats.Optimizations.Passed++
			stats.Optimizations.Total++
		case "Missed":
			stats.Optimizations.Missed++
			stats.Optimizations.Total++
		}

		// Track inlining statistics
		if remark.Pass == "inline" {
			stats.InliningStats.Total++
			if remark.Type == "Passed" {
				stats.InliningStats.Successful++
			} else {
				stats.InliningStats.Failed++
			}
		}

		// Track kernel statistics
		if remark.KernelInfo != nil {
			stats.KernelStats.TotalAccesses += len(remark.KernelInfo.MemoryAccesses)
			stats.KernelStats.TotalThreadLimit += int(remark.KernelInfo.ThreadLimit)
			stats.KernelStats.TotalDirectCalls += int(remark.KernelInfo.DirectCalls)
			stats.KernelStats.TotalAllocas += int(remark.KernelInfo.AllocasCount)
		}
	}

	return stats
}

func (r *Reporter) printSortedMap(w *tabwriter.Writer, m map[string]int, total int) {
	type kv struct {
		Key   string
		Value int
	}

	var items []kv
	for k, v := range m {
		items = append(items, kv{k, v})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Value > items[j].Value
	})

	for _, item := range items {
		percentage := float64(item.Value) / float64(total) * 100
		fmt.Fprintf(w, "  %s:\t%d\t(%.1f%%)\n", item.Key, item.Value, percentage)
	}
}
