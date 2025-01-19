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
		Compiler    string    `json:"compiler"`
		Source      string    `json:"source"`
		Succeeded   bool      `json:"succeeded"`
		Efficiency  float64   `json:"efficiency"`
		Bottlenecks []string  `json:"bottlenecks"`
	}{
		ID:          r.build.ID,
		Status:      r.getStatus(),
		StartTime:   r.build.StartTime,
		Duration:    r.build.Duration,
		Compiler:    r.build.Compiler.Name,
		Source:      r.build.Source.MainFile,
		Succeeded:   r.build.Success,
		Efficiency:  r.analysis.ResourceEfficiency,
		Bottlenecks: r.getBottleneckDescriptions(),
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
