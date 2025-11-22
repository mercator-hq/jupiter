package ast

// ValueType represents the type of a value in an MPL policy.
// MPL has a strong type system with no automatic coercion.
type ValueType string

const (
	ValueTypeString   ValueType = "string"
	ValueTypeNumber   ValueType = "number"
	ValueTypeBoolean  ValueType = "boolean"
	ValueTypeArray    ValueType = "array"
	ValueTypeObject   ValueType = "object"
	ValueTypeVariable ValueType = "variable" // Reference to a variable
	ValueTypeNull     ValueType = "null"
)

// ValueNode represents a value in the AST (used in conditions, actions, variables).
// Values can be literals (string, number, boolean) or references to variables.
type ValueNode struct {
	Type         ValueType   // Type of the value
	Value        interface{} // Actual value (nil for null, string for variable reference)
	VariableName string      // Name of variable if Type is Variable
	Location     Location    // Source location
}

// IsLiteral returns true if this value is a literal (not a variable reference).
func (v *ValueNode) IsLiteral() bool {
	return v.Type != ValueTypeVariable
}

// IsVariable returns true if this value is a variable reference.
func (v *ValueNode) IsVariable() bool {
	return v.Type == ValueTypeVariable
}

// String returns a string representation of the value.
func (v *ValueNode) String() string {
	if v.Type == ValueTypeVariable {
		return "{{ variables." + v.VariableName + " }}"
	}
	if v.Type == ValueTypeNull {
		return "null"
	}
	return v.Value.(string) // Simplified for now
}
