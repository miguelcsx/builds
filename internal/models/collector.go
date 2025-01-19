package models

import (
	"context"
)

// Collector defines the interface for data collectors
type Collector interface {
	// Initialize prepares the collector
	Initialize(ctx context.Context) error

	// Collect gathers data
	Collect(ctx context.Context) error

	// GetData returns the collected data
	GetData() interface{}

	// Cleanup performs any necessary cleanup
	Cleanup(ctx context.Context) error
}

// BaseCollector provides common functionality for collectors
type BaseCollector struct {
	Enabled bool
	Error   error
}

// CollectorConfig holds configuration for collectors
type CollectorConfig struct {
	Enabled     bool
	Timeout     int
	MaxAttempts int
	Options     map[string]interface{}
}

// BuildContext holds context for a build operation
type BuildContext struct {
	Context    context.Context
	BuildID    string
	SourceFile string
	OutputDir  string
	Compiler   string
	Args       []string
	Config     *CollectorConfig
}

// CollectorFactory manages collectors
type CollectorFactory struct {
	collectors map[string]Collector
}

// NewCollectorFactory creates a new collector factory
func NewCollectorFactory() *CollectorFactory {
	return &CollectorFactory{
		collectors: make(map[string]Collector),
	}
}

// RegisterCollector registers a collector
func (f *CollectorFactory) RegisterCollector(name string, collector Collector) {
	f.collectors[name] = collector
}

// GetCollector returns a specific collector
func (f *CollectorFactory) GetCollector(name string) (Collector, bool) {
	collector, exists := f.collectors[name]
	return collector, exists
}

// GetCollectors returns all registered collectors
func (f *CollectorFactory) GetCollectors() map[string]Collector {
	return f.collectors
}
