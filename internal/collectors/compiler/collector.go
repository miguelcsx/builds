package compiler

import (
	"context"
	"os/exec"
	"regexp"
	"strings"

	"builds/internal/models"
)

// Collector implements compiler information collection
type Collector struct {
	models.BaseCollector
	info         models.Compiler
	buildContext *models.BuildContext
}

// Common patterns for compiler version parsing
var (
	clangVersionPattern = regexp.MustCompile(`clang version (\d+\.\d+\.\d+)`)
	gccVersionPattern   = regexp.MustCompile(`gcc version (\d+\.\d+\.\d+)`)
	targetPattern       = regexp.MustCompile(`Target: (.+)`)
)

// NewCollector creates a new compiler collector
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

// Initialize prepares the compiler collector
func (c *Collector) Initialize(ctx context.Context) error {
	// Set compiler name based on executable
	c.info.Name = c.inferCompilerType(c.buildContext.Compiler)
	return nil
}

// Collect gathers compiler information
func (c *Collector) Collect(ctx context.Context) error {
	// Get compiler version
	version, err := c.collectVersion()
	if err != nil {
		return err
	}
	c.info.Version = version

	// Get target information
	target, err := c.collectTarget()
	if err != nil {
		return err
	}
	c.info.Target = target

	// Parse compiler options
	c.info.Options = c.parseCompilerOptions(c.buildContext.Args)

	// Collect optimization information
	opts, err := c.collectOptimizations()
	if err != nil {
		return err
	}
	c.info.Optimizations = opts

	// Set language information
	c.setLanguageInfo()

	// Collect compiler features
	c.collectFeatures()

	return nil
}

// GetData returns the collected compiler information
func (c *Collector) GetData() interface{} {
	return c.info
}

// Cleanup performs any necessary cleanup
func (c *Collector) Cleanup(ctx context.Context) error {
	return nil
}

// setLanguageInfo sets the language information based on compiler type
func (c *Collector) setLanguageInfo() {
	switch c.info.Name {
	case "clang":
		c.info.Language = models.Language{
			Name:          "C/C++",
			Version:       "C++17", // Default, could be determined from flags
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

// collectFeatures collects compiler feature information
func (c *Collector) collectFeatures() {
	c.info.Features = models.CompilerFeatures{
		SupportsOpenMP: c.hasOpenMPSupport(),
		SupportsGPU:    c.hasGPUSupport(),
		SupportsLTO:    c.hasLTOSupport(),
		SupportsPGO:    c.hasPGOSupport(),
		Extensions:     c.getCompilerExtensions(),
	}
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

// collectVersion gets the compiler version
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

// collectTarget gets the compiler target information
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

// parseCompilerOptions analyzes compiler options
func (c *Collector) parseCompilerOptions(args []string) []string {
	var options []string
	for _, arg := range args {
		// Filter out input/output files and add relevant options
		if strings.HasPrefix(arg, "-") {
			options = append(options, arg)
		}
	}
	return options
}

// collectOptimizations determines enabled optimizations
func (c *Collector) collectOptimizations() (map[string]bool, error) {
	optimizations := make(map[string]bool)

	// Parse optimization flags from options
	for _, arg := range c.info.Options {
		switch arg {
		case "-O0":
			optimizations["optimization_level"] = false
		case "-O1", "-O2", "-O3":
			optimizations["optimization_level"] = true
			optimizations[arg[1:]] = true
		case "-Ofast":
			optimizations["fast_math"] = true
		case "-fPIC":
			optimizations["position_independent"] = true
		case "-flto":
			optimizations["link_time_optimization"] = true
		case "-march=native":
			optimizations["native_architecture"] = true
		}

		// Add flag to the flags map with its value
		if strings.HasPrefix(arg, "-f") {
			c.info.Flags[arg] = "enabled"
		}
	}

	return optimizations, nil
}

// hasOpenMPSupport checks if OpenMP is supported
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

// hasGPUSupport checks if GPU compilation is supported
func (c *Collector) hasGPUSupport() bool {
	switch c.info.Name {
	case "clang":
		return c.hasClangGPUSupport()
	case "gcc":
		return c.hasGCCGPUSupport()
	}
	return false
}

// hasLTOSupport checks if Link Time Optimization is supported
func (c *Collector) hasLTOSupport() bool {
	cmd := exec.Command(c.buildContext.Compiler, "-flto=thin", "--help")
	return cmd.Run() == nil
}

// hasPGOSupport checks if Profile Guided Optimization is supported
func (c *Collector) hasPGOSupport() bool {
	cmd := exec.Command(c.buildContext.Compiler, "-fprofile-generate", "--help")
	return cmd.Run() == nil
}

// getCompilerExtensions returns supported compiler extensions
func (c *Collector) getCompilerExtensions() []string {
	var extensions []string

	switch c.info.Name {
	case "clang":
		extensions = []string{"OpenMP", "OpenCL", "CUDA", "HIP"}
	case "gcc":
		extensions = []string{"OpenMP", "OpenACC", "NVPTX"}
	}

	return extensions
}

// hasClangGPUSupport checks Clang-specific GPU support
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

// hasGCCGPUSupport checks GCC-specific GPU support
func (c *Collector) hasGCCGPUSupport() bool {
	cmd := exec.Command(c.buildContext.Compiler, "--help")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), "nvptx") ||
		strings.Contains(string(output), "amdgcn")
}
