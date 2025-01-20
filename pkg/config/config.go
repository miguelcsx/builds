// pkg/config/config.go

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config represents the global configuration
type Config struct {
	// Build settings
	BuildDir  string `json:"buildDir"`  // Directory for build outputs
	CacheDir  string `json:"cacheDir"`  // Cache directory
	MaxBuilds int    `json:"maxBuilds"` // Maximum number of builds to keep

	// Compiler settings
	DefaultCompiler string            `json:"defaultCompiler"` // Default compiler to use
	CompilerPaths   map[string]string `json:"compilerPaths"`   // Paths to different compilers

	// Collection settings
	CollectHardwareInfo bool `json:"collectHardwareInfo"` // Collect hardware information
	CollectResourceInfo bool `json:"collectResourceInfo"` // Collect resource usage
	CollectKernelInfo   bool `json:"collectKernelInfo"`   // Collect kernel information
	CollectTimeTrace    bool `json:"collectTimeTrace"`    // Collect time trace information

	// Analysis settings
	AnalyzeOptimizations bool `json:"analyzeOptimizations"` // Analyze optimization decisions
	AnalyzePerformance   bool `json:"analyzePerformance"`   // Analyze performance metrics

	// Reporter settings
	OutputFormat string `json:"outputFormat"` // Output format (html, json, etc.)
	ReportDir    string `json:"reportDir"`    // Directory for generated reports
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		BuildDir:             "builds",
		CacheDir:             "cache",
		MaxBuilds:            100,
		DefaultCompiler:      "clang",
		CompilerPaths:        map[string]string{"clang": "clang", "gcc": "gcc"},
		CollectHardwareInfo:  true,
		CollectResourceInfo:  true,
		CollectKernelInfo:    true,
		CollectTimeTrace:     true,
		AnalyzeOptimizations: true,
		AnalyzePerformance:   true,
		OutputFormat:         "html",
		ReportDir:            "reports",
	}
}

// LoadConfig loads configuration from a file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig(), err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return DefaultConfig(), err
	}

	return &config, nil
}

// SaveConfig saves configuration to a file
func (c *Config) SaveConfig(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
