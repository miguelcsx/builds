// internal/collectors/kernel/collector.go

package kernel

import (
	"context"
	"fmt"
	"io"
	"os/exec"
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
	if !c.hasKernelInfoPass() {
		return fmt.Errorf("kernel info pass not enabled")
	}

	cmd := exec.CommandContext(ctx, c.buildContext.Compiler, c.buildContext.Args...)

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	parser := kernelparser.NewParser(stderrPipe)
	remarks, err := parser.Parse()
	if err != nil {
		cmd.Wait()
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
		remarkTypes[string(remark.Type)]++
	}
	stats["remarkTypes"] = remarkTypes

	// Count memory access patterns
	memoryAccesses := make(map[string]int)
	for _, remark := range c.remarks {
		if strings.Contains(remark.Message, "memory") {
			if meta, ok := remark.Metadata["memory_access"]; ok {
				if access, ok := meta.(string); ok {
					memoryAccesses[access]++
				}
			}
		}
	}
	stats["memoryAccesses"] = memoryAccesses

	// Count function calls
	functionCalls := make(map[string]int)
	for _, remark := range c.remarks {
		if meta, ok := remark.Metadata["callee"]; ok {
			if callee, ok := meta.(string); ok {
				functionCalls[callee]++
			}
		}
	}
	stats["functionCalls"] = functionCalls

	return stats
}

// FilterRemarksByType returns remarks filtered by type
func (c *Collector) FilterRemarksByType(remarkType models.RemarkType) []models.CompilerRemark {
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
		case models.RemarkTypeKernel:
			if info := remark.KernelInfo; info != nil {
				metrics[remark.Function]["directCalls"] = int(info.DirectCalls)
				metrics[remark.Function]["flatAddressSpaceAccesses"] = int(info.FlatAddressSpaceAccesses)
				metrics[remark.Function]["allocasCount"] = int(info.AllocasCount)
			}
		}

		// Add any other metrics from metadata
		if meta, ok := remark.Metadata["metrics"].(map[string]int); ok {
			for k, v := range meta {
				metrics[remark.Function][k] = v
			}
		}
	}

	return metrics
}
