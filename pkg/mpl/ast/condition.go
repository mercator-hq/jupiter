package ast

// ConditionType represents the type of condition expression in MPL.
type ConditionType string

const (
	ConditionTypeSimple   ConditionType = "simple"   // field op value
	ConditionTypeAll      ConditionType = "all"      // AND of children
	ConditionTypeAny      ConditionType = "any"      // OR of children
	ConditionTypeNot      ConditionType = "not"      // NOT of children
	ConditionTypeFunction ConditionType = "function" // Function call
)

// Operator represents a comparison operator in MPL conditions.
type Operator string

const (
	OperatorEqual        Operator = "=="
	OperatorNotEqual     Operator = "!="
	OperatorLessThan     Operator = "<"
	OperatorGreaterThan  Operator = ">"
	OperatorLessEqual    Operator = "<="
	OperatorGreaterEqual Operator = ">="
	OperatorContains     Operator = "contains"
	OperatorMatches      Operator = "matches" // Regex match
	OperatorStartsWith   Operator = "starts_with"
	OperatorEndsWith     Operator = "ends_with"
	OperatorIn           Operator = "in"
	OperatorNotIn        Operator = "not_in"
)

// ConditionNode represents a condition expression in the AST.
// Conditions can be simple comparisons (field op value), logical operators (all/any/not),
// or function calls (has_pii(), has_injection(), etc.).
type ConditionNode struct {
	Type     ConditionType    // Type of condition
	Field    string           // Field name (for Simple conditions)
	Operator Operator         // Comparison operator (for Simple conditions)
	Value    *ValueNode       // Comparison value (for Simple conditions)
	Function string           // Function name (for Function conditions)
	Args     []*ValueNode     // Function arguments (for Function conditions)
	Children []*ConditionNode // Child conditions (for All/Any/Not)
	Location Location         // Source location
}

// IsSimple returns true if this is a simple comparison condition.
func (c *ConditionNode) IsSimple() bool {
	return c.Type == ConditionTypeSimple
}

// IsLogical returns true if this is a logical operator (all/any/not).
func (c *ConditionNode) IsLogical() bool {
	return c.Type == ConditionTypeAll || c.Type == ConditionTypeAny || c.Type == ConditionTypeNot
}

// IsFunction returns true if this is a function call condition.
func (c *ConditionNode) IsFunction() bool {
	return c.Type == ConditionTypeFunction
}
