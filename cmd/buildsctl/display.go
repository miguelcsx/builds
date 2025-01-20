// cmd/buildsctl/display.go

package main

import (
	buildv1 "builds/api/build"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"
)

func printBuildDetails(w *tabwriter.Writer, build *buildv1.Build) {
	fmt.Fprintf(w, "Build Information\n")
	fmt.Fprintf(w, "=================\n")
	fmt.Fprintf(w, "Build ID:\t%s\n", build.Id)
	fmt.Fprintf(w, "Status:\t%v\n", build.Success)
	fmt.Fprintf(w, "Start Time:\t%s\n", build.StartTime.AsTime().Format(time.RFC3339))
	fmt.Fprintf(w, "End Time:\t%s\n", build.EndTime.AsTime().Format(time.RFC3339))
	fmt.Fprintf(w, "Duration:\t%.2fs\n", build.Duration)
	if build.Error != "" {
		fmt.Fprintf(w, "Error:\t%s\n", build.Error)
	}

	// Environment Information
	fmt.Fprintf(w, "\nEnvironment Information\n")
	fmt.Fprintf(w, "=====================\n")
	if build.Environment != nil {
		fmt.Fprintf(w, "OS:\t%s\n", build.Environment.Os)
		fmt.Fprintf(w, "Architecture:\t%s\n", build.Environment.Arch)
		fmt.Fprintf(w, "Working Directory:\t%s\n", build.Environment.WorkingDir)

		if len(build.Environment.Variables) > 0 {
			fmt.Fprintf(w, "\nEnvironment Variables:\n")
			for k, v := range build.Environment.Variables {
				fmt.Fprintf(w, "  %s:\t%s\n", k, v)
			}
		}
	}

	// Hardware Information
	fmt.Fprintf(w, "\nHardware Information\n")
	fmt.Fprintf(w, "===================\n")
	if build.Hardware != nil {
		if build.Hardware.Cpu != nil {
			fmt.Fprintf(w, "CPU Model:\t%s\n", build.Hardware.Cpu.Model)
			fmt.Fprintf(w, "CPU Vendor:\t%s\n", build.Hardware.Cpu.Vendor)
			fmt.Fprintf(w, "CPU Cores:\t%d\n", build.Hardware.Cpu.Cores)
			fmt.Fprintf(w, "CPU Threads:\t%d\n", build.Hardware.Cpu.Threads)
			fmt.Fprintf(w, "CPU Frequency:\t%.2f MHz\n", build.Hardware.Cpu.Frequency)
		}

		if build.Hardware.Memory != nil {
			fmt.Fprintf(w, "\nMemory Information:\n")
			fmt.Fprintf(w, "  Total:\t%d bytes\n", build.Hardware.Memory.Total)
			fmt.Fprintf(w, "  Available:\t%d bytes\n", build.Hardware.Memory.Available)
			fmt.Fprintf(w, "  Used:\t%d bytes\n", build.Hardware.Memory.Used)
			fmt.Fprintf(w, "  Swap Total:\t%d bytes\n", build.Hardware.Memory.SwapTotal)
			fmt.Fprintf(w, "  Swap Free:\t%d bytes\n", build.Hardware.Memory.SwapFree)
		}

		if len(build.Hardware.Gpus) > 0 {
			fmt.Fprintf(w, "\nGPU Information:\n")
			for i, gpu := range build.Hardware.Gpus {
				fmt.Fprintf(w, "  GPU %d:\n", i+1)
				fmt.Fprintf(w, "    Model:\t%s\n", gpu.Model)
				fmt.Fprintf(w, "    Memory:\t%d bytes\n", gpu.Memory)
				fmt.Fprintf(w, "    Driver:\t%s\n", gpu.Driver)
				fmt.Fprintf(w, "    Compute Capabilities:\t%s\n", gpu.ComputeCaps)
			}
		}
	}

	// Compiler Information
	fmt.Fprintf(w, "\nCompiler Information\n")
	fmt.Fprintf(w, "===================\n")
	if build.Compiler != nil {
		fmt.Fprintf(w, "Name:\t%s\n", build.Compiler.Name)
		fmt.Fprintf(w, "Version:\t%s\n", build.Compiler.Version)
		fmt.Fprintf(w, "Target:\t%s\n", build.Compiler.Target)

		if build.Compiler.Language != nil {
			fmt.Fprintf(w, "\nLanguage:\n")
			fmt.Fprintf(w, "  Name:\t%s\n", build.Compiler.Language.Name)
			fmt.Fprintf(w, "  Version:\t%s\n", build.Compiler.Language.Version)
			fmt.Fprintf(w, "  Specification:\t%s\n", build.Compiler.Language.Specification)
		}

		if build.Compiler.Features != nil {
			fmt.Fprintf(w, "\nFeatures:\n")
			fmt.Fprintf(w, "  OpenMP Support:\t%v\n", build.Compiler.Features.SupportsOpenmp)
			fmt.Fprintf(w, "  GPU Support:\t%v\n", build.Compiler.Features.SupportsGpu)
			fmt.Fprintf(w, "  LTO Support:\t%v\n", build.Compiler.Features.SupportsLto)
			fmt.Fprintf(w, "  PGO Support:\t%v\n", build.Compiler.Features.SupportsPgo)

			if len(build.Compiler.Features.Extensions) > 0 {
				fmt.Fprintf(w, "  Extensions:\t%s\n", strings.Join(build.Compiler.Features.Extensions, ", "))
			}
		}

		if len(build.Compiler.Options) > 0 {
			fmt.Fprintf(w, "\nCompiler Options:\t%s\n", strings.Join(build.Compiler.Options, " "))
		}

		if len(build.Compiler.Optimizations) > 0 {
			fmt.Fprintf(w, "\nOptimizations:\n")
			for name, enabled := range build.Compiler.Optimizations {
				fmt.Fprintf(w, "  %s:\t%v\n", name, enabled)
			}
		}
	}

	// Resource Usage
	fmt.Fprintf(w, "\nResource Usage\n")
	fmt.Fprintf(w, "==============\n")
	if build.ResourceUsage != nil {
		fmt.Fprintf(w, "Max Memory:\t%d bytes\n", build.ResourceUsage.MaxMemory)
		fmt.Fprintf(w, "CPU Time:\t%.2fs\n", build.ResourceUsage.CpuTime)
		fmt.Fprintf(w, "Threads:\t%d\n", build.ResourceUsage.Threads)

		if build.ResourceUsage.Io != nil {
			fmt.Fprintf(w, "\nIO Statistics:\n")
			fmt.Fprintf(w, "  Read:\t%d bytes (%d operations)\n",
				build.ResourceUsage.Io.ReadBytes,
				build.ResourceUsage.Io.ReadCount)
			fmt.Fprintf(w, "  Write:\t%d bytes (%d operations)\n",
				build.ResourceUsage.Io.WriteBytes,
				build.ResourceUsage.Io.WriteCount)
		}
	}

	// Performance Information
	fmt.Fprintf(w, "\nPerformance Information\n")
	fmt.Fprintf(w, "=====================\n")
	if build.Performance != nil {
		fmt.Fprintf(w, "Compile Time:\t%.2fs\n", build.Performance.CompileTime)
		fmt.Fprintf(w, "Link Time:\t%.2fs\n", build.Performance.LinkTime)
		fmt.Fprintf(w, "Optimize Time:\t%.2fs\n", build.Performance.OptimizeTime)

		if len(build.Performance.Phases) > 0 {
			fmt.Fprintf(w, "\nPhase Timings:\n")
			for phase, duration := range build.Performance.Phases {
				fmt.Fprintf(w, "  %s:\t%.2fs\n", phase, duration)
			}
		}
	}

	// Compiler Remarks
	if len(build.Remarks) > 0 {
		fmt.Fprintf(w, "\nCompiler Remarks\n")
		fmt.Fprintf(w, "================\n")

		// Group remarks by type for a summary
		remarksByType := make(map[string]int)
		for _, remark := range build.Remarks {
			remarksByType[remark.Type]++
		}

		fmt.Fprintf(w, "Summary:\n")
		for remarkType, count := range remarksByType {
			fmt.Fprintf(w, "  %s:\t%d remarks\n", remarkType, count)
		}

		fmt.Fprintf(w, "\nDetailed Remarks:\n")
		for _, remark := range build.Remarks {
			fmt.Fprintf(w, "  - [%s] %s\n", remark.Type, remark.Message)
			if remark.Location != nil {
				fmt.Fprintf(w, "    at %s:%d:%d\n",
					remark.Location.File,
					remark.Location.Line,
					remark.Location.Column)
			}
		}
	}
}
