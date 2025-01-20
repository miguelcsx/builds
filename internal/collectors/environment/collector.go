// internal/collectors/environment/collector.go

package environment

import (
	"context"
	"os"
	"runtime"
	"strings"

	"builds/internal/models"
)

// Collector implements environment information collection
type Collector struct {
	models.BaseCollector
	info models.Environment
}

// NewCollector creates a new environment collector
func NewCollector() *Collector {
	return &Collector{}
}

// Initialize prepares the environment collector
func (c *Collector) Initialize(ctx context.Context) error {
	return nil
}

// Collect gathers environment information
func (c *Collector) Collect(ctx context.Context) error {
	// Get OS and architecture
	c.info.OS = runtime.GOOS
	c.info.Arch = runtime.GOARCH

	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	c.info.WorkingDir = wd

	// Get environment variables
	c.info.Variables = make(map[string]string)
	for _, env := range os.Environ() {
		if key, value, ok := splitEnv(env); ok {
			// Filter sensitive environment variables
			if !isSensitiveEnv(key) {
				c.info.Variables[key] = value
			}
		}
	}

	return nil
}

// GetData returns the collected environment information
func (c *Collector) GetData() interface{} {
	return c.info
}

// Cleanup performs any necessary cleanup
func (c *Collector) Cleanup(ctx context.Context) error {
	return nil
}

// splitEnv splits environment variable into key and value
func splitEnv(env string) (key, value string, ok bool) {
	parts := strings.SplitN(env, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// isSensitiveEnv checks if an environment variable is sensitive
func isSensitiveEnv(key string) bool {
	sensitiveKeys := map[string]bool{
		"PATH":           false,
		"HOME":           false,
		"USER":           false,
		"SHELL":          false,
		"TERM":           false,
		"DISPLAY":        false,
		"LANG":           false,
		"LC_ALL":         false,
		"SSH_AUTH_SOCK":  true,
		"SSH_AGENT_PID":  true,
		"GPG_AGENT_INFO": true,
		"AWS_SECRET_KEY": true,
		"AWS_ACCESS_KEY": true,
		"GITHUB_TOKEN":   true,
		"API_KEY":        true,
		"PASSWORD":       true,
		"PASSWD":         true,
		"SECRET":         true,
		"PRIVATE_KEY":    true,
	}

	sensitive, exists := sensitiveKeys[key]
	if exists {
		return sensitive
	}

	// Check for common sensitive patterns
	return containsSensitivePattern(key)
}

// containsSensitivePattern checks if a key contains sensitive patterns
func containsSensitivePattern(key string) bool {
	sensitivePatterns := []string{
		"TOKEN",
		"SECRET",
		"PASSWORD",
		"PASSWD",
		"PRIVATE",
		"KEY",
		"AUTH",
		"CREDENTIALS",
	}

	key = strings.ToUpper(key)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(key, pattern) {
			return true
		}
	}
	return false
}
