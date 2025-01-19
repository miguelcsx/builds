package models

// Location represents a source code location
type Location struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Function string `json:"function"`
}

// KernelRemark represents a single kernel info remark
type KernelRemark struct {
	Location    Location `json:"location"`
	Type        string   `json:"type"`    // e.g., "direct call", "load instruction"
	Message     string   `json:"message"` // The full message
	Callee      string   `json:"callee,omitempty"`
	Instruction string   `json:"instruction,omitempty"`
	Value       string   `json:"value,omitempty"`      // For metrics like thread limits
	AccessType  string   `json:"accessType,omitempty"` // Memory access type
}

// LLVMRemark represents an LLVM optimization remark from YAML
type LLVMRemark struct {
	Pass     string   `json:"pass"`
	Name     string   `json:"name"`
	DebugLoc Location `json:"debugLoc"`
	Function string   `json:"function"`
	Args     []Args   `json:"args"`
	Type     string   `json:"type"` // Passed, Missed, Analysis
}

// Args represents the arguments in an LLVM remark
type Args struct {
	String   string    `json:"string,omitempty"`
	Callee   string    `json:"callee,omitempty"`
	DebugLoc *Location `json:"debugLoc,omitempty"`
	Line     string    `json:"line,omitempty"`
	Column   string    `json:"column,omitempty"`
	Function string    `json:"function,omitempty"`
	Reason   string    `json:"reason,omitempty"`
	Impact   string    `json:"impact,omitempty"`
}
