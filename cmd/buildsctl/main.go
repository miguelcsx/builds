// cmd/buildsctl/main.go

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"text/tabwriter"
	"time"

	buildv1 "builds/api/build"
	"builds/internal/analysis/performance"
	"builds/internal/models"
	"builds/internal/reporters"

	grpcutil "builds/internal/utils/grpcutil"
)

var (
	serverAddr = flag.String("server", "localhost:50051", "The server address")
	format     = flag.String("format", "display", "Output format (display, text, json)")
	watch      = flag.Bool("watch", false, "Watch for new builds")
	useTLS     = flag.Bool("tls", false, "Use TLS when connecting to server")
	version    = flag.Bool("version", false, "Show version information")
)

const buildVersion = "0.1.0"

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("buildsctl version %s\n", buildVersion)
		return
	}

	conn, err := grpcutil.CreateGRPCConnection(*serverAddr, *useTLS)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := buildv1.NewBuildServiceClient(conn)

	if *watch {
		watchBuilds(client)
		return
	}

	args := flag.Args()
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch args[0] {
	case "get":
		if len(args) < 2 {
			log.Fatal("Build ID required")
		}
		getBuild(ctx, client, args[1])

	case "list":
		listBuilds(ctx, client)

	case "delete":
		if len(args) < 2 {
			log.Fatal("Build ID required")
		}
		deleteBuild(ctx, client, args[1])

	default:
		fmt.Printf("Unknown command: %s\n", args[0])
		printUsage()
		os.Exit(1)
	}
}

func getBuild(ctx context.Context, client buildv1.BuildServiceClient, id string) {
	build, err := client.GetBuild(ctx, &buildv1.GetBuildRequest{Id: id})
	if err != nil {
		log.Fatalf("Failed to get build: %v", err)
	}

	// Convert proto build to internal model
	modelBuild := convertProtoToModel(build)

	// Run analysis
	analyzer := performance.NewAnalyzer(modelBuild)
	analysisResult, err := analyzer.Analyze()
	if err != nil {
		log.Printf("Warning: analysis failed: %v", err)
	}

	// Create reporter options
	opts := reporters.Options{
		Format:   *format,
		Build:    modelBuild,
		Analysis: analysisResult,
		Writer:   os.Stdout,
	}

	// Create and use reporter
	reporter, err := reporters.NewReporter(opts)
	if err != nil {
		log.Fatalf("Failed to create reporter: %v", err)
	}

	if err := reporter.Generate(); err != nil {
		log.Fatalf("Failed to generate report: %v", err)
	}
}

func listBuilds(ctx context.Context, client buildv1.BuildServiceClient) {
	resp, err := client.ListBuilds(ctx, &buildv1.ListBuildsRequest{
		PageSize: 50,
	})
	if err != nil {
		log.Fatalf("Failed to list builds: %v", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, "BUILD ID\tSTATUS\tSTART TIME\tDURATION\tCOMPILER\n")
	for _, build := range resp.Builds {
		status := "Failed"
		if build.Success {
			status = "Success"
		}

		compilerName := "unknown"
		if build.Compiler != nil {
			compilerName = build.Compiler.Name
		}

		startTime := "N/A"
		if build.StartTime != nil {
			startTime = build.StartTime.AsTime().Format(time.RFC3339)
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%.2fs\t%s\n",
			build.Id,
			status,
			startTime,
			build.Duration,
			compilerName,
		)
	}

	if len(resp.Builds) == 0 {
		fmt.Println("No builds found")
	}
}

func deleteBuild(ctx context.Context, client buildv1.BuildServiceClient, id string) {
	_, err := client.DeleteBuild(ctx, &buildv1.DeleteBuildRequest{Id: id})
	if err != nil {
		log.Fatalf("Failed to delete build: %v", err)
	}
	fmt.Printf("Build %s deleted successfully\n", id)
}

func watchBuilds(client buildv1.BuildServiceClient) {
	ctx := context.Background()
	stream, err := client.StreamBuilds(ctx, &buildv1.StreamBuildsRequest{})
	if err != nil {
		log.Fatalf("Failed to watch builds: %v", err)
	}

	fmt.Println("Watching for new builds...")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	for {
		build, err := stream.Recv()
		if err != nil {
			log.Fatalf("Stream error: %v", err)
		}

		status := "Failed"
		if build.Success {
			status = "Success"
		}

		compilerName := "unknown"
		if build.Compiler != nil {
			compilerName = build.Compiler.Name
		}

		startTime := "N/A"
		if build.StartTime != nil {
			startTime = build.StartTime.AsTime().Format(time.RFC3339)
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%.2fs\t%s\n",
			build.Id,
			status,
			startTime,
			build.Duration,
			compilerName,
		)
		w.Flush()
	}
}

func printUsage() {
	fmt.Printf(`Usage: %s [options] <command> [arguments]

Commands:
  get <build-id>    Get details of a specific build
  list              List all builds
  delete <build-id> Delete a build

Options:
  -server string    The server address (default "localhost:50051")
  -format string    Output format (text, json) (default "text")
  -watch           Watch for new builds
  -version         Show version information

Examples:
  %[1]s get abc123                    # Get details of build abc123
  %[1]s list                          # List all builds
  %[1]s -watch                        # Watch for new builds
  %[1]s -server remote:50051 list     # List builds from remote server
`, os.Args[0], os.Args[0])
}

func convertProtoToModel(pb *buildv1.Build) *models.Build {
	build := &models.Build{
		ID:        pb.Id,
		StartTime: pb.StartTime.AsTime(),
		EndTime:   pb.EndTime.AsTime(),
		Duration:  pb.Duration,
		Success:   pb.Success,
		Error:     pb.Error,
	}

	// Convert Environment
	if pb.Environment != nil {
		build.Environment = models.Environment{
			OS:         pb.Environment.Os,
			Arch:       pb.Environment.Arch,
			WorkingDir: pb.Environment.WorkingDir,
			Variables:  pb.Environment.Variables, // This is already map[string]string
		}
	}

	// Convert Hardware
	if pb.Hardware != nil {
		build.Hardware = models.Hardware{
			CPU: models.CPU{
				Model:     pb.Hardware.Cpu.Model,
				Frequency: pb.Hardware.Cpu.Frequency,
				Cores:     pb.Hardware.Cpu.Cores,
				Threads:   pb.Hardware.Cpu.Threads,
				Vendor:    pb.Hardware.Cpu.Vendor,
				CacheSize: pb.Hardware.Cpu.CacheSize,
			},
			Memory: models.Memory{
				Total:     pb.Hardware.Memory.Total,
				Available: pb.Hardware.Memory.Available,
				Used:      pb.Hardware.Memory.Used,
				SwapTotal: pb.Hardware.Memory.SwapTotal,
				SwapFree:  pb.Hardware.Memory.SwapFree,
			},
			GPUs: make([]models.GPU, len(pb.Hardware.Gpus)),
		}

		for i, gpu := range pb.Hardware.Gpus {
			build.Hardware.GPUs[i] = models.GPU{
				Model:       gpu.Model,
				Memory:      gpu.Memory,
				Driver:      gpu.Driver,
				ComputeCaps: gpu.ComputeCaps,
			}
		}
	}

	// Convert Compiler
	if pb.Compiler != nil {
		build.Compiler = models.Compiler{
			Name:          pb.Compiler.Name,
			Version:       pb.Compiler.Version,
			Target:        pb.Compiler.Target,
			Options:       pb.Compiler.Options,
			Optimizations: pb.Compiler.Optimizations,
			Flags:         pb.Compiler.Flags,
			Language: models.Language{
				Name:          pb.Compiler.Language.Name,
				Version:       pb.Compiler.Language.Version,
				Specification: pb.Compiler.Language.Specification,
			},
			Extensions: pb.Compiler.Features.Extensions,
			Features: models.CompilerFeatures{
				SupportsOpenMP: pb.Compiler.Features.SupportsOpenmp,
				SupportsGPU:    pb.Compiler.Features.SupportsGpu,
				SupportsLTO:    pb.Compiler.Features.SupportsLto,
				SupportsPGO:    pb.Compiler.Features.SupportsPgo,
				Extensions:     pb.Compiler.Features.Extensions,
			},
		}
	}

	// Convert Command
	if pb.Command != nil {
		build.Command = models.Command{
			Executable: pb.Command.Executable,
			Arguments:  pb.Command.Arguments,
			WorkingDir: pb.Command.WorkingDir,
			Env:        pb.Command.Env,
		}
	}

	// Convert Output
	if pb.Output != nil {
		build.Output = models.Output{
			Stdout:    pb.Output.Stdout,
			Stderr:    pb.Output.Stderr,
			ExitCode:  pb.Output.ExitCode,
			Warnings:  pb.Output.Warnings,
			Errors:    pb.Output.Errors,
			Artifacts: make([]models.Artifact, len(pb.Output.Artifacts)),
		}
		for i, art := range pb.Output.Artifacts {
			build.Output.Artifacts[i] = models.Artifact{
				Path: art.Path,
				Type: art.Type,
				Size: art.Size,
				Hash: art.Hash,
			}
		}
	}

	// Convert Resource Usage
	if pb.ResourceUsage != nil {
		build.ResourceUsage = models.ResourceUsage{
			MaxMemory: pb.ResourceUsage.MaxMemory,
			CPUTime:   pb.ResourceUsage.CpuTime,
			Threads:   pb.ResourceUsage.Threads,
			IO: models.IOStats{
				ReadBytes:  pb.ResourceUsage.Io.ReadBytes,
				WriteBytes: pb.ResourceUsage.Io.WriteBytes,
				ReadCount:  pb.ResourceUsage.Io.ReadCount,
				WriteCount: pb.ResourceUsage.Io.WriteCount,
			},
		}
	}

	// Convert Performance
	if pb.Performance != nil {
		build.Performance = models.Performance{
			CompileTime:  pb.Performance.CompileTime,
			LinkTime:     pb.Performance.LinkTime,
			OptimizeTime: pb.Performance.OptimizeTime,
			Phases:       pb.Performance.Phases,
		}
	}

	// Convert Remarks
	remarks := make([]models.CompilerRemark, 0, len(pb.Remarks))
	for _, remark := range pb.Remarks {
		modelRemark := models.CompilerRemark{
			ID:        remark.Id,
			Type:      models.RemarkType(remark.Type.String()),
			Pass:      models.PassType(remark.Pass.String()),
			Status:    models.RemarkStatus(remark.Status.String()),
			Message:   remark.Message,
			Function:  remark.Function,
			Timestamp: remark.Timestamp.AsTime(),
			Location: models.Location{
				File:     remark.Location.File,
				Line:     remark.Location.Line,
				Column:   remark.Location.Column,
				Function: remark.Location.Function,
				Region:   remark.Location.Region,
				Artifact: remark.Location.Artifact,
			},
		}

		if remark.KernelInfo != nil {
			modelRemark.KernelInfo = &models.KernelInfo{
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
				Metrics:                  remark.KernelInfo.Metrics,
				Attributes:               remark.KernelInfo.Attributes,
				MemoryAccesses:           make([]models.MemoryAccess, len(remark.KernelInfo.MemoryAccesses)),
			}

			for i, acc := range remark.KernelInfo.MemoryAccesses {
				modelRemark.KernelInfo.MemoryAccesses[i] = models.MemoryAccess{
					Type:          acc.Type,
					AddressSpace:  acc.AddressSpace,
					Instruction:   acc.Instruction,
					Variable:      acc.Variable,
					AccessPattern: acc.AccessPattern,
				}
			}
		}

		if remark.Metadata != nil {
			modelRemark.Metadata = remark.Metadata.AsMap()
		}

		remarks = append(remarks, modelRemark)
	}
	build.Remarks = remarks

	return build
}
