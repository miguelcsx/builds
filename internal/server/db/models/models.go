// internal/server/db/models.go

package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type Build struct {
	ID            string `gorm:"primarykey"`
	StartTime     time.Time
	EndTime       time.Time
	Duration      float64
	Success       bool
	Error         string
	Environment   Environment      `gorm:"foreignKey:BuildID"`
	Hardware      Hardware         `gorm:"foreignKey:BuildID"`
	Compiler      Compiler         `gorm:"foreignKey:BuildID"`
	Command       Command          `gorm:"foreignKey:BuildID"`
	Output        Output           `gorm:"foreignKey:BuildID"`
	ResourceUsage ResourceUsage    `gorm:"foreignKey:BuildID"`
	Performance   Performance      `gorm:"foreignKey:BuildID"`
	Remarks       []CompilerRemark `gorm:"foreignKey:BuildID"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Environment struct {
	BuildID    string `gorm:"primarykey"`
	OS         string
	Arch       string
	WorkingDir string
	Variables  []EnvironmentVariable `gorm:"foreignKey:BuildID"`
}

type EnvironmentVariable struct {
	BuildID string `gorm:"primarykey"`
	Key     string `gorm:"primarykey"`
	Value   string
}

type Hardware struct {
	BuildID    string `gorm:"primarykey"`
	CPUModel   string
	CPUFreq    float64
	CPUCores   int32
	CPUThreads int32
	CPUVendor  string
	CacheSize  int64
	MemTotal   int64
	MemAvail   int64
	MemUsed    int64
	SwapTotal  int64
	SwapFree   int64
	GPUs       []GPU `gorm:"foreignKey:BuildID"`
}

type GPU struct {
	ID          uint `gorm:"primarykey"`
	BuildID     string
	Model       string
	Memory      int64
	Driver      string
	ComputeCaps string
}

type Compiler struct {
	BuildID         string `gorm:"primarykey"`
	Name            string
	Version         string
	Target          string
	LanguageName    string
	LanguageVersion string
	LanguageSpec    string
	Options         []CompilerOption       `gorm:"foreignKey:BuildID"`
	Optimizations   []CompilerOptimization `gorm:"foreignKey:BuildID"`
	Extensions      []CompilerExtension    `gorm:"foreignKey:BuildID"`
	SupportsOpenMP  bool
	SupportsGPU     bool
	SupportsLTO     bool
	SupportsPGO     bool
}

type CompilerOption struct {
	BuildID string `gorm:"primarykey"`
	Option  string `gorm:"primarykey"`
}

type CompilerOptimization struct {
	BuildID string `gorm:"primarykey"`
	Name    string `gorm:"primarykey"`
	Enabled bool
}

type CompilerExtension struct {
	BuildID   string `gorm:"primarykey"`
	Extension string `gorm:"primarykey"`
}

type Command struct {
	BuildID    string `gorm:"primarykey"`
	Executable string
	WorkingDir string
	Arguments  []CommandArgument `gorm:"foreignKey:BuildID"`
}

type CommandArgument struct {
	BuildID  string `gorm:"primarykey"`
	Position int    `gorm:"primarykey"`
	Argument string
}

type Output struct {
	BuildID   string `gorm:"primarykey"`
	Stdout    string
	Stderr    string
	ExitCode  int32
	Artifacts []Artifact `gorm:"foreignKey:BuildID"`
}

type Artifact struct {
	ID      uint `gorm:"primarykey"`
	BuildID string
	Path    string
	Type    string
	Size    int64
	Hash    string
}

type CompilerRemark struct {
	ID         uint   `gorm:"primarykey"`
	BuildID    string `gorm:"index"`
	Type       string // The YAML tag type (Passed, Missed, Analysis, etc)
	Pass       string `gorm:"type:text"`
	Name       string `gorm:"type:text"`
	Message    string `gorm:"type:text"`
	Function   string `gorm:"type:text"`
	Timestamp  time.Time
	Location   Location    `gorm:"embedded;embeddedPrefix:location_"`
	KernelInfo *KernelInfo `gorm:"foreignKey:RemarkID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Args       RemarkArgs  `gorm:"type:jsonb;serializer:json"`
	Hotness    int32       `gorm:"default:0"`
	RawMessage string      `gorm:"type:text"`
	Status     string      `gorm:"type:text"`
	Metadata   JSON        `gorm:"type:jsonb"`
}

// RemarkArgs represents the structured arguments from YAML
type RemarkArgs struct {
	Strings     []string          `json:"strings,omitempty"`
	Callee      string            `json:"callee,omitempty"`
	Caller      string            `json:"caller,omitempty"`
	Type        string            `json:"type,omitempty"`
	Line        string            `json:"line,omitempty"`
	Column      string            `json:"column,omitempty"`
	Cost        string            `json:"cost,omitempty"`
	Reason      string            `json:"reason,omitempty"`
	DebugLoc    *Location         `json:"debug_loc,omitempty"`
	OtherAccess *RemarkAccess     `json:"other_access,omitempty"`
	ClobberedBy *RemarkAccess     `json:"clobbered_by,omitempty"`
	Values      map[string]string `json:"values,omitempty"`
}

type RemarkAccess struct {
	Type     string    `json:"type,omitempty"`
	DebugLoc *Location `json:"debug_loc,omitempty"`
}

// Location represents source code location, now with optional fields
type Location struct {
	File     string `json:"file,omitempty"`
	Line     int32  `json:"line,omitempty"`
	Column   int32  `json:"column,omitempty"`
	Function string `json:"function,omitempty"`
	Region   string `json:"region,omitempty"`
}

// KernelInfo updates to better support YAML structure
type KernelInfo struct {
	ID                       uint `gorm:"primarykey"`
	RemarkID                 uint `gorm:"uniqueIndex"`
	ThreadLimit              int32
	MaxThreadsX              int32
	MaxThreadsY              int32
	MaxThreadsZ              int32
	SharedMemory             int64
	Target                   string
	DirectCalls              int32
	IndirectCalls            int32
	Callees                  StringArray `gorm:"type:text[]"`
	AllocasCount             int32
	AllocasStaticSize        int64
	AllocasDynamicCount      int32
	FlatAddressSpaceAccesses int32
	InlineAssemblyCalls      int32
	NumStackBytes            int64
	NumInstructions          int32
	Metrics                  JSON           `gorm:"type:jsonb"`
	Attributes               JSON           `gorm:"type:jsonb"`
	MemoryAccesses           []MemoryAccess `gorm:"foreignKey:KernelInfoID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	BasicBlocks              []BasicBlock   `gorm:"foreignKey:KernelInfoID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type MemoryAccess struct {
	ID            uint     `gorm:"primarykey"`
	KernelInfoID  uint     `gorm:"index"`
	Type          string   `gorm:"type:text"`
	AddressSpace  string   `gorm:"type:text"`
	Instruction   string   `gorm:"type:text"`
	Variable      string   `gorm:"type:text"`
	AccessPattern string   `gorm:"type:text"`
	Location      Location `gorm:"embedded;embeddedPrefix:mem_loc_"`
}

type BasicBlock struct {
	ID           uint   `gorm:"primarykey"`
	KernelInfoID uint   `gorm:"index"`
	Name         string `gorm:"type:text"`
	Instructions int32
	Location     Location `gorm:"embedded;embeddedPrefix:bb_loc_"`
}

type ResourceUsage struct {
	BuildID    string `gorm:"primarykey"`
	MaxMemory  int64
	CPUTime    float64
	Threads    int32
	ReadBytes  int64
	WriteBytes int64
	ReadCount  int64
	WriteCount int64
}

type Performance struct {
	BuildID      string `gorm:"primarykey"`
	CompileTime  float64
	LinkTime     float64
	OptimizeTime float64
	Phases       []PerformancePhase `gorm:"foreignKey:BuildID"`
}

type PerformancePhase struct {
	BuildID  string `gorm:"primarykey"`
	Phase    string `gorm:"primarykey"`
	Duration float64
}

// Custom types for handling arrays and JSON
type StringArray []string

func (a StringArray) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = nil
		return nil
	}
	return json.Unmarshal(value.([]byte), a)
}

type JSON map[string]interface{}

func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, j)
	case string:
		return json.Unmarshal([]byte(v), j)
	default:
		return fmt.Errorf("unsupported type: %T", value)
	}
}

// JSON marshaling for RemarkArgs
func (r RemarkArgs) Value() (driver.Value, error) {
	return json.Marshal(r)
}

func (r *RemarkArgs) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("invalid scan type for RemarkArgs: %T", value)
	}

	return json.Unmarshal(b, r)
}
