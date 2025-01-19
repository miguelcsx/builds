package performance

import (
	"strconv"
	"strings"

	"builds/internal/models"
)

type Analyzer struct {
	build *models.Build
}

func NewAnalyzer(build *models.Build) *Analyzer {
	return &Analyzer{
		build: build,
	}
}

type AnalysisResult struct {
	ResourceEfficiency  float64                     `json:"resourceEfficiency"`
	MemoryUsageProfile  map[string]int64            `json:"memoryUsageProfile"`
	CompilationOverhead map[string]float64          `json:"compilationOverhead"`
	OptimizationMetrics map[string]int              `json:"optimizationMetrics"`
	Bottlenecks         []PerformanceBottleneck     `json:"bottlenecks"`
	Recommendations     []PerformanceRecommendation `json:"recommendations"`
}

type PerformanceBottleneck struct {
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	Impact      float64 `json:"impact"`
}

type PerformanceRecommendation struct {
	Category string `json:"category"`
	Action   string `json:"action"`
	Impact   string `json:"impact"`
	Details  string `json:"details"`
}

func (a *Analyzer) Analyze() (*AnalysisResult, error) {
	result := &AnalysisResult{
		MemoryUsageProfile:  make(map[string]int64),
		CompilationOverhead: make(map[string]float64),
		OptimizationMetrics: make(map[string]int),
	}

	result.ResourceEfficiency = a.calculateResourceEfficiency()
	result.MemoryUsageProfile = a.analyzeMemoryUsage()
	result.CompilationOverhead = a.analyzeCompilationOverhead()
	result.OptimizationMetrics = a.analyzeOptimizationMetrics()
	result.Bottlenecks = a.identifyBottlenecks()
	result.Recommendations = a.generateRecommendations(result.Bottlenecks)

	return result, nil
}

func (a *Analyzer) calculateResourceEfficiency() float64 {
	if a.build.ResourceUsage.CPUTime == 0 {
		return 0
	}

	// Calculate efficiency based on CPU utilization and memory usage
	cpuEfficiency := float64(a.build.ResourceUsage.Threads) / float64(a.build.Hardware.CPU.Cores)
	memoryEfficiency := float64(a.build.ResourceUsage.MaxMemory) / float64(a.build.Hardware.Memory.Total)

	return (cpuEfficiency + memoryEfficiency) / 2.0
}

func (a *Analyzer) analyzeMemoryUsage() map[string]int64 {
	usage := make(map[string]int64)

	usage["peak"] = a.build.ResourceUsage.MaxMemory
	usage["average"] = a.build.ResourceUsage.MaxMemory / 2 // Simplified estimation
	usage["allocated"] = a.build.ResourceUsage.MaxMemory
	usage["wasted"] = a.calculateWastedMemory()

	return usage
}

func (a *Analyzer) calculateWastedMemory() int64 {
	var wastedMemory int64

	// Check for large allocations in compiler remarks
	for _, remark := range a.build.Remarks {
		if strings.Contains(remark.Message, "alloca") {
			// Look for memory size in remark args
			for _, arg := range remark.Args {
				if arg.String != "" {
					if size, err := strconv.ParseInt(arg.String, 10, 64); err == nil {
						wastedMemory += size
					}
				}
			}
		}
	}

	return wastedMemory
}

func (a *Analyzer) analyzeCompilationOverhead() map[string]float64 {
	overhead := make(map[string]float64)

	overhead["parsing"] = a.build.Performance.CompileTime * 0.2 // Estimated
	overhead["optimization"] = a.build.Performance.OptimizeTime
	overhead["codegen"] = a.build.Performance.CompileTime * 0.4 // Estimated
	overhead["linking"] = a.build.Performance.LinkTime

	return overhead
}

func (a *Analyzer) analyzeOptimizationMetrics() map[string]int {
	metrics := make(map[string]int)

	// Count optimization remarks by type
	for _, remark := range a.build.Remarks {
		switch remark.Type {
		case "Passed":
			metrics["successful_optimizations"]++
		case "Missed":
			metrics["missed_optimizations"]++
		case "Analysis":
			metrics["analysis_remarks"]++
		}
	}

	return metrics
}

func (a *Analyzer) identifyBottlenecks() []PerformanceBottleneck {
	var bottlenecks []PerformanceBottleneck

	// Check memory usage
	memoryUtilization := float64(a.build.ResourceUsage.MaxMemory) / float64(a.build.Hardware.Memory.Total)
	if memoryUtilization > 0.9 {
		bottlenecks = append(bottlenecks, PerformanceBottleneck{
			Type:        "memory",
			Severity:    "high",
			Description: "High memory utilization",
			Impact:      memoryUtilization,
		})
	}

	// Check compilation time
	if a.build.Performance.CompileTime > 60.0 {
		bottlenecks = append(bottlenecks, PerformanceBottleneck{
			Type:        "compilation",
			Severity:    "medium",
			Description: "Long compilation time",
			Impact:      a.build.Performance.CompileTime,
		})
	}

	// Check optimization effectiveness
	missedOpts := 0
	for _, remark := range a.build.Remarks {
		if remark.Type == "Missed" {
			missedOpts++
		}
	}
	if missedOpts > 10 {
		bottlenecks = append(bottlenecks, PerformanceBottleneck{
			Type:        "optimization",
			Severity:    "low",
			Description: "High number of missed optimizations",
			Impact:      float64(missedOpts),
		})
	}

	return bottlenecks
}

func (a *Analyzer) generateRecommendations(bottlenecks []PerformanceBottleneck) []PerformanceRecommendation {
	var recommendations []PerformanceRecommendation

	for _, bottleneck := range bottlenecks {
		switch bottleneck.Type {
		case "memory":
			recommendations = append(recommendations, PerformanceRecommendation{
				Category: "Memory Usage",
				Action:   "Consider reducing static allocations",
				Impact:   "High",
				Details:  "Large static allocations detected. Consider using dynamic allocation or reducing buffer sizes.",
			})

		case "compilation":
			recommendations = append(recommendations, PerformanceRecommendation{
				Category: "Build Performance",
				Action:   "Enable parallel compilation",
				Impact:   "Medium",
				Details:  "Long compilation time detected. Consider using -j flag or distributed compilation.",
			})

		case "optimization":
			recommendations = append(recommendations, PerformanceRecommendation{
				Category: "Optimization",
				Action:   "Review missed optimization opportunities",
				Impact:   "Medium",
				Details:  "Multiple optimization opportunities were missed. Consider reviewing the code structure.",
			})
		}
	}

	return recommendations
}
