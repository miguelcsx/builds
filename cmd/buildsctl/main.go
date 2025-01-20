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

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	serverAddr = flag.String("server", "localhost:8080", "The server address")
	format     = flag.String("format", "text", "Output format (text, json)")
	watch      = flag.Bool("watch", false, "Watch for new builds")
	version    = flag.Bool("version", false, "Show version information")
)

const buildVersion = "0.1.0"

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("buildsctl version %s\n", buildVersion)
		return
	}

	conn, err := grpc.NewClient(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	printBuildDetails(w, build)
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

		printBuildDetails(w, build)
		fmt.Println("\n---")
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

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatDuration(d float64) string {
	duration := time.Duration(d * float64(time.Second))
	if duration < time.Second {
		return fmt.Sprintf("%dms", duration.Milliseconds())
	}
	return duration.Round(time.Millisecond).String()
}

func formatBool(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}
