package validator

import (
	"mercator-hq/jupiter/pkg/mpl/ast"
	mplErrors "mercator-hq/jupiter/pkg/mpl/errors"
)

// Validator is the main validator that orchestrates all validation passes.
// It runs structural, semantic, and action validation in sequence.
type Validator struct {
	structural *StructuralValidator
	semantic   *SemanticValidator
	actions    *ActionValidator
}

// NewValidator creates a new validator with all validation passes.
func NewValidator() *Validator {
	return &Validator{
		structural: NewStructuralValidator(),
		semantic:   NewSemanticValidator(),
		actions:    NewActionValidator(),
	}
}

// Validate runs all validation passes on a policy.
// It accumulates errors from all passes and returns them together.
func (v *Validator) Validate(policy *ast.Policy) error {
	errors := mplErrors.NewErrorList()

	// Run structural validation
	if err := v.structural.Validate(policy); err != nil {
		if errList, ok := err.(*mplErrors.ErrorList); ok {
			errors.Errors = append(errors.Errors, errList.Errors...)
		}
	}

	// Run semantic validation (only if structural validation passed)
	// This prevents cascading errors
	if !errors.HasErrorType(mplErrors.ErrorTypeStructural) {
		if err := v.semantic.Validate(policy); err != nil {
			if errList, ok := err.(*mplErrors.ErrorList); ok {
				errors.Errors = append(errors.Errors, errList.Errors...)
			}
		}
	}

	// Run action validation (only if structural validation passed)
	if !errors.HasErrorType(mplErrors.ErrorTypeStructural) {
		if err := v.actions.Validate(policy); err != nil {
			if errList, ok := err.(*mplErrors.ErrorList); ok {
				errors.Errors = append(errors.Errors, errList.Errors...)
			}
		}
	}

	return errors.ToError()
}

// ValidateStructural runs only structural validation.
func (v *Validator) ValidateStructural(policy *ast.Policy) error {
	return v.structural.Validate(policy)
}

// ValidateSemantic runs only semantic validation.
func (v *Validator) ValidateSemantic(policy *ast.Policy) error {
	return v.semantic.Validate(policy)
}

// ValidateActions runs only action validation.
func (v *Validator) ValidateActions(policy *ast.Policy) error {
	return v.actions.Validate(policy)
}
