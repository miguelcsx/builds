// internal/collectors/remarks/collector.go

package remarks

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"builds/internal/models"
	"builds/internal/parsers/remarks"
)

type Collector struct {
	models.BaseCollector
	buildContext *models.BuildContext
	remarks      []models.CompilerRemark
	yamlPath     string
	mu           sync.Mutex
}

func NewCollector(ctx *models.BuildContext) *Collector {
	return &Collector{
		buildContext: ctx,
	}
}

func (c *Collector) Initialize(ctx context.Context) error {
	log.Printf("Initializing remarks collector for build %s", c.buildContext.BuildID)
	c.yamlPath = filepath.Join(os.TempDir(), fmt.Sprintf("remarks_%s.yml", c.buildContext.BuildID))
	c.addCompilerFlags()
	return nil
}

func (c *Collector) addCompilerFlags() {
	// Store original args for comparison
	originalArgs := append([]string{}, c.buildContext.Args...)

	// Add YAML output flags
	optimFlags := []string{
		"-fsave-optimization-record",
		fmt.Sprintf("-foptimization-record-file=%s", c.yamlPath),
		"-O2",
	}

	// Remove any existing optimization flags
	var cleanedArgs []string
	for _, arg := range c.buildContext.Args {
		if !c.isOptimizationFlag(arg) {
			cleanedArgs = append(cleanedArgs, arg)
		}
	}

	// Combine flags
	c.buildContext.Args = append(optimFlags, cleanedArgs...)

	log.Printf("Original args: %v", originalArgs)
	log.Printf("Modified args: %v", c.buildContext.Args)
}

func (c *Collector) isOptimizationFlag(arg string) bool {
	return strings.HasPrefix(arg, "-fsave-optimization-record") ||
		strings.HasPrefix(arg, "-foptimization-record-file") ||
		strings.HasPrefix(arg, "-O") ||
		strings.HasPrefix(arg, "-Rpass")
}

func (c *Collector) Collect(ctx context.Context) error {
	// Ensure YAML file cleanup
	defer func() {
		if err := c.Cleanup(ctx); err != nil {
			log.Printf("Warning: failed to cleanup YAML file: %v", err)
		}
	}()

	// Run compiler to generate YAML file
	cmd := exec.CommandContext(ctx, c.buildContext.Compiler, c.buildContext.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("Compilation completed with status: %v", err)
	}

	// Check if YAML file exists
	if _, err := os.Stat(c.yamlPath); err != nil {
		return fmt.Errorf("optimization record file not created: %w", err)
	}

	// Parse the YAML file
	parser := remarks.NewParser(c.yamlPath)
	parsedRemarks, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("failed to parse remarks: %w", err)
	}

	// Update remarks with build ID
	for i := range parsedRemarks {
		parsedRemarks[i].ID = c.buildContext.BuildID
	}

	c.mu.Lock()
	c.remarks = parsedRemarks
	c.mu.Unlock()

	log.Printf("Collected %d remarks", len(parsedRemarks))
	return nil
}

func (c *Collector) GetData() interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.remarks
}

func (c *Collector) Cleanup(ctx context.Context) error {
	if c.yamlPath != "" {
		if err := os.Remove(c.yamlPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to cleanup remarks file: %w", err)
		}
	}
	return nil
}
