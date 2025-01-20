// internal/collectors/hardware/collector.go

package hardware

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"

	"builds/internal/models"
)

// Collector implements hardware information collection
type Collector struct {
	models.BaseCollector
	info models.Hardware
}

// NewCollector creates a new hardware collector
func NewCollector() *Collector {
	return &Collector{}
}

// Initialize prepares the hardware collector
func (c *Collector) Initialize(ctx context.Context) error {
	return nil
}

// Collect gathers hardware information
func (c *Collector) Collect(ctx context.Context) error {
	// Collect CPU information
	cpuInfo, err := c.collectCPUInfo()
	if err != nil {
		return err
	}
	c.info.CPU = cpuInfo

	// Collect memory information
	memInfo, err := c.collectMemoryInfo()
	if err != nil {
		return err
	}
	c.info.Memory = memInfo

	// Collect GPU information
	gpus, err := c.collectGPUInfo()
	if err != nil {
		return err
	}
	c.info.GPUs = gpus

	return nil
}

// GetData returns the collected hardware information
func (c *Collector) GetData() interface{} {
	return c.info
}

// Cleanup performs any necessary cleanup
func (c *Collector) Cleanup(ctx context.Context) error {
	return nil
}

// collectCPUInfo gathers CPU information
func (c *Collector) collectCPUInfo() (models.CPU, error) {
	var cpuInfo models.CPU

	info, err := cpu.Info()
	if err != nil {
		return cpuInfo, err
	}

	if len(info) > 0 {
		cpuInfo.Model = info[0].ModelName
		cpuInfo.Vendor = info[0].VendorID
		cpuInfo.Frequency = float64(info[0].Mhz)
		cpuInfo.Cores = int32(runtime.NumCPU())
		cpuInfo.Threads = int32(runtime.GOMAXPROCS(0))
		// CPU cache size might require platform-specific code
		cpuInfo.CacheSize = 0 // This would need to be implemented based on platform
	}

	return cpuInfo, nil
}

// collectMemoryInfo gathers memory information
func (c *Collector) collectMemoryInfo() (models.Memory, error) {
	var memInfo models.Memory

	virtualMemory, err := mem.VirtualMemory()
	if err != nil {
		return memInfo, err
	}

	swapMemory, err := mem.SwapMemory()
	if err != nil {
		return memInfo, err
	}

	memInfo.Total = int64(virtualMemory.Total)
	memInfo.Available = int64(virtualMemory.Available)
	memInfo.SwapTotal = int64(swapMemory.Total)
	memInfo.SwapFree = int64(swapMemory.Free)
	memInfo.Used = int64(virtualMemory.Used)

	return memInfo, nil
}

// collectGPUInfo gathers GPU information
func (c *Collector) collectGPUInfo() ([]models.GPU, error) {
	var gpus []models.GPU

	// Try NVIDIA-SMI first
	if nvidiaGPUs, err := c.collectNvidiaGPUInfo(); err == nil {
		gpus = append(gpus, nvidiaGPUs...)
	}

	// Try AMD ROCm
	if amdGPUs, err := c.collectAMDGPUInfo(); err == nil {
		gpus = append(gpus, amdGPUs...)
	}

	return gpus, nil
}

// collectNvidiaGPUInfo gathers NVIDIA GPU information using nvidia-smi
func (c *Collector) collectNvidiaGPUInfo() ([]models.GPU, error) {
	var gpus []models.GPU

	// Execute nvidia-smi and parse output
	cmd := exec.Command("nvidia-smi", "--query-gpu=gpu_name,memory.total,driver_version,compute_cap", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ", ")
		if len(fields) == 4 {
			memory, _ := strconv.ParseInt(strings.TrimSpace(fields[1]), 10, 64)
			gpus = append(gpus, models.GPU{
				Model:       strings.TrimSpace(fields[0]),
				Memory:      memory * 1024 * 1024, // Convert MB to bytes
				Driver:      strings.TrimSpace(fields[2]),
				ComputeCaps: strings.TrimSpace(fields[3]),
			})
		}
	}

	return gpus, nil
}

// collectAMDGPUInfo gathers AMD GPU information using rocm-smi
func (c *Collector) collectAMDGPUInfo() ([]models.GPU, error) {
	var gpus []models.GPU

	// Execute rocm-smi and parse output
	cmd := exec.Command("rocm-smi", "--showproductname", "--showmeminfo", "--showdriver")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	var currentGPU models.GPU

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "GPU[") {
			if currentGPU.Model != "" {
				gpus = append(gpus, currentGPU)
				currentGPU = models.GPU{}
			}
		} else if strings.Contains(line, "Product Name") {
			currentGPU.Model = strings.TrimSpace(strings.Split(line, ":")[1])
		} else if strings.Contains(line, "Driver Version") {
			currentGPU.Driver = strings.TrimSpace(strings.Split(line, ":")[1])
		} else if strings.Contains(line, "Memory") {
			memStr := strings.TrimSpace(strings.Split(line, ":")[1])
			memory, _ := strconv.ParseInt(strings.Fields(memStr)[0], 10, 64)
			currentGPU.Memory = memory * 1024 * 1024 // Convert MB to bytes
		}
	}

	if currentGPU.Model != "" {
		gpus = append(gpus, currentGPU)
	}

	return gpus, nil
}

// GetGPUDevices returns a list of available GPU devices
func (c *Collector) GetGPUDevices() []string {
	var devices []string

	// Check for NVIDIA GPUs
	if _, err := exec.LookPath("nvidia-smi"); err == nil {
		cmd := exec.Command("nvidia-smi", "-L")
		if output, err := cmd.Output(); err == nil {
			scanner := bufio.NewScanner(bytes.NewReader(output))
			for scanner.Scan() {
				if strings.HasPrefix(scanner.Text(), "GPU ") {
					devices = append(devices, "nvidia")
				}
			}
		}
	}

	// Check for AMD GPUs
	if _, err := exec.LookPath("rocm-smi"); err == nil {
		cmd := exec.Command("rocm-smi", "-l")
		if output, err := cmd.Output(); err == nil {
			scanner := bufio.NewScanner(bytes.NewReader(output))
			for scanner.Scan() {
				if strings.Contains(scanner.Text(), "GPU[") {
					devices = append(devices, "amd")
				}
			}
		}
	}

	return devices
}
