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
	info models.HardwareInfo
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
	gpuInfo, err := c.collectGPUInfo()
	if err != nil {
		return err
	}
	c.info.GPU = gpuInfo

	// Set basic system information
	c.info.NumCores = runtime.NumCPU()
	c.info.NumThreads = runtime.GOMAXPROCS(0)

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
func (c *Collector) collectCPUInfo() (models.CPUInfo, error) {
	var cpuInfo models.CPUInfo

	info, err := cpu.Info()
	if err != nil {
		return cpuInfo, err
	}

	if len(info) > 0 {
		cpuInfo.Model = info[0].ModelName
		cpuInfo.Vendor = info[0].VendorID
		cpuInfo.Frequency = float64(info[0].Mhz)
		// CPU cache size might require platform-specific code
	}

	return cpuInfo, nil
}

// collectMemoryInfo gathers memory information
func (c *Collector) collectMemoryInfo() (models.MemoryInfo, error) {
	var memInfo models.MemoryInfo

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

	return memInfo, nil
}

// collectGPUInfo gathers GPU information
func (c *Collector) collectGPUInfo() ([]models.GPUInfo, error) {
	var gpuInfos []models.GPUInfo

	// Try NVIDIA-SMI first
	if gpus, err := c.collectNvidiaGPUInfo(); err == nil {
		gpuInfos = append(gpuInfos, gpus...)
	}

	// Try AMD ROCm
	if gpus, err := c.collectAMDGPUInfo(); err == nil {
		gpuInfos = append(gpuInfos, gpus...)
	}

	return gpuInfos, nil
}

// collectNvidiaGPUInfo gathers NVIDIA GPU information using nvidia-smi
func (c *Collector) collectNvidiaGPUInfo() ([]models.GPUInfo, error) {
	var gpuInfos []models.GPUInfo

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
			gpuInfos = append(gpuInfos, models.GPUInfo{
				Model:       strings.TrimSpace(fields[0]),
				Memory:      memory * 1024 * 1024, // Convert MB to bytes
				Driver:      strings.TrimSpace(fields[2]),
				ComputeCaps: strings.TrimSpace(fields[3]),
			})
		}
	}

	return gpuInfos, nil
}

// collectAMDGPUInfo gathers AMD GPU information using rocm-smi
func (c *Collector) collectAMDGPUInfo() ([]models.GPUInfo, error) {
	var gpuInfos []models.GPUInfo

	// Execute rocm-smi and parse output
	cmd := exec.Command("rocm-smi", "--showproductname", "--showmeminfo", "--showdriver")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Parse rocm-smi output (implementation depends on output format)
	// This is a simplified example
	scanner := bufio.NewScanner(bytes.NewReader(output))
	var currentGPU models.GPUInfo

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "GPU[") {
			if currentGPU.Model != "" {
				gpuInfos = append(gpuInfos, currentGPU)
				currentGPU = models.GPUInfo{}
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
		gpuInfos = append(gpuInfos, currentGPU)
	}

	return gpuInfos, nil
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
