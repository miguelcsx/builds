// internal/collectors/compiler/collector.go
package compiler

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"builds/internal/models"
)

var (
	clangVersionPattern = regexp.MustCompile(`clang version (\d+\.\d+\.\d+)`)
	gccVersionPattern   = regexp.MustCompile(`gcc version (\d+\.\d+\.\d+)`)
	targetPattern       = regexp.MustCompile(`Target: (.+)`)
)

type Collector struct {
	models.BaseCollector
	info         models.Compiler
	buildContext *models.BuildContext
}

func NewCollector(ctx *models.BuildContext) *Collector {
	return &Collector{
		buildContext: ctx,
		info: models.Compiler{
			Language:      models.Language{},
			Features:      models.CompilerFeatures{},
			Options:       []string{},
			Optimizations: make(map[string]bool),
			Flags:         make(map[string]string),
		},
	}
}

func (c *Collector) Initialize(ctx context.Context) error {
	c.info.Name = c.inferCompilerType(c.buildContext.Compiler)
	return nil
}

func (c *Collector) Collect(ctx context.Context) error {
	// Get compiler version
	version, err := c.collectVersion()
	if err != nil {
		return fmt.Errorf("version collection failed: %w", err)
	}
	c.info.Version = version

	// Get target information
	target, err := c.collectTarget()
	if err != nil {
		return fmt.Errorf("target collection failed: %w", err)
	}
	c.info.Target = target

	// Parse current options (without modifying them)
	c.info.Options = c.parseCompilerOptions(c.buildContext.Args)

	// Set language information
	c.setLanguageInfo()

	// Collect compiler features
	c.collectFeatures()

	return nil
}

func (c *Collector) GetData() interface{} {
	return c.info
}

func (c *Collector) Cleanup(ctx context.Context) error {
	return nil
}

func (c *Collector) inferCompilerType(compiler string) string {
	base := strings.ToLower(compiler)
	switch {
	case strings.Contains(base, "clang"):
		return "clang"
	case strings.Contains(base, "gcc"):
		return "gcc"
	default:
		return "unknown"
	}
}

func (c *Collector) collectVersion() (string, error) {
	cmd := exec.Command(c.buildContext.Compiler, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	var version string
	switch c.info.Name {
	case "clang":
		if matches := clangVersionPattern.FindStringSubmatch(string(output)); len(matches) > 1 {
			version = matches[1]
		}
	case "gcc":
		if matches := gccVersionPattern.FindStringSubmatch(string(output)); len(matches) > 1 {
			version = matches[1]
		}
	}

	return version, nil
}

func (c *Collector) collectTarget() (string, error) {
	var args []string
	switch c.info.Name {
	case "clang":
		args = []string{"--version", "-v"}
	case "gcc":
		args = []string{"-v"}
	}

	cmd := exec.Command(c.buildContext.Compiler, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	if matches := targetPattern.FindStringSubmatch(string(output)); len(matches) > 1 {
		return matches[1], nil
	}

	return "", nil
}

func (c *Collector) parseCompilerOptions(args []string) []string {
	var options []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			options = append(options, arg)
		}
	}
	return options
}

func (c *Collector) setLanguageInfo() {
	switch c.info.Name {
	case "clang":
		c.info.Language = models.Language{
			Name:          "C/C++",
			Version:       "C++17",
			Specification: "ISO/IEC 14882:2017",
		}
	case "gcc":
		c.info.Language = models.Language{
			Name:          "C/C++",
			Version:       "C++17",
			Specification: "ISO/IEC 14882:2017",
		}
	}
}

func (c *Collector) collectFeatures() {
	c.info.Features = models.CompilerFeatures{
		SupportsOpenMP: c.hasOpenMPSupport(),
		SupportsGPU:    c.hasGPUSupport(),
		SupportsLTO:    c.hasLTOSupport(),
		SupportsPGO:    c.hasPGOSupport(),
		Extensions:     c.getCompilerExtensions(),
	}
}

func (c *Collector) hasOpenMPSupport() bool {
	var testProgram string
	switch c.info.Name {
	case "clang", "gcc":
		testProgram = "#include <omp.h>\nint main() { return 0; }"
	default:
		return false
	}

	cmd := exec.Command(c.buildContext.Compiler, "-fopenmp", "-x", "c", "-")
	cmd.Stdin = strings.NewReader(testProgram)
	return cmd.Run() == nil
}

func (c *Collector) hasGPUSupport() bool {
	switch c.info.Name {
	case "clang":
		return c.hasClangGPUSupport()
	case "gcc":
		return c.hasGCCGPUSupport()
	}
	return false
}

func (c *Collector) hasLTOSupport() bool {
	cmd := exec.Command(c.buildContext.Compiler, "-flto=thin", "--help")
	return cmd.Run() == nil
}

func (c *Collector) hasPGOSupport() bool {
	cmd := exec.Command(c.buildContext.Compiler, "-fprofile-generate", "--help")
	return cmd.Run() == nil
}

func (c *Collector) getCompilerExtensions() []string {
	switch c.info.Name {
	case "clang":
		return []string{"OpenMP", "OpenCL", "CUDA", "HIP"}
	case "gcc":
		return []string{"OpenMP", "OpenACC", "NVPTX"}
	}
	return nil
}

func (c *Collector) hasClangGPUSupport() bool {
	cmd := exec.Command(c.buildContext.Compiler, "--help")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "cuda") ||
		strings.Contains(string(output), "opencl") ||
		strings.Contains(string(output), "hip")
}

func (c *Collector) hasGCCGPUSupport() bool {
	cmd := exec.Command(c.buildContext.Compiler, "--help")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "nvptx") ||
		strings.Contains(string(output), "amdgcn")
}
