package engine

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"mercator-hq/jupiter/pkg/mpl/ast"
)

// evaluateOperator evaluates an operator comparison between actual and expected values.
func evaluateOperator(op ast.Operator, actual, expected interface{}) (bool, error) {
	switch op {
	case ast.OperatorEqual:
		return evaluateEqual(actual, expected)

	case ast.OperatorNotEqual:
		equal, err := evaluateEqual(actual, expected)
		return !equal, err

	case ast.OperatorLessThan:
		return evaluateLessThan(actual, expected)

	case ast.OperatorGreaterThan:
		return evaluateGreaterThan(actual, expected)

	case ast.OperatorLessEqual:
		return evaluateLessEqual(actual, expected)

	case ast.OperatorGreaterEqual:
		return evaluateGreaterEqual(actual, expected)

	case ast.OperatorContains:
		return evaluateContains(actual, expected)

	case ast.OperatorMatches:
		return evaluateMatches(actual, expected)

	case ast.OperatorStartsWith:
		return evaluateStartsWith(actual, expected)

	case ast.OperatorEndsWith:
		return evaluateEndsWith(actual, expected)

	case ast.OperatorIn:
		return evaluateIn(actual, expected)

	case ast.OperatorNotIn:
		in, err := evaluateIn(actual, expected)
		return !in, err

	default:
		return false, fmt.Errorf("unknown operator: %q", op)
	}
}

// evaluateEqual checks if two values are equal.
func evaluateEqual(actual, expected interface{}) (bool, error) {
	// Handle nil cases
	if actual == nil && expected == nil {
		return true, nil
	}
	if actual == nil || expected == nil {
		return false, nil
	}

	// Try numeric comparison first (handles int vs float64)
	actualNum, actualErr := convertToFloat64(actual)
	expectedNum, expectedErr := convertToFloat64(expected)
	if actualErr == nil && expectedErr == nil {
		return actualNum == expectedNum, nil
	}

	// Use reflection for deep comparison for non-numeric types
	return reflect.DeepEqual(actual, expected), nil
}

// evaluateLessThan checks if actual < expected (numeric comparison).
func evaluateLessThan(actual, expected interface{}) (bool, error) {
	actualNum, expectedNum, err := toNumeric(actual, expected)
	if err != nil {
		return false, err
	}

	return actualNum < expectedNum, nil
}

// evaluateGreaterThan checks if actual > expected (numeric comparison).
func evaluateGreaterThan(actual, expected interface{}) (bool, error) {
	actualNum, expectedNum, err := toNumeric(actual, expected)
	if err != nil {
		return false, err
	}

	return actualNum > expectedNum, nil
}

// evaluateLessEqual checks if actual <= expected (numeric comparison).
func evaluateLessEqual(actual, expected interface{}) (bool, error) {
	actualNum, expectedNum, err := toNumeric(actual, expected)
	if err != nil {
		return false, err
	}

	return actualNum <= expectedNum, nil
}

// evaluateGreaterEqual checks if actual >= expected (numeric comparison).
func evaluateGreaterEqual(actual, expected interface{}) (bool, error) {
	actualNum, expectedNum, err := toNumeric(actual, expected)
	if err != nil {
		return false, err
	}

	return actualNum >= expectedNum, nil
}

// evaluateContains checks if actual contains expected (substring or element).
func evaluateContains(actual, expected interface{}) (bool, error) {
	// Convert to strings for substring matching
	actualStr, ok := toString(actual)
	if !ok {
		// Try slice/array contains
		return containsElement(actual, expected)
	}

	expectedStr, ok := toString(expected)
	if !ok {
		return false, fmt.Errorf("contains operator requires string or convertible value for expected")
	}

	return strings.Contains(actualStr, expectedStr), nil
}

// evaluateMatches checks if actual matches the expected regex pattern.
func evaluateMatches(actual, expected interface{}) (bool, error) {
	// Convert actual to string
	actualStr, ok := toString(actual)
	if !ok {
		return false, fmt.Errorf("matches operator requires string or convertible value for actual")
	}

	// Get regex pattern
	pattern, ok := expected.(string)
	if !ok {
		return false, fmt.Errorf("matches operator requires string pattern for expected")
	}

	// Compile and match regex
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
	}

	return re.MatchString(actualStr), nil
}

// evaluateStartsWith checks if actual starts with expected.
func evaluateStartsWith(actual, expected interface{}) (bool, error) {
	actualStr, ok := toString(actual)
	if !ok {
		return false, fmt.Errorf("starts_with operator requires string or convertible value for actual")
	}

	expectedStr, ok := toString(expected)
	if !ok {
		return false, fmt.Errorf("starts_with operator requires string or convertible value for expected")
	}

	return strings.HasPrefix(actualStr, expectedStr), nil
}

// evaluateEndsWith checks if actual ends with expected.
func evaluateEndsWith(actual, expected interface{}) (bool, error) {
	actualStr, ok := toString(actual)
	if !ok {
		return false, fmt.Errorf("ends_with operator requires string or convertible value for actual")
	}

	expectedStr, ok := toString(expected)
	if !ok {
		return false, fmt.Errorf("ends_with operator requires string or convertible value for expected")
	}

	return strings.HasSuffix(actualStr, expectedStr), nil
}

// evaluateIn checks if actual is in the expected list.
func evaluateIn(actual, expected interface{}) (bool, error) {
	// Expected should be a slice or array
	expectedVal := reflect.ValueOf(expected)
	if expectedVal.Kind() != reflect.Slice && expectedVal.Kind() != reflect.Array {
		return false, fmt.Errorf("in operator requires slice or array for expected, got %s", expectedVal.Kind())
	}

	// Check if actual is in the slice
	for i := 0; i < expectedVal.Len(); i++ {
		elem := expectedVal.Index(i).Interface()
		if reflect.DeepEqual(actual, elem) {
			return true, nil
		}
	}

	return false, nil
}

// containsElement checks if a slice/array contains an element.
func containsElement(slice, elem interface{}) (bool, error) {
	sliceVal := reflect.ValueOf(slice)
	if sliceVal.Kind() != reflect.Slice && sliceVal.Kind() != reflect.Array {
		return false, fmt.Errorf("contains operator on non-string requires slice or array, got %s", sliceVal.Kind())
	}

	for i := 0; i < sliceVal.Len(); i++ {
		if reflect.DeepEqual(sliceVal.Index(i).Interface(), elem) {
			return true, nil
		}
	}

	return false, nil
}

// toNumeric converts values to float64 for numeric comparison.
func toNumeric(actual, expected interface{}) (float64, float64, error) {
	actualNum, err := convertToFloat64(actual)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot convert actual value to number: %w", err)
	}

	expectedNum, err := convertToFloat64(expected)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot convert expected value to number: %w", err)
	}

	return actualNum, expectedNum, nil
}

// convertToFloat64 converts a value to float64.
func convertToFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int8:
		return float64(val), nil
	case int16:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case uint:
		return float64(val), nil
	case uint8:
		return float64(val), nil
	case uint16:
		return float64(val), nil
	case uint32:
		return float64(val), nil
	case uint64:
		return float64(val), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

// toString converts a value to string.
func toString(v interface{}) (string, bool) {
	switch val := v.(type) {
	case string:
		return val, true
	case fmt.Stringer:
		return val.String(), true
	default:
		// Try fmt.Sprint as fallback
		return fmt.Sprint(v), true
	}
}
