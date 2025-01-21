// internal/reporters/reporter.go
package reporters

import (
	"builds/internal/analysis/performance"
	"builds/internal/models"
	"builds/internal/reporters/json"
	"builds/internal/reporters/stdout"
	"builds/internal/reporters/text"
	"io"
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
	Writer    io.Writer
}

// NewReporter creates a new reporter based on the specified format
func NewReporter(opts Options) (Reporter, error) {
	switch opts.Format {
	case "json":
		return json.NewReporter(opts.Build, opts.Analysis, opts.OutputDir), nil
	case "text":
		return text.NewReporter(opts.Build, opts.Analysis, opts.OutputDir), nil
	case "display", "stdout":
		return stdout.NewReporter(opts.Build, opts.Analysis, opts.Writer), nil
	default:
		return stdout.NewReporter(opts.Build, opts.Analysis, opts.Writer), nil
	}
}
