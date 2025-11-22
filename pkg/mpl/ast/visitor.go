package ast

// Visitor provides an interface for traversing the AST.
// Implement this interface to perform operations on AST nodes
// (validation, transformation, analysis, etc.).
type Visitor interface {
	VisitPolicy(*Policy) error
	VisitRule(*Rule) error
	VisitCondition(*ConditionNode) error
	VisitAction(*Action) error
	VisitValue(*ValueNode) error
	VisitVariable(*Variable) error
}

// Walk traverses the AST starting from the policy node and calls the visitor
// for each node. It returns the first error encountered, or nil if traversal completes.
func Walk(policy *Policy, visitor Visitor) error {
	if err := visitor.VisitPolicy(policy); err != nil {
		return err
	}

	// Visit variables
	for _, variable := range policy.Variables {
		if err := visitor.VisitVariable(variable); err != nil {
			return err
		}
		if err := visitor.VisitValue(variable.Value); err != nil {
			return err
		}
	}

	// Visit rules
	for _, rule := range policy.Rules {
		if err := visitor.VisitRule(rule); err != nil {
			return err
		}

		// Visit conditions
		if rule.Conditions != nil {
			if err := walkCondition(rule.Conditions, visitor); err != nil {
				return err
			}
		}

		// Visit actions
		for _, action := range rule.Actions {
			if err := visitor.VisitAction(action); err != nil {
				return err
			}

			// Visit action parameters
			for _, param := range action.Parameters {
				if err := visitor.VisitValue(param); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// walkCondition recursively walks a condition tree.
func walkCondition(cond *ConditionNode, visitor Visitor) error {
	if err := visitor.VisitCondition(cond); err != nil {
		return err
	}

	// Visit condition value (for simple conditions)
	if cond.Value != nil {
		if err := visitor.VisitValue(cond.Value); err != nil {
			return err
		}
	}

	// Visit function arguments (for function conditions)
	for _, arg := range cond.Args {
		if err := visitor.VisitValue(arg); err != nil {
			return err
		}
	}

	// Visit child conditions (for logical operators)
	for _, child := range cond.Children {
		if err := walkCondition(child, visitor); err != nil {
			return err
		}
	}

	return nil
}
