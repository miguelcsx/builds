// internal/collectors/kernel/collector.go

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
	buildContext *models.BuildContext
	remarks      []models.CompilerRemark
	stderr       io.Writer
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

	c.remarks = remarks
	return nil
}

// GetData returns the collected kernel information
func (c *Collector) GetData() interface{} {
	return c.remarks
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
	for _, remark := range c.remarks {
		remarkTypes[remark.Type]++
	}
	stats["remarkTypes"] = remarkTypes

	// Count memory access patterns
	memoryAccesses := make(map[string]int)
	for _, remark := range c.remarks {
		// Look for memory access pattern in args
		for _, arg := range remark.Args {
			if arg.String != "" && strings.Contains(remark.Message, "memory") {
				memoryAccesses[arg.String]++
			}
		}
	}
	stats["memoryAccesses"] = memoryAccesses

	// Count function calls
	functionCalls := make(map[string]int)
	for _, remark := range c.remarks {
		for _, arg := range remark.Args {
			if arg.Callee != "" {
				functionCalls[arg.Callee]++
			}
		}
	}
	stats["functionCalls"] = functionCalls

	return stats
}

// FilterRemarksByType returns remarks of a specific type
func (c *Collector) FilterRemarksByType(remarkType string) []models.CompilerRemark {
	var filtered []models.CompilerRemark
	for _, remark := range c.remarks {
		if remark.Type == remarkType {
			filtered = append(filtered, remark)
		}
	}
	return filtered
}

// GetKernelNames returns the names of all kernels found
func (c *Collector) GetKernelNames() []string {
	kernelSet := make(map[string]struct{})
	for _, remark := range c.remarks {
		if remark.Function != "" {
			kernelSet[remark.Function] = struct{}{}
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

	for _, remark := range c.remarks {
		if remark.Function == "" {
			continue
		}

		if _, exists := metrics[remark.Function]; !exists {
			metrics[remark.Function] = make(map[string]int)
		}

		switch remark.Type {
		case "function_call":
			// Look for direct call count in args
			for _, arg := range remark.Args {
				if arg.String != "" {
					if val, err := strconv.Atoi(arg.String); err == nil {
						metrics[remark.Function]["directCalls"] = val
					}
				}
			}
		case "memory":
			// Look for allocation information in args
			for _, arg := range remark.Args {
				if arg.String != "" && strings.Contains(remark.Message, "alloca") {
					if val, err := strconv.Atoi(arg.String); err == nil {
						metrics[remark.Function]["allocas"] = val
					}
				}
			}
		case "memory_access":
			// Look for flat memory access information
			for _, arg := range remark.Args {
				if arg.String != "" && strings.Contains(arg.String, "flat") {
					if val, err := strconv.Atoi(arg.String); err == nil {
						metrics[remark.Function]["flatAddrspaceAccesses"] = val
					}
				}
			}
		}
	}

	return metrics
}
