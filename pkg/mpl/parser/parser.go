package parser

import (
	"fmt"
	"os"

	"mercator-hq/jupiter/pkg/mpl/ast"
	mplErrors "mercator-hq/jupiter/pkg/mpl/errors"
)

// Parser parses MPL policy files into Abstract Syntax Trees.
// It handles YAML parsing, AST construction, and basic structural validation.
type Parser struct {
	// Configuration
	maxFileSize int64 // Maximum file size in bytes (default: 10MB)
	maxDepth    int   // Maximum condition nesting depth (default: 10)
	strictMode  bool  // Strict validation mode (warnings become errors)
}

// NewParser creates a new parser with default configuration.
func NewParser() *Parser {
	return &Parser{
		maxFileSize: 10 * 1024 * 1024, // 10MB
		maxDepth:    10,
		strictMode:  false,
	}
}

// WithMaxFileSize sets the maximum file size limit.
func (p *Parser) WithMaxFileSize(size int64) *Parser {
	p.maxFileSize = size
	return p
}

// WithMaxDepth sets the maximum condition nesting depth.
func (p *Parser) WithMaxDepth(depth int) *Parser {
	p.maxDepth = depth
	return p
}

// WithStrictMode enables strict validation (warnings become errors).
func (p *Parser) WithStrictMode(strict bool) *Parser {
	p.strictMode = strict
	return p
}

// Parse parses a policy file at the given path and returns the AST.
// It returns an error if the file cannot be read, has invalid YAML syntax,
// or contains structural errors.
func (p *Parser) Parse(path string) (*ast.Policy, error) {
	// Check file size
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, &mplErrors.Error{
			Type:    mplErrors.ErrorTypeIO,
			Message: fmt.Sprintf("Failed to access file: %v", err),
			Location: ast.Location{
				File: path,
			},
		}
	}

	if fileInfo.Size() > p.maxFileSize {
		return nil, &mplErrors.Error{
			Type:    mplErrors.ErrorTypeIO,
			Message: fmt.Sprintf("File size %d exceeds maximum %d bytes", fileInfo.Size(), p.maxFileSize),
			Location: ast.Location{
				File: path,
			},
		}
	}

	// Parse YAML
	yamlPolicy, err := parseYAMLFile(path)
	if err != nil {
		return nil, &mplErrors.Error{
			Type:    mplErrors.ErrorTypeSyntax,
			Message: fmt.Sprintf("YAML parsing failed: %v", err),
			Location: ast.Location{
				File: path,
				Line: 1,
			},
			Suggestion: "Check YAML syntax (indentation, colons, quotes)",
		}
	}

	// Build AST
	builder := newBuilder(path)
	policy, err := builder.buildPolicy(yamlPolicy)
	if err != nil {
		// Add context to errors
		if errList, ok := err.(*mplErrors.ErrorList); ok {
			for i, e := range errList.Errors {
				errList.Errors[i] = mplErrors.AddContextToError(e)
			}
		}
		return nil, err
	}

	return policy, nil
}

// ParseBytes parses policy YAML from a byte slice.
// This is useful for testing or parsing policies from memory.
func (p *Parser) ParseBytes(data []byte, sourcePath string) (*ast.Policy, error) {
	if int64(len(data)) > p.maxFileSize {
		return nil, &mplErrors.Error{
			Type:    mplErrors.ErrorTypeIO,
			Message: fmt.Sprintf("Data size %d exceeds maximum %d bytes", len(data), p.maxFileSize),
			Location: ast.Location{
				File: sourcePath,
			},
		}
	}

	// Parse YAML
	yamlPolicy, err := parseYAMLBytes(data, sourcePath)
	if err != nil {
		return nil, &mplErrors.Error{
			Type:    mplErrors.ErrorTypeSyntax,
			Message: fmt.Sprintf("YAML parsing failed: %v", err),
			Location: ast.Location{
				File:   sourcePath,
				Line:   1,
				Column: 1,
			},
			Suggestion: "Check YAML syntax (indentation, colons, quotes)",
		}
	}

	// Build AST
	builder := newBuilder(sourcePath)
	policy, err := builder.buildPolicy(yamlPolicy)
	if err != nil {
		// Add context to errors (won't work for in-memory data, but safe to call)
		if errList, ok := err.(*mplErrors.ErrorList); ok {
			for i, e := range errList.Errors {
				errList.Errors[i] = mplErrors.AddContextToError(e)
			}
		}
		return nil, err
	}

	return policy, nil
}

// ParseMulti parses multiple policy files and merges them into a single policy.
// Rules from all files are combined in order. The first file's metadata is used.
// This is used for policy composition.
func (p *Parser) ParseMulti(paths []string) (*ast.Policy, error) {
	if len(paths) == 0 {
		return nil, &mplErrors.Error{
			Type:    mplErrors.ErrorTypeIO,
			Message: "No policy files provided",
		}
	}

	// Parse first file as base policy
	policy, err := p.Parse(paths[0])
	if err != nil {
		return nil, err
	}

	// Parse and merge additional files
	for _, path := range paths[1:] {
		additionalPolicy, err := p.Parse(path)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", path, err)
		}

		// Merge variables (later files override earlier ones)
		for name, variable := range additionalPolicy.Variables {
			policy.Variables[name] = variable
		}

		// Append rules
		policy.Rules = append(policy.Rules, additionalPolicy.Rules...)
	}

	return policy, nil
}
