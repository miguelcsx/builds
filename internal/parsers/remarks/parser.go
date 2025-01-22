// internal/parsers/remarks/parser.go

package remarks

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"builds/internal/models"

	"gopkg.in/yaml.v3"
)

type Parser struct {
	filepath string
}

type YamlRemark struct {
	Pass     string        `yaml:"Pass"`
	Name     string        `yaml:"Name"`
	Function string        `yaml:"Function"`
	DebugLoc *YamlLocation `yaml:"DebugLoc,omitempty"`
	Args     []YamlArg     `yaml:"Args,omitempty"`
	Hotness  int32         `yaml:"Hotness,omitempty"`
}

type YamlLocation struct {
	File     string `yaml:"File"`
	Line     int32  `yaml:"Line"`
	Column   int32  `yaml:"Column"`
	Function string `yaml:"Function,omitempty"`
	Region   string `yaml:"Region,omitempty"`
}

type YamlArg struct {
	String      string        `yaml:"String,omitempty"`
	Callee      string        `yaml:"Callee,omitempty"`
	Caller      string        `yaml:"Caller,omitempty"`
	Type        string        `yaml:"Type,omitempty"`
	Line        string        `yaml:"Line,omitempty"`
	Column      string        `yaml:"Column,omitempty"`
	DebugLoc    *YamlLocation `yaml:"DebugLoc,omitempty"`
	OtherAccess *struct {
		Type     string        `yaml:"type,omitempty"`
		DebugLoc *YamlLocation `yaml:"DebugLoc,omitempty"`
	} `yaml:"OtherAccess,omitempty"`
	ClobberedBy *struct {
		Type     string        `yaml:"type,omitempty"`
		DebugLoc *YamlLocation `yaml:"DebugLoc,omitempty"`
	} `yaml:"ClobberedBy,omitempty"`
}

func NewParser(filepath string) *Parser {
	return &Parser{filepath: filepath}
}

func (p *Parser) Parse() ([]models.CompilerRemark, error) {
	data, err := os.ReadFile(p.filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var remarks []models.CompilerRemark

	for {
		var node yaml.Node
		err := decoder.Decode(&node)
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		if node.Kind != yaml.DocumentNode || len(node.Content) == 0 {
			continue
		}

		root := node.Content[0]
		var yamlRemark YamlRemark
		if err := root.Decode(&yamlRemark); err != nil {
			continue
		}

		// Extract the type from the tag (e.g., "!Passed" -> "Passed")
		remarkType := strings.TrimPrefix(root.Tag, "!")
		if remarkType == "" {
			// Skip if no type tag found
			continue
		}

		remark := models.CompilerRemark{
			Type:      strings.ToLower(remarkType), // Convert to lowercase for consistency
			Pass:      yamlRemark.Pass,
			Message:   p.buildMessage(yamlRemark),
			Function:  yamlRemark.Function,
			Timestamp: time.Now(),
			Hotness:   yamlRemark.Hotness,
		}

		// Set status based on type
		switch remark.Type {
		case "passed":
			remark.Status = "passed"
		case "missed":
			remark.Status = "missed"
		case "analysis":
			remark.Status = "analysis"
		default:
			remark.Status = "info"
		}

		// Convert location
		if yamlRemark.DebugLoc != nil {
			remark.Location = models.Location{
				File:     yamlRemark.DebugLoc.File,
				Line:     yamlRemark.DebugLoc.Line,
				Column:   yamlRemark.DebugLoc.Column,
				Function: yamlRemark.DebugLoc.Function,
				Region:   yamlRemark.DebugLoc.Region,
			}
		}

		// Process arguments
		if len(yamlRemark.Args) > 0 {
			remark.Args = models.RemarkArgs{
				Strings: make([]string, 0),
				Values:  make(map[string]string),
			}

			for _, arg := range yamlRemark.Args {
				if arg.String != "" {
					remark.Args.Strings = append(remark.Args.Strings, arg.String)
				}
				if arg.Callee != "" {
					remark.Args.Callee = arg.Callee
				}
				if arg.Caller != "" {
					remark.Args.Caller = arg.Caller
				}
				if arg.Type != "" {
					remark.Args.Type = arg.Type
				}
				if arg.Line != "" {
					remark.Args.Line = arg.Line
				}
				if arg.Column != "" {
					remark.Args.Column = arg.Column
				}
				if arg.DebugLoc != nil {
					remark.Args.DebugLoc = &models.Location{
						File:   arg.DebugLoc.File,
						Line:   arg.DebugLoc.Line,
						Column: arg.DebugLoc.Column,
					}
				}

				// Handle OtherAccess and ClobberedBy
				if arg.OtherAccess != nil {
					remark.Args.OtherAccess = &models.RemarkAccess{
						Type: arg.OtherAccess.Type,
					}
					if arg.OtherAccess.DebugLoc != nil {
						remark.Args.OtherAccess.DebugLoc = &models.Location{
							File:   arg.OtherAccess.DebugLoc.File,
							Line:   arg.OtherAccess.DebugLoc.Line,
							Column: arg.OtherAccess.DebugLoc.Column,
						}
					}
				}

				if arg.ClobberedBy != nil {
					remark.Args.ClobberedBy = &models.RemarkAccess{
						Type: arg.ClobberedBy.Type,
					}
					if arg.ClobberedBy.DebugLoc != nil {
						remark.Args.ClobberedBy.DebugLoc = &models.Location{
							File:   arg.ClobberedBy.DebugLoc.File,
							Line:   arg.ClobberedBy.DebugLoc.Line,
							Column: arg.ClobberedBy.DebugLoc.Column,
						}
					}
				}
			}
		}

		remarks = append(remarks, remark)
	}

	return remarks, nil
}

func (p *Parser) buildMessage(remark YamlRemark) string {
	var parts []string

	// Start with pass and name
	parts = append(parts, fmt.Sprintf("%s: %s", remark.Pass, remark.Name))

	// Add arguments
	for _, arg := range remark.Args {
		if arg.String != "" {
			parts = append(parts, arg.String)
		}
		if arg.Callee != "" && arg.Caller != "" {
			parts = append(parts, fmt.Sprintf("%s -> %s", arg.Callee, arg.Caller))
		}
		if arg.Type != "" {
			parts = append(parts, fmt.Sprintf("type: %s", arg.Type))
		}
	}

	return strings.Join(parts, " ")
}
