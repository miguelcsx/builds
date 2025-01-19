package kernel

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"

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
			continue // Skip malformed remarks
		}

		remarks = append(remarks, remark)
	}

	// Add accumulated metrics as separate remarks
	for metric, value := range p.metrics {
		remarks = append(remarks, models.CompilerRemark{
			Type:    "metric",
			Message: metric,
			Args: []models.RemarkArg{
				{
					String: strconv.Itoa(value),
				},
			},
		})
	}

	return remarks, scanner.Err()
}

func (p *Parser) parseLine(line string) (models.CompilerRemark, error) {
	var remark models.CompilerRemark

	// Parse location and function info
	locMatches := locationRegex.FindStringSubmatch(line)
	if len(locMatches) >= 5 {
		remark.Location = models.Location{
			File:     locMatches[1],
			Line:     int32(parseInt(locMatches[2])),
			Column:   int32(parseInt(locMatches[3])),
			Function: strings.Trim(locMatches[4], "'"),
		}
		p.currentFunc = remark.Location.Function
		line = line[len(locMatches[0]):]
	}

	// Extract the remaining message
	line = strings.TrimSpace(line)
	remark.Message = line

	// Parse different types of remarks
	if metricMatches := metricsRegex.FindStringSubmatch(line); metricMatches != nil {
		remark.Type = "metric"
		remark.Message = metricMatches[1]
		p.metrics[metricMatches[1]] = parseInt(metricMatches[2])
		remark.Args = []models.RemarkArg{
			{
				String: metricMatches[2],
			},
		}
	} else if callMatches := callRegex.FindStringSubmatch(line); callMatches != nil {
		remark.Type = "function_call"
		remark.Args = []models.RemarkArg{
			{
				Callee: callMatches[1],
			},
		}
	} else if memMatches := memoryRegex.FindStringSubmatch(line); memMatches != nil {
		remark.Type = "memory_access"
		remark.Args = []models.RemarkArg{
			{
				String: memMatches[1], // instruction
			},
		}
		if memMatches[2] != "" {
			remark.Args = append(remark.Args, models.RemarkArg{
				String: memMatches[2], // value
			})
		}
		remark.Args = append(remark.Args, models.RemarkArg{
			Reason: memMatches[3], // access type
		})
	} else {
		// Default to info type for unrecognized remarks
		remark.Type = "info"
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
