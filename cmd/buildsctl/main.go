// cmd/buildsctl/main.go

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
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
	verbose    = flag.Bool("verbose", false, "Enable verbose output")
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

	case "inspect":
		if len(args) < 2 {
			log.Fatal("Build ID required")
		}
		inspectBuild(ctx, client, args[1])

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
  inspect <build-id> Inspect a build in detail

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
	if pb == nil {
		return nil
	}

	build := &models.Build{
		ID:      pb.Id,
		Success: pb.Success,
		Error:   pb.Error,
	}

	// Handle timestamps safely
	if pb.StartTime != nil {
		build.StartTime = pb.StartTime.AsTime()
	}
	if pb.EndTime != nil {
		build.EndTime = pb.EndTime.AsTime()
	}
	build.Duration = pb.Duration

	// Convert Environment
	if pb.Environment != nil {
		build.Environment = models.Environment{
			OS:         pb.Environment.Os,
			Arch:       pb.Environment.Arch,
			WorkingDir: pb.Environment.WorkingDir,
			Variables:  pb.Environment.Variables,
		}
	}

	// Convert Hardware
	if pb.Hardware != nil && pb.Hardware.Cpu != nil && pb.Hardware.Memory != nil {
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
		}

		// Handle GPUs safely
		if pb.Hardware.Gpus != nil {
			build.Hardware.GPUs = make([]models.GPU, len(pb.Hardware.Gpus))
			for i, gpu := range pb.Hardware.Gpus {
				if gpu != nil {
					build.Hardware.GPUs[i] = models.GPU{
						Model:       gpu.Model,
						Memory:      gpu.Memory,
						Driver:      gpu.Driver,
						ComputeCaps: gpu.ComputeCaps,
					}
				}
			}
		}
	}

	// Convert Remarks
	if pb.Remarks != nil {
		build.Remarks = make([]models.CompilerRemark, 0, len(pb.Remarks))
		for _, remark := range pb.Remarks {
			if remark == nil {
				continue
			}

			modelRemark := models.CompilerRemark{
				Type:     strings.ToLower(remark.Type.String()),
				Pass:     strings.ToLower(remark.Pass.String()),
				Status:   strings.ToLower(remark.Status.String()),
				Message:  remark.Message,
				Function: remark.Function,
				Hotness:  remark.Hotness,
			}

			if remark.Timestamp != nil {
				modelRemark.Timestamp = remark.Timestamp.AsTime()
			}

			// Handle Location
			if remark.Location != nil {
				modelRemark.Location = models.Location{
					File:     remark.Location.File,
					Line:     remark.Location.Line,
					Column:   remark.Location.Column,
					Function: remark.Location.Function,
					Region:   remark.Location.Region,
				}
			}

			// Handle KernelInfo
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
					Metrics:                  make(map[string]int64),
					Attributes:               make(map[string]string),
				}

				// Copy metrics
				if remark.KernelInfo.Metrics != nil {
					for k, v := range remark.KernelInfo.Metrics {
						modelRemark.KernelInfo.Metrics[k] = v
					}
				}

				// Copy attributes
				if remark.KernelInfo.Attributes != nil {
					for k, v := range remark.KernelInfo.Attributes {
						modelRemark.KernelInfo.Attributes[k] = v
					}
				}

				// Handle memory accesses
				if remark.KernelInfo.MemoryAccesses != nil {
					modelRemark.KernelInfo.MemoryAccesses = make([]models.MemoryAccess, len(remark.KernelInfo.MemoryAccesses))
					for i, acc := range remark.KernelInfo.MemoryAccesses {
						if acc != nil {
							modelRemark.KernelInfo.MemoryAccesses[i] = models.MemoryAccess{
								Type:          acc.Type,
								AddressSpace:  acc.AddressSpace,
								Instruction:   acc.Instruction,
								Variable:      acc.Variable,
								AccessPattern: acc.AccessPattern,
							}
						}
					}
				}
			}

			// Handle metadata
			if remark.Metadata != nil {
				modelRemark.Metadata = remark.Metadata.AsMap()
			}

			// Handle args
			if remark.Args != nil {
				modelRemark.Args.Strings = remark.Args.Strings
				modelRemark.Args.Values = make(map[string]string)
			}

			build.Remarks = append(build.Remarks, modelRemark)
		}
	}

	return build
}

func inspectBuild(ctx context.Context, client buildv1.BuildServiceClient, id string) {
	build, err := client.GetBuild(ctx, &buildv1.GetBuildRequest{Id: id})
	if err != nil {
		log.Fatalf("Failed to get build: %v", err)
	}

	// Create a detailed inspection report
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, "Database Inspection for Build %s\n", build.Id)
	fmt.Fprintf(w, "=================================\n\n")

	// Main Build table
	fmt.Fprintf(w, "Build Table:\n")
	fmt.Fprintf(w, "  ID:\t%s\n", build.Id)
	fmt.Fprintf(w, "  Success:\t%v\n", build.Success)
	fmt.Fprintf(w, "  Duration:\t%.2f\n", build.Duration)
	fmt.Fprintf(w, "\n")

	// Remarks table
	fmt.Fprintf(w, "Compiler Remarks (%d remarks):\n", len(build.Remarks))
	if len(build.Remarks) > 0 {
		fmt.Fprintf(w, "  ID\tType\tPass\tStatus\tMessage\tLocation\n")
		fmt.Fprintf(w, "  --\t----\t----\t------\t-------\t--------\n")
		for _, remark := range build.Remarks {
			location := fmt.Sprintf("%s:%d:%d",
				remark.Location.File,
				remark.Location.Line,
				remark.Location.Column)
			fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\t%s\n",
				remark.Id,
				remark.Type,
				remark.Pass,
				remark.Status,
				truncate(remark.Message, 30),
				location)
		}
	} else {
		fmt.Fprintf(w, "  No remarks found in database\n")
	}
	fmt.Fprintf(w, "\n")

	// Show raw data if -verbose flag is set
	if *verbose {
		fmt.Fprintf(w, "Raw Remark Data:\n")
		for i, remark := range build.Remarks {
			fmt.Fprintf(w, "Remark %d:\n", i+1)
			fmt.Fprintf(w, "  Message:\t%s\n", remark.Message)
			fmt.Fprintf(w, "  Function:\t%s\n", remark.Function)
			fmt.Fprintf(w, "  Location:\t%s:%d:%d\n",
				remark.Location.File,
				remark.Location.Line,
				remark.Location.Column)
			if remark.KernelInfo != nil {
				fmt.Fprintf(w, "  Kernel Info:\n")
				fmt.Fprintf(w, "    Thread Limit:\t%d\n", remark.KernelInfo.ThreadLimit)
				fmt.Fprintf(w, "    Direct Calls:\t%d\n", remark.KernelInfo.DirectCalls)
				fmt.Fprintf(w, "    Memory Accesses:\t%d\n", len(remark.KernelInfo.MemoryAccesses))
			}
			if remark.Metadata != nil {
				fmt.Fprintf(w, "  Metadata:\n")
				for k, v := range remark.Metadata.AsMap() {
					fmt.Fprintf(w, "    %s:\t%v\n", k, v)
				}
			}
			fmt.Fprintf(w, "\n")
		}
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
