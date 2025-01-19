package remarks

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"builds/internal/models"
)

var (
	// Patterns for parsing different types of remarks
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

func (p *Parser) Parse() ([]models.LLVMRemark, error) {
	var remarks []models.LLVMRemark
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

func (p *Parser) parseLine(line string) (models.LLVMRemark, error) {
	var remark models.LLVMRemark
	matches := remarkRegex.FindStringSubmatch(line)
	if len(matches) < 6 {
		return remark, fmt.Errorf("invalid remark format")
	}

	// Basic remark info
	remark.DebugLoc = models.Location{
		File:   matches[1],
		Line:   parseInt(matches[2]),
		Column: parseInt(matches[3]),
	}

	message := matches[4]
	pass := matches[5]
	remark.Pass = strings.TrimPrefix(pass, "Rpass")
	remark.Pass = strings.TrimPrefix(remark.Pass, "Rpass-missed")
	remark.Pass = strings.TrimPrefix(remark.Pass, "Rpass-analysis")

	// Parse different remark types
	if strings.Contains(pass, "inline") {
		remark.Type = "!Passed"
		remark.Args = p.parseInlineRemark(message)
	} else if strings.Contains(pass, "missed") {
		remark.Type = "!Missed"
		remark.Args = p.parseMissedRemark(message)
	} else {
		remark.Type = "!Analysis"
		remark.Args = p.parseAnalysisRemark(message)
	}

	return remark, nil
}

func (p *Parser) parseInlineRemark(message string) []models.Args {
	matches := passedRegex.FindStringSubmatch(message)
	if len(matches) < 7 {
		return nil
	}

	return []models.Args{
		{Callee: matches[1]},
		{String: matches[2]},
		{Function: matches[3]},
		{String: matches[4]},
		{Reason: matches[5]},
		{String: "at callsite " + matches[6]},
	}
}

func (p *Parser) parseMissedRemark(message string) []models.Args {
	matches := missedRegex.FindStringSubmatch(message)
	if len(matches) < 3 {
		return nil
	}

	return []models.Args{
		{String: matches[1]},
		{String: matches[2]},
	}
}

func (p *Parser) parseAnalysisRemark(message string) []models.Args {
	matches := analysisRegex.FindStringSubmatch(message)
	if len(matches) < 3 {
		return nil
	}

	return []models.Args{
		{String: matches[1]},
		{String: matches[2]},
	}
}

func parseInt(s string) int {
	val, _ := strconv.Atoi(s)
	return val
}
