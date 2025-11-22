package engine

import (
	"sort"

	"mercator-hq/jupiter/pkg/mpl/ast"
)

// PolicyPriority defines default priorities for different policy types.
const (
	// PriorityHigh is for security and compliance policies (blocking)
	PriorityHigh = 100

	// PriorityMedium is for routing and transformation policies
	PriorityMedium = 50

	// PriorityLow is for monitoring and tagging policies
	PriorityLow = 10

	// PriorityDefault is the default priority when not specified
	PriorityDefault = PriorityMedium
)

// GetPolicyPriority returns the effective priority for a policy.
// It uses explicit priority from metadata if available, otherwise uses default based on policy type.
func GetPolicyPriority(policy *ast.Policy) int {
	// Check if policy has explicit priority in tags
	for _, tag := range policy.Tags {
		switch tag {
		case "security", "compliance", "blocking":
			return PriorityHigh
		case "routing", "transformation":
			return PriorityMedium
		case "monitoring", "tagging", "analytics":
			return PriorityLow
		}
	}

	// Analyze rules to determine default priority
	hasBlockingRule := false
	hasRoutingRule := false

	for _, rule := range policy.Rules {
		if !rule.IsEnabled() {
			continue
		}

		for _, action := range rule.Actions {
			switch action.Type {
			case ast.ActionTypeDeny, ast.ActionTypeRateLimit, ast.ActionTypeBudget:
				hasBlockingRule = true
			case ast.ActionTypeRoute:
				hasRoutingRule = true
			}
		}
	}

	// Assign priority based on action types
	if hasBlockingRule {
		return PriorityHigh
	}
	if hasRoutingRule {
		return PriorityMedium
	}

	return PriorityLow
}

// GetRulePriority returns the effective priority for a rule.
// Uses explicit priority if set, otherwise defaults based on action types.
func GetRulePriority(rule *ast.Rule) int {
	// Use explicit priority if set (non-zero)
	if rule.Priority != 0 {
		return rule.Priority
	}

	// Determine priority based on action types
	hasBlockingAction := false
	hasRoutingAction := false

	for _, action := range rule.Actions {
		switch action.Type {
		case ast.ActionTypeDeny, ast.ActionTypeRateLimit, ast.ActionTypeBudget:
			hasBlockingAction = true
		case ast.ActionTypeRoute:
			hasRoutingAction = true
		}
	}

	if hasBlockingAction {
		return PriorityHigh
	}
	if hasRoutingAction {
		return PriorityMedium
	}

	return PriorityLow
}

// SortPoliciesByPriority sorts policies by priority (highest first).
func SortPoliciesByPriority(policies []*ast.Policy) {
	sort.Slice(policies, func(i, j int) bool {
		priorityI := GetPolicyPriority(policies[i])
		priorityJ := GetPolicyPriority(policies[j])

		// Higher priority comes first
		if priorityI != priorityJ {
			return priorityI > priorityJ
		}

		// If priorities are equal, sort by name for deterministic ordering
		return policies[i].Name < policies[j].Name
	})
}

// SortRulesByPriority sorts rules by priority (highest first).
func SortRulesByPriority(rules []*ast.Rule) {
	sort.Slice(rules, func(i, j int) bool {
		priorityI := GetRulePriority(rules[i])
		priorityJ := GetRulePriority(rules[j])

		// Higher priority comes first
		if priorityI != priorityJ {
			return priorityI > priorityJ
		}

		// If priorities are equal, sort by name for deterministic ordering
		return rules[i].Name < rules[j].Name
	})
}

// NormalizePolicyPriorities normalizes all policy and rule priorities.
// This should be called after loading policies to ensure consistent priority assignment.
func NormalizePolicyPriorities(policies []*ast.Policy) {
	for _, policy := range policies {
		// Sort rules within each policy by priority
		if len(policy.Rules) > 1 {
			SortRulesByPriority(policy.Rules)
		}
	}

	// Sort policies by priority
	if len(policies) > 1 {
		SortPoliciesByPriority(policies)
	}
}
