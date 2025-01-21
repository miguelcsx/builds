// internal/parsers/remarks/parser.go

package remarks

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"builds/internal/models"
)

var (
	remarkRegex   = regexp.MustCompile(`remark: ([^:]+):(\d+):(\d+): (.+?) \[([-\w]+)\]$`)
	passedRegex   = regexp.MustCompile(`'([^']+)' (inlined into) '([^']+)' with \(([^)]+)\):(.*?) at callsite ([^;]+);`)
	missedRegex   = regexp.MustCompile(`([^:]+): (.+)`)
	analysisRegex = regexp.MustCompile(`(.+): (.+)`)
)

type Parser struct {
	reader io.Reader
}

func NewParser(reader io.Reader) *Parser {
	return &Parser{
		reader: reader,
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

		remark, err := p.parseLine(line)
		if err != nil {
			continue // Skip malformed remarks
		}
		remarks = append(remarks, remark)
	}

	return remarks, scanner.Err()
}

func (p *Parser) parseLine(line string) (models.CompilerRemark, error) {
	var remark models.CompilerRemark
	remark.Timestamp = time.Now()

	matches := remarkRegex.FindStringSubmatch(line)
	if len(matches) < 6 {
		return remark, fmt.Errorf("invalid remark format")
	}

	// Basic remark info
	remark.Location = models.Location{
		File:     matches[1],
		Line:     int32(parseInt(matches[2])),
		Column:   int32(parseInt(matches[3])),
		Function: "",
	}

	message := matches[4]
	pass := matches[5]
	remark.Message = message

	// Parse pass type and remark type based on the pass string
	if strings.Contains(pass, "inline") {
		remark.Type = models.RemarkTypeOptimization
		remark.Pass = models.PassTypeInlining
		remark.Status = models.RemarkStatusPassed
		parseInlineRemark(&remark, message)
	} else if strings.Contains(pass, "missed") {
		remark.Type = models.RemarkTypeOptimization
		remark.Pass = models.PassTypeVectorization
		remark.Status = models.RemarkStatusMissed
		parseMissedRemark(&remark, message)
	} else if strings.Contains(pass, "analysis") {
		remark.Type = models.RemarkTypeAnalysis
		remark.Pass = models.PassTypeAnalysis
		remark.Status = models.RemarkStatusAnalysis
		parseAnalysisRemark(&remark, message)
	} else if strings.Contains(pass, "size-info") {
		remark.Type = models.RemarkTypeMetric
		remark.Pass = models.PassTypeSizeInfo
		remark.Status = models.RemarkStatusAnalysis
	}

	return remark, nil
}

func parseInlineRemark(remark *models.CompilerRemark, message string) {
	matches := passedRegex.FindStringSubmatch(message)
	if len(matches) < 7 {
		return
	}

	remark.Metadata = map[string]interface{}{
		"callee":   matches[1],
		"action":   matches[2],
		"caller":   matches[3],
		"params":   matches[4],
		"reason":   matches[5],
		"callsite": matches[6],
	}
}

func parseMissedRemark(remark *models.CompilerRemark, message string) {
	matches := missedRegex.FindStringSubmatch(message)
	if len(matches) < 3 {
		return
	}

	remark.Metadata = map[string]interface{}{
		"optimization": matches[1],
		"reason":       matches[2],
	}
}

func parseAnalysisRemark(remark *models.CompilerRemark, message string) {
	matches := analysisRegex.FindStringSubmatch(message)
	if len(matches) < 3 {
		return
	}

	remark.Metadata = map[string]interface{}{
		"analysis": matches[1],
		"result":   matches[2],
	}
}

func parseInt(s string) int {
	val, _ := strconv.Atoi(s)
	return val
}
