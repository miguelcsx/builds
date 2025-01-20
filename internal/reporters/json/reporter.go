// internal/reporters/json/reporter.go
package json

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

	// Generate full report
	report := struct {
		Build     *models.Build               `json:"build"`
		Analysis  *performance.AnalysisResult `json:"analysis"`
		Generated time.Time                   `json:"generated"`
	}{
		Build:     r.build,
		Analysis:  r.analysis,
		Generated: time.Now(),
	}

	// Write full report
	fullReportPath := filepath.Join(r.outDir, fmt.Sprintf("build-%s-full.json", r.build.ID))
	if err := r.writeJSON(fullReportPath, report); err != nil {
		return fmt.Errorf("writing full report: %w", err)
	}

	// Write summary report
	summary := r.generateSummary()
	summaryPath := filepath.Join(r.outDir, fmt.Sprintf("build-%s-summary.json", r.build.ID))
	if err := r.writeJSON(summaryPath, summary); err != nil {
		return fmt.Errorf("writing summary: %w", err)
	}

	return nil
}

func (r *Reporter) writeJSON(path string, data interface{}) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func (r *Reporter) generateSummary() interface{} {
	return struct {
		ID          string    `json:"id"`
		Status      string    `json:"status"`
		StartTime   time.Time `json:"startTime"`
		Duration    float64   `json:"duration"`
		Environment struct {
			OS         string `json:"os"`
			Arch       string `json:"arch"`
			WorkingDir string `json:"workingDir"`
		} `json:"environment"`
		Compiler struct {
			Name     string   `json:"name"`
			Version  string   `json:"version"`
			Target   string   `json:"target"`
			Language string   `json:"language"`
			Features []string `json:"features"`
		} `json:"compiler"`
		Performance struct {
			CompileTime float64  `json:"compileTime"`
			LinkTime    float64  `json:"linkTime"`
			Efficiency  float64  `json:"efficiency"`
			Bottlenecks []string `json:"bottlenecks"`
			MaxMemory   int64    `json:"maxMemory"`
			CPUTime     float64  `json:"cpuTime"`
		} `json:"performance"`
		Success bool `json:"success"`
	}{
		ID:        r.build.ID,
		Status:    r.getStatus(),
		StartTime: r.build.StartTime,
		Duration:  r.build.Duration,
		Environment: struct {
			OS         string `json:"os"`
			Arch       string `json:"arch"`
			WorkingDir string `json:"workingDir"`
		}{
			OS:         r.build.Environment.OS,
			Arch:       r.build.Environment.Arch,
			WorkingDir: r.build.Environment.WorkingDir,
		},
		Compiler: struct {
			Name     string   `json:"name"`
			Version  string   `json:"version"`
			Target   string   `json:"target"`
			Language string   `json:"language"`
			Features []string `json:"features"`
		}{
			Name:     r.build.Compiler.Name,
			Version:  r.build.Compiler.Version,
			Target:   r.build.Compiler.Target,
			Language: r.build.Compiler.Language.Name,
			Features: r.build.Compiler.Features.Extensions,
		},
		Performance: struct {
			CompileTime float64  `json:"compileTime"`
			LinkTime    float64  `json:"linkTime"`
			Efficiency  float64  `json:"efficiency"`
			Bottlenecks []string `json:"bottlenecks"`
			MaxMemory   int64    `json:"maxMemory"`
			CPUTime     float64  `json:"cpuTime"`
		}{
			CompileTime: r.build.Performance.CompileTime,
			LinkTime:    r.build.Performance.LinkTime,
			Efficiency:  r.analysis.ResourceEfficiency,
			Bottlenecks: r.getBottleneckDescriptions(),
			MaxMemory:   r.build.ResourceUsage.MaxMemory,
			CPUTime:     r.build.ResourceUsage.CPUTime,
		},
		Success: r.build.Success,
	}
}

func (r *Reporter) getStatus() string {
	if r.build.Success {
		return "success"
	}
	return "failed"
}

func (r *Reporter) getBottleneckDescriptions() []string {
	var descriptions []string
	for _, b := range r.analysis.Bottlenecks {
		descriptions = append(descriptions, b.Description)
	}
	return descriptions
}
