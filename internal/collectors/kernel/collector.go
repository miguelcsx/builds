package kernel

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"builds/internal/models"
	kernelparser "builds/internal/parsers/kernel"
)

// Collector implements kernel info collection
type Collector struct {
	models.BaseCollector
	buildContext  *models.BuildContext
	kernelRemarks []models.KernelRemark
	stderr        io.Writer
}

// NewCollector creates a new kernel collector
func NewCollector(ctx *models.BuildContext, stderr io.Writer) *Collector {
	return &Collector{
		buildContext: ctx,
		stderr:       stderr,
	}
}

// Initialize prepares the kernel collector
func (c *Collector) Initialize(ctx context.Context) error {
	return nil
}

// Collect gathers kernel information
func (c *Collector) Collect(ctx context.Context) error {
	// Check if kernel info pass is enabled in the compiler flags
	if !c.hasKernelInfoPass() {
		return fmt.Errorf("kernel info pass not enabled")
	}

	// Run compilation with kernel info pass
	cmd := exec.CommandContext(ctx, c.buildContext.Compiler, c.buildContext.Args...)

	// Capture stderr where kernel info remarks are written
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// Parse kernel info remarks from stderr
	parser := kernelparser.NewParser(stderrPipe)
	remarks, err := parser.Parse()
	if err != nil {
		cmd.Wait() // Wait for command to finish before returning error
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	c.kernelRemarks = remarks
	return nil
}

// GetData returns the collected kernel information
func (c *Collector) GetData() interface{} {
	return c.kernelRemarks
}

// Cleanup performs any necessary cleanup
func (c *Collector) Cleanup(ctx context.Context) error {
	return nil
}

// hasKernelInfoPass checks if kernel info pass is enabled in compiler flags
func (c *Collector) hasKernelInfoPass() bool {
	for _, arg := range c.buildContext.Args {
		if strings.Contains(arg, "Rpass=kernel-info") {
			return true
		}
	}
	return false
}

// getKernelStatistics returns statistical information about kernels
func (c *Collector) getKernelStatistics() map[string]interface{} {
	stats := make(map[string]interface{})

	// Count different types of remarks
	remarkTypes := make(map[string]int)
	for _, remark := range c.kernelRemarks {
		remarkTypes[remark.Type]++
	}
	stats["remarkTypes"] = remarkTypes

	// Count memory access patterns
	memoryAccesses := make(map[string]int)
	for _, remark := range c.kernelRemarks {
		if remark.AccessType != "" {
			memoryAccesses[remark.AccessType]++
		}
	}
	stats["memoryAccesses"] = memoryAccesses

	// Count function calls
	functionCalls := make(map[string]int)
	for _, remark := range c.kernelRemarks {
		if remark.Callee != "" {
			functionCalls[remark.Callee]++
		}
	}
	stats["functionCalls"] = functionCalls

	return stats
}

// FilterRemarksByType returns remarks of a specific type
func (c *Collector) FilterRemarksByType(remarkType string) []models.KernelRemark {
	var filtered []models.KernelRemark
	for _, remark := range c.kernelRemarks {
		if remark.Type == remarkType {
			filtered = append(filtered, remark)
		}
	}
	return filtered
}

// GetKernelNames returns the names of all kernels found
func (c *Collector) GetKernelNames() []string {
	kernelSet := make(map[string]struct{})
	for _, remark := range c.kernelRemarks {
		if remark.Location.Function != "" {
			kernelSet[remark.Location.Function] = struct{}{}
		}
	}

	var kernels []string
	for kernel := range kernelSet {
		kernels = append(kernels, kernel)
	}
	return kernels
}

// GetKernelMetrics returns performance-related metrics for kernels
func (c *Collector) GetKernelMetrics() map[string]map[string]int {
	metrics := make(map[string]map[string]int)

	for _, remark := range c.kernelRemarks {
		if remark.Location.Function == "" {
			continue
		}

		if _, exists := metrics[remark.Location.Function]; !exists {
			metrics[remark.Location.Function] = make(map[string]int)
		}

		switch remark.Type {
		case "DirectCalls":
			if val, err := strconv.Atoi(remark.Value); err == nil {
				metrics[remark.Location.Function]["directCalls"] = val
			}
		case "Allocas":
			if val, err := strconv.Atoi(remark.Value); err == nil {
				metrics[remark.Location.Function]["allocas"] = val
			}
		case "FlatAddrspaceAccesses":
			if val, err := strconv.Atoi(remark.Value); err == nil {
				metrics[remark.Location.Function]["flatAddrspaceAccesses"] = val
			}
		}
	}

	return metrics
}
