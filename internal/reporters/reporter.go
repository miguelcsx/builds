package reporters

import (
	"builds/internal/analysis/performance"
	"builds/internal/models"
	"builds/internal/reporters/json"
	"builds/internal/reporters/text"
)

// Reporter defines the interface for build report generators
type Reporter interface {
	Generate() error
}

// Options holds configuration for report generation
type Options struct {
	OutputDir string
	Format    string
	Build     *models.Build
	Analysis  *performance.AnalysisResult
}

// NewReporter creates a new reporter based on the specified format
func NewReporter(opts Options) (Reporter, error) {
	switch opts.Format {
	case "json":
		return json.NewReporter(opts.Build, opts.Analysis, opts.OutputDir), nil
	case "txt":
		return text.NewReporter(opts.Build, opts.Analysis, opts.OutputDir), nil
	default:
		return text.NewReporter(opts.Build, opts.Analysis, opts.OutputDir), nil
	}
}
