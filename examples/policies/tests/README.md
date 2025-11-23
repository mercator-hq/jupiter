# Policy Test Examples

This directory contains example test cases for Mercator policy files. These tests demonstrate how to write and run policy unit tests.

## Test File Format

Test files use YAML format with the following structure:

```yaml
tests:
  - name: "Test case name"
    description: "What this test verifies"
    request:
      model: "model-name"
      messages:
        - role: "user"
          content: "message content"
    metadata:
      user_id: "user-123"
      # Additional metadata fields
    expect:
      action: "allow"  # or "deny", "transform", "route"
      reason: "Expected reason for deny"  # Optional
      logged: true  # Optional: verify logging occurred
```

## Running Tests

### Basic Usage

```bash
# Run tests for a specific policy
mercator test --policy examples/policies/simple-logging.yaml \
              --tests examples/policies/tests/simple-logging-tests.yaml

# Run tests with verbose output
mercator test --policy examples/policies/rate-limiting.yaml \
              --tests examples/policies/tests/rate-limiting-tests.yaml \
              --verbose
```

### Output Formats

```bash
# JSON output (useful for CI/CD)
mercator test --policy policy.yaml \
              --tests tests.yaml \
              --format json

# JUnit XML output
mercator test --policy policy.yaml \
              --tests tests.yaml \
              --format junit
```

### Coverage Reports

```bash
# Generate coverage report (shows which rules were exercised)
mercator test --policy policy.yaml \
              --tests tests.yaml \
              --coverage
```

## Available Test Files

- `simple-logging-tests.yaml` - Tests for the basic logging policy
- `rate-limiting-tests.yaml` - Tests for rate limiting and quota policies

## Writing Effective Tests

### Test Different Scenarios

Cover both positive and negative cases:

```yaml
tests:
  # Positive case - request should be allowed
  - name: "Allow valid request"
    request:
      model: "gpt-3.5-turbo"
      messages: [{"role": "user", "content": "Hello"}]
    expect:
      action: "allow"

  # Negative case - request should be blocked
  - name: "Block over-limit request"
    metadata:
      rate_limit:
        hourly: 101
    expect:
      action: "deny"
      reason: "Rate limit exceeded"
```

### Test Edge Cases

Include edge cases in your tests:

```yaml
tests:
  - name: "Handle empty metadata"
    request:
      model: "gpt-3.5-turbo"
      messages: []
    metadata: {}
    expect:
      action: "allow"

  - name: "Handle missing fields"
    request:
      model: ""
      messages: []
    expect:
      action: "deny"
```

### Use Descriptive Names

Make test names clear and specific:

```yaml
# Good
- name: "Block free tier user exceeding 100 requests per hour"

# Bad
- name: "Test 1"
```

## Test Metadata Fields

Common metadata fields used in tests:

| Field | Description | Example |
|-------|-------------|---------|
| `user.id` | User identifier | `"user-123"` |
| `user.tier` | User subscription tier | `"free"`, `"basic"`, `"pro"` |
| `rate_limit.hourly` | Current hourly request count | `50` |
| `rate_limit.daily_tokens` | Total tokens used today | `10000` |
| `rate_limit.per_second` | Requests in last second | `5` |
| `estimated_cost` | Estimated request cost | `0.05` |
| `user_budget_spent` | Amount spent by user | `25.50` |
| `user_budget_limit` | User's budget limit | `100.00` |

## CI/CD Integration

### GitHub Actions Example

```yaml
- name: Run policy tests
  run: |
    mercator test --policy policies.yaml \
                  --tests tests.yaml \
                  --format junit \
                  --output test-results.xml

- name: Upload test results
  uses: actions/upload-artifact@v3
  with:
    name: test-results
    path: test-results.xml
```

### Exit Codes

The `mercator test` command returns:
- `0` - All tests passed
- `3` - One or more tests failed

## Best Practices

1. **Test every rule** - Each policy rule should have at least one test
2. **Test boundaries** - Test values at, above, and below limits
3. **Document expectations** - Use clear descriptions for what each test verifies
4. **Keep tests isolated** - Each test should be independent
5. **Use realistic data** - Mirror production scenarios in your tests

## Further Reading

- [Policy Language Reference](../../../docs/mpl/README.md)
- [CLI Testing Guide](../../../docs/CLI-COOKBOOK.md#policy-testing)
- [Policy Best Practices](../README.md)
