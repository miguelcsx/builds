package models

import (
	"time"
)

// Build represents a complete build process and its information
type Build struct {
	ID        string    `json:"id"`        // Unique build identifier
	StartTime time.Time `json:"startTime"` // Build start time
	EndTime   time.Time `json:"endTime"`   // Build end time
	Duration  float64   `json:"duration"`  // Build duration in seconds
	Success   bool      `json:"success"`   // Build success status
	Error     string    `json:"error"`     // Error message if build failed

	// Build environment
	Environment EnvironmentInfo `json:"environment"` // Build environment information
	Hardware    HardwareInfo    `json:"hardware"`    // Hardware information
	Compiler    CompilerInfo    `json:"compiler"`    // Compiler information

	// Build data
	Source  SourceInfo  `json:"source"`  // Source code information
	Command CommandInfo `json:"command"` // Build command information
	Output  OutputInfo  `json:"output"`  // Build output information

	// Analysis data
	KernelInfo    []KernelRemark  `json:"kernelInfo"`    // Kernel information
	LLVMRemarks   []LLVMRemark    `json:"llvmRemarks"`   // LLVM optimization remarks
	ResourceUsage ResourceUsage   `json:"resourceUsage"` // Resource usage statistics
	Performance   PerformanceInfo `json:"performance"`   // Performance analysis
}

// EnvironmentInfo contains information about the build environment
type EnvironmentInfo struct {
	OS           string            `json:"os"`           // Operating system
	Architecture string            `json:"architecture"` // System architecture
	Environment  map[string]string `json:"environment"`  // Environment variables
	WorkingDir   string            `json:"workingDir"`   // Working directory
}

// HardwareInfo contains system hardware information
type HardwareInfo struct {
	CPU        CPUInfo    `json:"cpu"`        // CPU information
	Memory     MemoryInfo `json:"memory"`     // Memory information
	GPU        []GPUInfo  `json:"gpu"`        // GPU information
	NumCores   int        `json:"numCores"`   // Number of CPU cores
	NumThreads int        `json:"numThreads"` // Number of CPU threads
}

// CompilerInfo contains compiler-specific information
type CompilerInfo struct {
	Name          string          `json:"name"`          // Compiler name
	Version       string          `json:"version"`       // Compiler version
	Target        string          `json:"target"`        // Target triple
	Options       []string        `json:"options"`       // Compiler options
	Optimizations map[string]bool `json:"optimizations"` // Enabled optimizations
}

// SourceInfo contains information about the source code
type SourceInfo struct {
	MainFile     string   `json:"mainFile"`     // Main source file
	Dependencies []string `json:"dependencies"` // Source dependencies
	Hash         string   `json:"hash"`         // Source code hash
}

// CommandInfo contains information about the build command
type CommandInfo struct {
	Executable  string   `json:"executable"`  // Compiler executable
	Args        []string `json:"args"`        // Command arguments
	FullCommand string   `json:"fullCommand"` // Full command string
}

// OutputInfo contains build output information
type OutputInfo struct {
	Stdout    string     `json:"stdout"`    // Standard output
	Stderr    string     `json:"stderr"`    // Standard error
	Artifacts []Artifact `json:"artifacts"` // Build artifacts
	ExitCode  int        `json:"exitCode"`  // Command exit code
}

// Artifact represents a build artifact
type Artifact struct {
	Path string `json:"path"` // Artifact path
	Type string `json:"type"` // Artifact type
	Size int64  `json:"size"` // Artifact size
	Hash string `json:"hash"` // Artifact hash
}

// ResourceUsage contains resource usage statistics
type ResourceUsage struct {
	MaxMemory   int64   `json:"maxMemory"`   // Peak memory usage
	CPUTime     float64 `json:"cpuTime"`     // CPU time used
	ThreadCount int     `json:"threadCount"` // Number of threads used
	IOStats     IOStats `json:"ioStats"`     // I/O statistics
}

// IOStats contains I/O statistics
type IOStats struct {
	ReadBytes    int64 `json:"readBytes"`    // Bytes read
	WrittenBytes int64 `json:"writtenBytes"` // Bytes written
	ReadCount    int64 `json:"readCount"`    // Number of read operations
	WriteCount   int64 `json:"writeCount"`   // Number of write operations
}

// CPUInfo contains CPU information
type CPUInfo struct {
	Model     string  `json:"model"`     // CPU model
	Frequency float64 `json:"frequency"` // CPU frequency in MHz
	CacheSize int     `json:"cacheSize"` // CPU cache size
	Vendor    string  `json:"vendor"`    // CPU vendor
}

// MemoryInfo contains memory information
type MemoryInfo struct {
	Total     int64 `json:"total"`     // Total memory in bytes
	Available int64 `json:"available"` // Available memory in bytes
	SwapTotal int64 `json:"swapTotal"` // Total swap in bytes
	SwapFree  int64 `json:"swapFree"`  // Free swap in bytes
}

// GPUInfo contains GPU information
type GPUInfo struct {
	Model       string `json:"model"`       // GPU model
	Memory      int64  `json:"memory"`      // GPU memory in bytes
	Driver      string `json:"driver"`      // GPU driver version
	ComputeCaps string `json:"computeCaps"` // Compute capabilities
}

// PerformanceInfo contains performance analysis information
type PerformanceInfo struct {
	CompileTime   float64            `json:"compileTime"`   // Compilation time
	LinkTime      float64            `json:"linkTime"`      // Linking time
	OptimizeTime  float64            `json:"optimizeTime"`  // Optimization time
	CacheMissRate float64            `json:"cacheMissRate"` // Cache miss rate
	PhaseTimings  map[string]float64 `json:"phaseTimings"`  // Timing for each phase
}
