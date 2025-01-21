// internal/parsers/kernel/parser.go

package kernel

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"builds/internal/models"
)

var (
	locationRegex = regexp.MustCompile(`([^:]+):(\d+):(\d+): in (artificial function '[^']+')`)
	metricsRegex  = regexp.MustCompile(`([a-zA-Z_]+) = (\d+)`)
	callRegex     = regexp.MustCompile(`direct call, callee is '([^']+)'`)
	memoryRegex   = regexp.MustCompile(`'([^']+)' (?:instruction|call) \('?([^']*)'?\) accesses memory in ([a-zA-Z]+) address space`)
	functionRegex = regexp.MustCompile(`function '([^']+)'`)
)

type Parser struct {
	reader      io.Reader
	currentFunc string
	metrics     map[string]int
}

func NewParser(reader io.Reader) *Parser {
	return &Parser{
		reader:  reader,
		metrics: make(map[string]int),
	}
}

func (p *Parser) Parse() ([]models.CompilerRemark, error) {
	var remarks []models.CompilerRemark
	scanner := bufio.NewScanner(p.reader)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "remark: ") {
			continue
		}

		line = strings.TrimPrefix(line, "remark: ")
		remark, err := p.parseLine(line)
		if err != nil {
			continue
		}

		remarks = append(remarks, remark)
	}

	// Add accumulated metrics as separate remarks
	for metric, value := range p.metrics {
		remarks = append(remarks, models.CompilerRemark{
			Type:    models.RemarkTypeMetric,
			Pass:    models.PassTypeKernelInfo,
			Message: metric,
			Metadata: map[string]interface{}{
				"value": value,
			},
			Timestamp: time.Now(),
		})
	}

	return remarks, scanner.Err()
}

func (p *Parser) parseLine(line string) (models.CompilerRemark, error) {
	var remark models.CompilerRemark
	remark.Timestamp = time.Now()
	remark.Type = models.RemarkTypeKernel
	remark.Pass = models.PassTypeKernelInfo
	remark.Status = models.RemarkStatusAnalysis

	// Parse location and function info
	locMatches := locationRegex.FindStringSubmatch(line)
	if len(locMatches) >= 5 {
		remark.Location = models.Location{
			File:     locMatches[1],
			Line:     int32(parseInt(locMatches[2])),
			Column:   int32(parseInt(locMatches[3])),
			Function: strings.Trim(locMatches[4], "'"),
			Artifact: true,
		}
		p.currentFunc = remark.Location.Function
		line = line[len(locMatches[0]):]
	}

	// Extract the remaining message
	line = strings.TrimSpace(line)
	remark.Message = line

	// Initialize kernel info if not present
	if remark.KernelInfo == nil {
		remark.KernelInfo = &models.KernelInfo{
			Metrics:    make(map[string]int64),
			Attributes: make(map[string]string),
			Callees:    make([]string, 0),
		}
	}

	// Parse different types of remarks
	if metricMatches := metricsRegex.FindStringSubmatch(line); metricMatches != nil {
		metricName := metricMatches[1]
		value := parseInt(metricMatches[2])
		remark.KernelInfo.Metrics[metricName] = int64(value)

		switch metricName {
		case "DirectCalls":
			remark.KernelInfo.DirectCalls = int32(value)
		case "IndirectCalls":
			remark.KernelInfo.IndirectCalls = int32(value)
		case "FlatAddressSpaceAccesses":
			remark.KernelInfo.FlatAddressSpaceAccesses = int32(value)
		case "AllocasCount":
			remark.KernelInfo.AllocasCount = int32(value)
		case "AllocasStaticSize":
			remark.KernelInfo.AllocasStaticSize = int64(value)
		}

	} else if callMatches := callRegex.FindStringSubmatch(line); callMatches != nil {
		remark.KernelInfo.DirectCalls++
		remark.KernelInfo.Callees = append(remark.KernelInfo.Callees, callMatches[1])

	} else if memMatches := memoryRegex.FindStringSubmatch(line); memMatches != nil {
		remark.KernelInfo.MemoryAccesses = append(remark.KernelInfo.MemoryAccesses, models.MemoryAccess{
			Type:         memMatches[1],
			Instruction:  memMatches[2],
			AddressSpace: memMatches[3],
		})
		remark.KernelInfo.FlatAddressSpaceAccesses++
	}

	return remark, nil
}

func (p *Parser) GetMetrics() map[string]int {
	return p.metrics
}

func parseInt(s string) int {
	val, _ := strconv.Atoi(s)
	return val
}
