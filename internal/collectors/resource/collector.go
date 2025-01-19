package resource

import (
	"context"
	"os"
	"runtime"
	"time"

	"builds/internal/models"

	"github.com/shirou/gopsutil/v3/process"
)

// Collector implements resource usage collection
type Collector struct {
	models.BaseCollector
	info         models.ResourceUsage
	startTime    time.Time
	proc         *process.Process
	buildContext *models.BuildContext
}

// NewCollector creates a new resource usage collector
func NewCollector(ctx *models.BuildContext) *Collector {
	return &Collector{
		buildContext: ctx,
		startTime:    time.Now(),
	}
}

// Initialize prepares the resource collector
func (c *Collector) Initialize(ctx context.Context) error {
	proc, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return err
	}
	c.proc = proc

	// Initialize statistics
	c.info.ThreadCount = runtime.GOMAXPROCS(0)
	return nil
}

// Collect gathers resource usage information
func (c *Collector) Collect(ctx context.Context) error {
	// Get memory info
	memInfo, err := c.proc.MemoryInfo()
	if err != nil {
		return err
	}
	c.info.MaxMemory = int64(memInfo.RSS)

	// Get CPU times
	cpuTimes, err := c.proc.Times()
	if err != nil {
		return err
	}
	c.info.CPUTime = cpuTimes.User + cpuTimes.System

	// Get IO statistics
	ioStats, err := c.proc.IOCounters()
	if err == nil {
		c.info.IOStats = models.IOStats{
			ReadBytes:    int64(ioStats.ReadBytes),
			WrittenBytes: int64(ioStats.WriteBytes),
			ReadCount:    int64(ioStats.ReadCount),
			WriteCount:   int64(ioStats.WriteCount),
		}
	}

	// Get thread count
	threads, err := c.proc.NumThreads()
	if err == nil {
		c.info.ThreadCount = int(threads)
	}

	return nil
}

// GetData returns the collected resource usage information
func (c *Collector) GetData() interface{} {
	return c.info
}

// Cleanup performs any necessary cleanup
func (c *Collector) Cleanup(ctx context.Context) error {
	// Perform one final collection before cleanup
	if err := c.Collect(ctx); err != nil {
		return err
	}
	return nil
}

// StartTracking begins resource tracking
func (c *Collector) StartTracking() error {
	c.startTime = time.Now()
	return nil
}

// StopTracking ends resource tracking and updates statistics
func (c *Collector) StopTracking() error {
	return c.Collect(context.Background())
}

// GetResourceSnapshot takes a snapshot of current resource usage
func (c *Collector) GetResourceSnapshot() (*models.ResourceUsage, error) {
	err := c.Collect(context.Background())
	if err != nil {
		return nil, err
	}

	snapshot := c.info

	return &snapshot, nil
}
