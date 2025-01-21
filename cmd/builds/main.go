// cmd/builds/main.go

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	buildv1 "builds/api/build"
	"builds/internal/collectors/compiler"
	"builds/internal/collectors/environment"
	"builds/internal/collectors/hardware"
	"builds/internal/collectors/kernel"
	"builds/internal/collectors/remarks"
	"builds/internal/collectors/resource"
	"builds/internal/models"
	grpcutil "builds/internal/utils/grpcutil"
)

var (
	serverAddr = flag.String("server", "localhost:50051", "The server address") // Changed from 8080 to 50051
	useTLS     = flag.Bool("tls", false, "Use TLS when connecting to server")
	verbose    = flag.Bool("verbose", false, "Enable verbose output")
	version    = flag.Bool("version", false, "Show version information")
)

const buildVersion = "0.1.0"

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("builds version %s\n", buildVersion)
		return
	}

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] compiler [args...]\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	buildID := uuid.New().String()
	startTime := time.Now()

	// Create build context
	buildCtx := &models.BuildContext{
		Context:  context.Background(),
		BuildID:  buildID,
		Compiler: flag.Arg(0),
		Args:     flag.Args()[1:],
		Config: &models.CollectorConfig{
			Enabled:     true,
			Timeout:     300,
			MaxAttempts: 3,
		},
	}

	// Initialize collectors
	factory := models.NewCollectorFactory()
	factory.RegisterCollector("environment", environment.NewCollector())
	factory.RegisterCollector("hardware", hardware.NewCollector())
	factory.RegisterCollector("compiler", compiler.NewCollector(buildCtx))
	factory.RegisterCollector("kernel", kernel.NewCollector(buildCtx, os.Stderr))
	factory.RegisterCollector("remarks", remarks.NewCollector(buildCtx))
	factory.RegisterCollector("resource", resource.NewCollector(buildCtx))

	// Initialize and run collectors
	build := &buildv1.Build{
		Id:        buildID,
		StartTime: timestamppb.New(startTime),
	}

	ctx := context.Background()

	// Initialize collectors
	for name, collector := range factory.GetCollectors() {
		if err := collector.Initialize(ctx); err != nil {
			log.Printf("Warning: failed to initialize %s collector: %v", name, err)
			continue
		}
	}

	// Run collectors
	for name, collector := range factory.GetCollectors() {
		if err := collector.Collect(ctx); err != nil {
			log.Printf("Warning: collection failed for %s: %v", name, err)
			continue
		}

		// Store collected data
		if data := collector.GetData(); data != nil {
			switch name {
			case "environment":
				if env, ok := data.(models.Environment); ok {
					build.Environment = convertEnvironment(env)
				}
			case "hardware":
				if hw, ok := data.(models.Hardware); ok {
					build.Hardware = convertHardware(hw)
				}
			case "compiler":
				if comp, ok := data.(models.Compiler); ok {
					build.Compiler = convertCompiler(comp)
				}
			case "resource":
				if res, ok := data.(models.ResourceUsage); ok {
					build.ResourceUsage = convertResourceUsage(res)
				}
			case "remarks":
				if remarks, ok := data.([]models.CompilerRemark); ok {
					build.Remarks = convertRemarks(remarks)
				}
			}
		}
	}

	// Set end time and duration
	endTime := time.Now()
	build.EndTime = timestamppb.New(endTime)
	build.Duration = endTime.Sub(startTime).Seconds()

	// Connect to the server
	conn, err := grpcutil.CreateGRPCConnection(*serverAddr, *useTLS)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := buildv1.NewBuildServiceClient(conn)

	// Store build
	response, err := client.CreateBuild(ctx, &buildv1.CreateBuildRequest{
		Build: build,
	})
	if err != nil {
		log.Fatalf("Failed to store build: %v", err)
	}

	if *verbose {
		fmt.Printf("Build completed. Build ID: %s\n", response.Id)
		fmt.Printf("Build success: %v\n", build.Success)
		if build.Error != "" {
			fmt.Printf("Build error: %s\n", build.Error)
		}
	} else {
		fmt.Printf("Build ID: %s\n", response.Id)
	}
}

// Converter functions for collected data
func convertEnvironment(env models.Environment) *buildv1.Environment {
	variables := make(map[string]string)
	for _, v := range env.Variables {
		if pair := strings.SplitN(v, "=", 2); len(pair) == 2 {
			variables[pair[0]] = pair[1]
		}
	}

	return &buildv1.Environment{
		Os:         env.OS,
		Arch:       env.Arch,
		WorkingDir: env.WorkingDir,
		Variables:  variables,
	}
}

func convertHardware(hw models.Hardware) *buildv1.Hardware {
	gpus := make([]*buildv1.GPU, len(hw.GPUs))
	for i, gpu := range hw.GPUs {
		gpus[i] = &buildv1.GPU{
			Model:       gpu.Model,
			Memory:      gpu.Memory,
			Driver:      gpu.Driver,
			ComputeCaps: gpu.ComputeCaps,
		}
	}

	return &buildv1.Hardware{
		Cpu: &buildv1.CPU{
			Model:     hw.CPU.Model,
			Vendor:    hw.CPU.Vendor,
			Cores:     hw.CPU.Cores,
			Threads:   hw.CPU.Threads,
			Frequency: hw.CPU.Frequency,
			CacheSize: hw.CPU.CacheSize,
		},
		Memory: &buildv1.Memory{
			Total:     hw.Memory.Total,
			Available: hw.Memory.Available,
			Used:      hw.Memory.Used,
			SwapTotal: hw.Memory.SwapTotal,
			SwapFree:  hw.Memory.SwapFree,
		},
		Gpus: gpus,
	}
}

func convertCompiler(comp models.Compiler) *buildv1.Compiler {
	return &buildv1.Compiler{
		Name:    comp.Name,
		Version: comp.Version,
		Target:  comp.Target,
		Language: &buildv1.Language{
			Name:          comp.Language.Name,
			Version:       comp.Language.Version,
			Specification: comp.Language.Specification,
		},
		Features: &buildv1.CompilerFeatures{
			SupportsOpenmp: comp.Features.SupportsOpenMP,
			SupportsGpu:    comp.Features.SupportsGPU,
			SupportsLto:    comp.Features.SupportsLTO,
			SupportsPgo:    comp.Features.SupportsPGO,
			Extensions:     comp.Features.Extensions,
		},
		Options:       comp.Options,
		Optimizations: comp.Optimizations,
		Flags:         comp.Flags,
	}
}

func convertResourceUsage(res models.ResourceUsage) *buildv1.ResourceUsage {
	return &buildv1.ResourceUsage{
		MaxMemory: res.MaxMemory,
		CpuTime:   res.CPUTime,
		Threads:   res.Threads,
		Io: &buildv1.IOStats{
			ReadBytes:  res.IO.ReadBytes,
			WriteBytes: res.IO.WriteBytes,
			ReadCount:  res.IO.ReadCount,
			WriteCount: res.IO.WriteCount,
		},
	}
}

func convertRemarks(remarks []models.CompilerRemark) []*buildv1.CompilerRemark {
	pbRemarks := make([]*buildv1.CompilerRemark, len(remarks))
	for i, remark := range remarks {
		pbRemark := &buildv1.CompilerRemark{
			Id:       remark.ID,
			Message:  remark.Message,
			Function: remark.Function,
			Location: &buildv1.Location{
				File:     remark.Location.File,
				Line:     remark.Location.Line,
				Column:   remark.Location.Column,
				Function: remark.Location.Function,
				Region:   remark.Location.Region,
				Artifact: remark.Location.Artifact,
			},
			Timestamp: timestamppb.New(remark.Timestamp),
		}

		// Convert enums
		switch remark.Type {
		case models.RemarkTypeOptimization:
			pbRemark.Type = buildv1.CompilerRemark_OPTIMIZATION
		case models.RemarkTypeKernel:
			pbRemark.Type = buildv1.CompilerRemark_KERNEL
		case models.RemarkTypeAnalysis:
			pbRemark.Type = buildv1.CompilerRemark_ANALYSIS
		case models.RemarkTypeMetric:
			pbRemark.Type = buildv1.CompilerRemark_METRIC
		case models.RemarkTypeInfo:
			pbRemark.Type = buildv1.CompilerRemark_INFO
		}

		switch remark.Pass {
		case models.PassTypeVectorization:
			pbRemark.Pass = buildv1.CompilerRemark_VECTORIZATION
		case models.PassTypeInlining:
			pbRemark.Pass = buildv1.CompilerRemark_INLINING
		case models.PassTypeKernelInfo:
			pbRemark.Pass = buildv1.CompilerRemark_KERNEL_INFO
		case models.PassTypeSizeInfo:
			pbRemark.Pass = buildv1.CompilerRemark_SIZE_INFO
		case models.PassTypeAnalysis:
			pbRemark.Pass = buildv1.CompilerRemark_PASS_ANALYSIS
		}

		switch remark.Status {
		case models.RemarkStatusPassed:
			pbRemark.Status = buildv1.CompilerRemark_PASSED
		case models.RemarkStatusMissed:
			pbRemark.Status = buildv1.CompilerRemark_MISSED
		case models.RemarkStatusAnalysis:
			pbRemark.Status = buildv1.CompilerRemark_STATUS_ANALYSIS
		}

		// Convert kernel info if present
		if remark.KernelInfo != nil {
			memAccesses := make([]*buildv1.MemoryAccess, len(remark.KernelInfo.MemoryAccesses))
			for j, acc := range remark.KernelInfo.MemoryAccesses {
				memAccesses[j] = &buildv1.MemoryAccess{
					Type:          acc.Type,
					AddressSpace:  acc.AddressSpace,
					Instruction:   acc.Instruction,
					Variable:      acc.Variable,
					AccessPattern: acc.AccessPattern,
				}
			}

			pbRemark.KernelInfo = &buildv1.KernelInfo{
				ThreadLimit:              remark.KernelInfo.ThreadLimit,
				MaxThreadsX:              remark.KernelInfo.MaxThreadsX,
				MaxThreadsY:              remark.KernelInfo.MaxThreadsY,
				MaxThreadsZ:              remark.KernelInfo.MaxThreadsZ,
				SharedMemory:             remark.KernelInfo.SharedMemory,
				Target:                   remark.KernelInfo.Target,
				DirectCalls:              remark.KernelInfo.DirectCalls,
				IndirectCalls:            remark.KernelInfo.IndirectCalls,
				Callees:                  remark.KernelInfo.Callees,
				AllocasCount:             remark.KernelInfo.AllocasCount,
				AllocasStaticSize:        remark.KernelInfo.AllocasStaticSize,
				AllocasDynamicCount:      remark.KernelInfo.AllocasDynamicCount,
				FlatAddressSpaceAccesses: remark.KernelInfo.FlatAddressSpaceAccesses,
				InlineAssemblyCalls:      remark.KernelInfo.InlineAssemblyCalls,
				MemoryAccesses:           memAccesses,
				Metrics:                  remark.KernelInfo.Metrics,
				Attributes:               remark.KernelInfo.Attributes,
			}
		}

		// Convert metadata to structpb
		if len(remark.Metadata) > 0 {
			metadataJSON, err := json.Marshal(remark.Metadata)
			if err == nil {
				metadataStruct := &structpb.Struct{}
				if err := metadataStruct.UnmarshalJSON(metadataJSON); err == nil {
					pbRemark.Metadata = metadataStruct
				}
			}
		}

		pbRemarks[i] = pbRemark
	}
	return pbRemarks
}
