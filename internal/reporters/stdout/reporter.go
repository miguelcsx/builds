// internal/reporters/stdout/reporter.go

package stdout

import (
	"io"
	"os"
	"text/tabwriter"

	"builds/internal/analysis/performance"
	"builds/internal/models"
	"builds/internal/reporters/text"
)

type Reporter struct {
	build    *models.Build
	analysis *performance.AnalysisResult
	writer io.Writer
}

func NewReporter(build *models.Build, analysis *performance.AnalysisResult, writer io.Writer) *Reporter {
	if writer == nil {
		writer = os.Stdout
	}
	return &Reporter{
		build:    build,
		analysis: analysis,
		writer: writer,
	}
}

func (r *Reporter) Generate() error {
	w := tabwriter.NewWriter(r.writer, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Reuse the text reporter
	reporter := text.NewReporter(r.build, r.analysis, "")
	return reporter.GenerateToWriter(w)
}
