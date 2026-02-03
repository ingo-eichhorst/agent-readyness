# TDD Best Practices Reference

## The Three Laws of TDD

1. **You may not write production code until you have written a failing unit test**
2. **You may not write more of a unit test than is sufficient to fail** (and not compiling is failing)
3. **You may not write more production code than is sufficient to pass the currently failing test**

## RED-GREEN-REFACTOR Cycle

### RED Phase
- Write the smallest test that fails
- Test should express intent, not implementation
- Name tests descriptively: `TestFeature_Scenario_ExpectedBehavior`
- Verify the test fails for the right reason

### GREEN Phase
- Write minimal code to make the test pass
- Don't over-engineer - just make it work
- It's okay if the code is ugly at this stage
- Focus on correctness, not elegance

### REFACTOR Phase
- Clean up both test and production code
- Remove duplication
- Improve naming
- Simplify complex logic
- ENSURE TESTS STILL PASS after each change

## Go Testing Patterns

### Table-Driven Tests
```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive numbers", 1, 2, 3},
        {"negative numbers", -1, -2, -3},
        {"mixed", -1, 2, 1},
        {"zeros", 0, 0, 0},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Add(tt.a, tt.b)
            if result != tt.expected {
                t.Errorf("Add(%d, %d) = %d; want %d", tt.a, tt.b, result, tt.expected)
            }
        })
    }
}
```

### Test Helpers
```go
func TestSomething(t *testing.T) {
    t.Helper() // Mark as helper for better error reporting
    // ...
}
```

### Subtests for Organization
```go
func TestCalculator(t *testing.T) {
    t.Run("Addition", func(t *testing.T) {
        // addition tests
    })
    t.Run("Subtraction", func(t *testing.T) {
        // subtraction tests
    })
}
```

## Test Naming Conventions

| Pattern | Example |
|---------|---------|
| `TestFunction_Condition_Expectation` | `TestParse_InvalidJSON_ReturnsError` |
| `TestType_Method_Scenario` | `TestUser_Validate_EmptyEmail` |
| `TestFeature_EdgeCase` | `TestCache_ExpirationBoundary` |

## What to Test

### DO Test
- Happy path (normal expected behavior)
- Edge cases (empty input, boundaries, limits)
- Error conditions (invalid input, failures)
- Integration points (API calls, database)

### DON'T Test
- Third-party library internals
- Private implementation details
- Trivial getters/setters
- Generated code

## Test Organization

```
package/
├── feature.go           # Production code
├── feature_test.go      # Unit tests
├── testdata/            # Test fixtures
│   ├── valid.json
│   └── invalid.json
└── integration_test.go  # Integration tests (build tag)
```

## Common Testing Mistakes

1. **Testing implementation, not behavior**
   - Bad: Test that a specific private method is called
   - Good: Test the observable outcome

2. **Too many assertions per test**
   - Bad: One test with 10 assertions
   - Good: Multiple focused tests

3. **Test interdependence**
   - Bad: Test B relies on Test A running first
   - Good: Each test is independent

4. **Ignoring test failures**
   - Bad: Skip failing tests
   - Good: Fix or remove broken tests

5. **Not testing error paths**
   - Bad: Only test happy path
   - Good: Test all error conditions
