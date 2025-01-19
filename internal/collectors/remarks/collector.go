package remarks

import (
	"context"
	"fmt"
	"io"
	"os/exec"

	"builds/internal/models"
	"builds/internal/parsers/remarks"
)

// Collector implements compiler remarks collection
type Collector struct {
	models.BaseCollector
	buildContext *models.BuildContext
	remarks      []models.CompilerRemark
	stderr       io.Writer
}

// NewCollector creates a new remarks collector
func NewCollector(ctx *models.BuildContext) *Collector {
	return &Collector{
		buildContext: ctx,
	}
}

// Initialize prepares the remarks collector
func (c *Collector) Initialize(ctx context.Context) error {
	return nil
}

// Collect gathers compiler remarks from stdout
func (c *Collector) Collect(ctx context.Context) error {
	// Create command with all optimization remarks enabled
	args := append([]string{
		"-O2",
		"-Rpass=.*",
		"-Rpass-missed=.*",
		"-Rpass-analysis=.*",
	}, c.buildContext.Args...)

	cmd := exec.CommandContext(ctx, c.buildContext.Compiler, args...)

	// Capture stdout and stderr
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("getting stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting compilation: %w", err)
	}

	// Parse remarks from stderr
	parser := remarks.NewParser(stderr)
	remarks, err := parser.Parse()
	if err != nil {
		cmd.Wait() // Wait for command to finish before returning error
		return fmt.Errorf("parsing remarks: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("compilation failed: %w", err)
	}

	c.remarks = remarks
	return nil
}

// GetData returns the collected remarks
func (c *Collector) GetData() interface{} {
	return c.remarks
}

// Cleanup performs any necessary cleanup
func (c *Collector) Cleanup(ctx context.Context) error {
	return nil
}

// FilterRemarksByPass filters remarks by pass name
func (c *Collector) FilterRemarksByPass(pass string) []models.CompilerRemark {
	var filtered []models.CompilerRemark
	for _, remark := range c.remarks {
		if remark.Pass == pass {
			filtered = append(filtered, remark)
		}
	}
	return filtered
}

// FilterRemarksByType filters remarks by type
func (c *Collector) FilterRemarksByType(remarkType string) []models.CompilerRemark {
	var filtered []models.CompilerRemark
	for _, remark := range c.remarks {
		if remark.Type == remarkType {
			filtered = append(filtered, remark)
		}
	}
	return filtered
}

// GetOptimizationSummary returns a summary of optimization remarks
func (c *Collector) GetOptimizationSummary() map[string]int {
	summary := make(map[string]int)
	for _, remark := range c.remarks {
		switch remark.Type {
		case "Passed":
			summary["passed"]++
		case "Missed":
			summary["missed"]++
		case "Analysis":
			summary["analysis"]++
		}
	}
	return summary
}

// GetOptimizationsByFunction returns optimization remarks grouped by function
func (c *Collector) GetOptimizationsByFunction() map[string][]models.CompilerRemark {
	byFunction := make(map[string][]models.CompilerRemark)
	for _, remark := range c.remarks {
		if remark.Function != "" {
			byFunction[remark.Function] = append(byFunction[remark.Function], remark)
		}
	}
	return byFunction
}

// GetRemarksWithReason returns remarks that have a specific reason
func (c *Collector) GetRemarksWithReason(reason string) []models.CompilerRemark {
	var filtered []models.CompilerRemark
	for _, remark := range c.remarks {
		for _, arg := range remark.Args {
			if arg.Reason == reason {
				filtered = append(filtered, remark)
				break
			}
		}
	}
	return filtered
}
