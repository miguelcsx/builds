// internal/models/build.go

package models

import "time"

// Build represents a complete build process and its information
type Build struct {
	ID        string    `json:"id"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	Duration  float64   `json:"duration"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`

	// Build environment and configuration
	Environment Environment `json:"environment"`
	Hardware    Hardware    `json:"hardware"`
	Compiler    Compiler    `json:"compiler"`

	// Build execution and output
	Command Command      `json:"command"`
	Output  Output       `json:"output"`
	Metrics BuildMetrics `json:"metrics"`

	// Analysis data
	Remarks       []CompilerRemark `json:"remarks"` // Generic compiler remarks
	ResourceUsage ResourceUsage    `json:"resourceUsage"`
	Performance   Performance      `json:"performance"`
}

// Environment represents the build environment
type Environment struct {
	OS         string            `json:"os"`
	Arch       string            `json:"arch"`
	Variables  map[string]string `json:"variables"`
	WorkingDir string            `json:"workingDir"`
}

// Hardware represents system hardware information
type Hardware struct {
	CPU    CPU    `json:"cpu"`
	Memory Memory `json:"memory"`
	GPUs   []GPU  `json:"gpus,omitempty"`
}

type CPU struct {
	Model     string  `json:"model"`
	Frequency float64 `json:"frequency"`
	Cores     int32   `json:"cores"`
	Threads   int32   `json:"threads"`
	Vendor    string  `json:"vendor"`
	CacheSize int64   `json:"cacheSize"`
}

type Memory struct {
	Total     int64 `json:"total"`
	Available int64 `json:"available"`
	SwapTotal int64 `json:"swapTotal"`
	SwapFree  int64 `json:"swapFree"`
	Used      int64 `json:"used"`
}

type GPU struct {
	Model       string `json:"model"`
	Memory      int64  `json:"memory"`
	Driver      string `json:"driver"`
	ComputeCaps string `json:"computeCaps"`
}

// Compiler represents the compiler configuration
type Compiler struct {
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	Target        string            `json:"target"`
	Options       []string          `json:"options"`
	Optimizations map[string]bool   `json:"optimizations"`
	Flags         map[string]string `json:"flags"`
	Language      Language          `json:"language"`
	Extensions    []string          `json:"extensions"`
	Features      CompilerFeatures  `json:"features"`
}

type Language struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	Specification string `json:"specification"`
}

type CompilerFeatures struct {
	SupportsOpenMP bool     `json:"supportsOpenMP"`
	SupportsGPU    bool     `json:"supportsGPU"`
	SupportsLTO    bool     `json:"supportsLTO"`
	SupportsPGO    bool     `json:"supportsPGO"`
	Extensions     []string `json:"extensions"`
}

// Command represents the build command execution
type Command struct {
	Executable string            `json:"executable"`
	Arguments  []string          `json:"arguments"`
	WorkingDir string            `json:"workingDir"`
	Env        map[string]string `json:"env"`
}

// Output represents build output information
type Output struct {
	Stdout    string     `json:"stdout"`
	Stderr    string     `json:"stderr"`
	Artifacts []Artifact `json:"artifacts"`
	ExitCode  int32      `json:"exitCode"`
	Warnings  []string   `json:"warnings"`
	Errors    []string   `json:"errors"`
}

type Artifact struct {
	Path string `json:"path"`
	Type string `json:"type"`
	Size int64  `json:"size"`
	Hash string `json:"hash"`
}

// Represents the type of compiler remark
type RemarkType string

const (
	RemarkTypeOptimization RemarkType = "optimization"
	RemarkTypeKernel       RemarkType = "kernel"
	RemarkTypeAnalysis     RemarkType = "analysis"
	RemarkTypeMetric       RemarkType = "metric"
	RemarkTypeInfo         RemarkType = "info"
)

// Represents the type of compiler pass
type PassType string

const (
	PassTypeVectorization PassType = "vectorization"
	PassTypeInlining      PassType = "inlining"
	PassTypeAnalysis      PassType = "analysis"
	PassTypeKernelInfo    PassType = "kernel-info"
	PassTypeSizeInfo      PassType = "size-info"
)

// Represents the status of a remark
type RemarkStatus string

const (
	RemarkStatusPassed   RemarkStatus = "passed"
	RemarkStatusMissed   RemarkStatus = "missed"
	RemarkStatusAnalysis RemarkStatus = "analysis"
)

// CompilerRemark represents a generic compiler remark/diagnostic
type CompilerRemark struct {
	ID        string       `json:"id"`
	Type      RemarkType   `json:"type"`
	Pass      PassType     `json:"pass"`
	Status    RemarkStatus `json:"status"`
	Message   string       `json:"message"`
	Function  string       `json:"function"`
	Timestamp time.Time    `json:"timestamp"`

	// Source location
	Location Location `json:"location"`

	// Kernel-specific information
	KernelInfo *KernelInfo `json:"kernelInfo,omitempty"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type Location struct {
	File     string `json:"file"`
	Line     int32  `json:"line"`
	Column   int32  `json:"column"`
	Function string `json:"function"`
	Region   string `json:"region"`
	Artifact bool   `json:"artifact"`
}

// KernelInfo contains kernel-specific information
type KernelInfo struct {
	// Kernel characteristics
	ThreadLimit  int32  `json:"thread_limit,omitempty"`
	MaxThreadsX  int32  `json:"max_threads_x,omitempty"`
	MaxThreadsY  int32  `json:"max_threads_y,omitempty"`
	MaxThreadsZ  int32  `json:"max_threads_z,omitempty"`
	SharedMemory int64  `json:"shared_memory,omitempty"`
	Target       string `json:"target,omitempty"`

	// Memory access patterns
	MemoryAccesses []MemoryAccess `json:"memory_accesses,omitempty"`

	// Function calls
	DirectCalls   int32    `json:"direct_calls,omitempty"`
	IndirectCalls int32    `json:"indirect_calls,omitempty"`
	Callees       []string `json:"callees,omitempty"`

	// Resource usage
	AllocasCount        int32 `json:"allocas_count,omitempty"`
	AllocasStaticSize   int64 `json:"allocas_static_size,omitempty"`
	AllocasDynamicCount int32 `json:"allocas_dynamic_count,omitempty"`

	// Other metrics
	FlatAddressSpaceAccesses int32             `json:"flat_address_space_accesses,omitempty"`
	InlineAssemblyCalls      int32             `json:"inline_assembly_calls,omitempty"`
	Metrics                  map[string]int64  `json:"metrics,omitempty"`
	Attributes               map[string]string `json:"attributes,omitempty"`
}

// MemoryAccess represents a memory access pattern in kernel code
type MemoryAccess struct {
	Type          string `json:"type"`           // load, store, atomic, etc.
	AddressSpace  string `json:"address_space"`  // flat, shared, global, etc.
	Instruction   string `json:"instruction"`    // LLVM IR instruction
	Variable      string `json:"variable"`       // Variable being accessed
	AccessPattern string `json:"access_pattern"` // Sequential, strided, etc.
}

// ResourceUsage represents resource utilization during the build
type ResourceUsage struct {
	MaxMemory int64   `json:"maxMemory"`
	CPUTime   float64 `json:"cpuTime"`
	Threads   int32   `json:"threads"`
	IO        IOStats `json:"io"`
}

type IOStats struct {
	ReadBytes  int64 `json:"readBytes"`
	WriteBytes int64 `json:"writeBytes"`
	ReadCount  int64 `json:"readCount"`
	WriteCount int64 `json:"writeCount"`
}

// Performance represents performance metrics
type Performance struct {
	CompileTime  float64            `json:"compileTime"`
	LinkTime     float64            `json:"linkTime"`
	OptimizeTime float64            `json:"optimizeTime"`
	Phases       map[string]float64 `json:"phases"`
}

// BuildMetrics represents build statistics
type BuildMetrics struct {
	TotalFiles     int32              `json:"totalFiles"`
	ProcessedFiles int32              `json:"processedFiles"`
	Warnings       int32              `json:"warnings"`
	Errors         int32              `json:"errors"`
	InputSize      int64              `json:"inputSize"`
	OutputSize     int64              `json:"outputSize"`
	Metrics        map[string]float64 `json:"metrics"`
}
