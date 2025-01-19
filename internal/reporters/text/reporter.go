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
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(r.outDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Generate report
	reportPath := filepath.Join(r.outDir, fmt.Sprintf("build-%s.txt", r.build.ID))
	file, err := os.Create(reportPath)
	if err != nil {
		return fmt.Errorf("creating report file: %w", err)
	}
	defer file.Close()

	w := tabwriter.NewWriter(file, 0, 0, 2, ' ', 0)

	// Write build summary
	fmt.Fprintf(w, "Build Report\n")
	fmt.Fprintf(w, "============\n\n")
	fmt.Fprintf(w, "Build ID:\t%s\n", r.build.ID)
	fmt.Fprintf(w, "Status:\t%s\n", r.getStatus())
	fmt.Fprintf(w, "Start Time:\t%s\n", r.build.StartTime.Format(time.RFC3339))
	fmt.Fprintf(w, "Duration:\t%.2f seconds\n", r.build.Duration)
	fmt.Fprintf(w, "Source File:\t%s\n", r.build.Source.MainFile)
	fmt.Fprintf(w, "\n")

	// Write compiler information
	fmt.Fprintf(w, "Compiler Information\n")
	fmt.Fprintf(w, "-------------------\n")
	fmt.Fprintf(w, "Compiler:\t%s\n", r.build.Compiler.Name)
	fmt.Fprintf(w, "Version:\t%s\n", r.build.Compiler.Version)
	fmt.Fprintf(w, "Target:\t%s\n", r.build.Compiler.Target)
	fmt.Fprintf(w, "Options:\t%s\n", strings.Join(r.build.Compiler.Options, " "))
	fmt.Fprintf(w, "\n")

	// Write performance analysis
	fmt.Fprintf(w, "Performance Analysis\n")
	fmt.Fprintf(w, "--------------------\n")
	fmt.Fprintf(w, "Resource Efficiency:\t%.2f%%\n", r.analysis.ResourceEfficiency*100)
	fmt.Fprintf(w, "\n")

	// Write memory usage
	fmt.Fprintf(w, "Memory Usage\n")
	fmt.Fprintf(w, "------------\n")
	for metric, value := range r.analysis.MemoryUsageProfile {
		fmt.Fprintf(w, "%s:\t%s\n", metric, formatBytes(value))
	}
	fmt.Fprintf(w, "\n")

	// Write optimization metrics
	fmt.Fprintf(w, "Optimization Metrics\n")
	fmt.Fprintf(w, "-------------------\n")
	for metric, value := range r.analysis.OptimizationMetrics {
		fmt.Fprintf(w, "%s:\t%d\n", metric, value)
	}
	fmt.Fprintf(w, "\n")

	// Write bottlenecks
	if len(r.analysis.Bottlenecks) > 0 {
		fmt.Fprintf(w, "Performance Bottlenecks\n")
		fmt.Fprintf(w, "----------------------\n")
		for _, b := range r.analysis.Bottlenecks {
			fmt.Fprintf(w, "- %s (Severity: %s)\n", b.Description, b.Severity)
			fmt.Fprintf(w, "  Impact: %.2f\n", b.Impact)
		}
		fmt.Fprintf(w, "\n")
	}

	// Write recommendations
	if len(r.analysis.Recommendations) > 0 {
		fmt.Fprintf(w, "Recommendations\n")
		fmt.Fprintf(w, "---------------\n")
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
