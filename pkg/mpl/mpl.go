package mpl

import (
	"mercator-hq/jupiter/pkg/mpl/ast"
	"mercator-hq/jupiter/pkg/mpl/parser"
	"mercator-hq/jupiter/pkg/mpl/validator"
)

// ParseAndValidate is a convenience function that parses and validates a policy file.
// It returns the parsed AST if successful, or an error if parsing or validation fails.
func ParseAndValidate(path string) (*ast.Policy, error) {
	// Parse
	p := parser.NewParser()
	policy, err := p.Parse(path)
	if err != nil {
		return nil, err
	}

	// Validate
	v := validator.NewValidator()
	if err := v.Validate(policy); err != nil {
		return nil, err
	}

	return policy, nil
}

// ParseAndValidateBytes is a convenience function that parses and validates policy YAML from bytes.
func ParseAndValidateBytes(data []byte, sourcePath string) (*ast.Policy, error) {
	// Parse
	p := parser.NewParser()
	policy, err := p.ParseBytes(data, sourcePath)
	if err != nil {
		return nil, err
	}

	// Validate
	v := validator.NewValidator()
	if err := v.Validate(policy); err != nil {
		return nil, err
	}

	return policy, nil
}

// Parse parses a policy file without validation.
// Use this if you want to inspect the AST before validation.
func Parse(path string) (*ast.Policy, error) {
	p := parser.NewParser()
	return p.Parse(path)
}

// Validate validates a parsed policy.
func Validate(policy *ast.Policy) error {
	v := validator.NewValidator()
	return v.Validate(policy)
}
