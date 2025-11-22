package parser

import (
	"os"

	"gopkg.in/yaml.v3"
)

// yamlPolicy represents the intermediate structure for parsing YAML policies.
// It matches the YAML structure before transformation to AST.
type yamlPolicy struct {
	MPLVersion  string                 `yaml:"mpl_version"`
	Name        string                 `yaml:"name"`
	Version     string                 `yaml:"version"`
	Description string                 `yaml:"description"`
	Author      string                 `yaml:"author"`
	Created     string                 `yaml:"created"`
	Updated   string                 `yaml:"updated"`
	Tags      []string               `yaml:"tags"`
	Variables map[string]interface{} `yaml:"variables"`
	Rules     []yamlRule             `yaml:"rules"`
	Includes  []string               `yaml:"includes"`
	Tests     []yamlTest             `yaml:"tests"`
}

// yamlRule represents an intermediate rule structure.
type yamlRule struct {
	Name        string                   `yaml:"name"`
	Description string                   `yaml:"description"`
	Enabled     *bool                    `yaml:"enabled"` // Pointer to distinguish unset vs false
	Conditions  interface{}              `yaml:"conditions"`
	Actions     []map[string]interface{} `yaml:"actions"`
	Priority    int                      `yaml:"priority"`
}

// yamlTest represents an intermediate test structure.
type yamlTest struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Request     map[string]interface{} `yaml:"request"`
	Expected    yamlTestExpectation    `yaml:"expected"`
}

// yamlTestExpectation represents expected test outcomes.
type yamlTestExpectation struct {
	Action      string                 `yaml:"action"`
	RuleMatches []string               `yaml:"rule_matches"`
	Transforms  map[string]interface{} `yaml:"transforms"`
	Error       bool                   `yaml:"error"`
	ErrorMsg    string                 `yaml:"error_msg"`
}

// parseYAMLFile reads and parses a YAML file into the intermediate structure.
// It preserves line numbers from the YAML parser for error reporting.
func parseYAMLFile(path string) (*yamlPolicy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return parseYAMLBytes(data, path)
}

// parseYAMLBytes parses YAML bytes into the intermediate structure.
func parseYAMLBytes(data []byte, sourcePath string) (*yamlPolicy, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, err
	}

	var policy yamlPolicy
	if err := node.Decode(&policy); err != nil {
		return nil, err
	}

	return &policy, nil
}
