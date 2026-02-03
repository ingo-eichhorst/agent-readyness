# Implementation Plan Template

Use this template when presenting the implementation plan to the user for approval.

---

## Implementation Plan for Issue #{ISSUE_NUMBER}: {ISSUE_TITLE}

**Branch:** `{BRANCH_NAME}`
**Created:** {DATE}

---

### Requirements Analysis

#### From Issue Description
| # | Requirement | Priority | Testable |
|---|-------------|----------|----------|
| 1 | {requirement} | {High/Medium/Low} | {Yes/No} |
| 2 | {requirement} | {High/Medium/Low} | {Yes/No} |

#### Acceptance Criteria
- [ ] {criterion 1}
- [ ] {criterion 2}

#### Implicit Requirements
- {Any implied requirements from context}

---

### Technical Approach

#### Solution Overview
{2-3 sentence description of the approach}

#### Architecture Impact
- **Components Affected:** {list}
- **New Dependencies:** {if any}
- **Breaking Changes:** {if any}

#### Alternative Approaches Considered
1. **{Alternative 1}:** {why not chosen}
2. **{Alternative 2}:** {why not chosen}

---

### Implementation Details

#### Files to Modify
| File | Type | Changes |
|------|------|---------|
| `path/to/file.go` | Modify | {description} |
| `path/to/new_file.go` | Create | {description} |
| `path/to/file_test.go` | Modify | {description} |

#### Key Functions/Types
```go
// New/modified function signatures
func NewFunction(param Type) (Result, error)
type NewType struct { ... }
```

---

### Test Strategy (TDD Order)

#### Phase 1: Core Functionality
| Test | Purpose | Expected Outcome |
|------|---------|------------------|
| `TestX_HappyPath` | Basic success case | {expected} |
| `TestX_InvalidInput` | Error handling | Returns error |

#### Phase 2: Edge Cases
| Test | Purpose | Expected Outcome |
|------|---------|------------------|
| `TestX_EmptyInput` | Boundary condition | {expected} |
| `TestX_MaxLimit` | Upper bound | {expected} |

#### Phase 3: Integration
| Test | Purpose | Expected Outcome |
|------|---------|------------------|
| `TestX_WithY` | Component interaction | {expected} |

---

### Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| {risk description} | {Low/Med/High} | {Low/Med/High} | {mitigation strategy} |

---

### Estimated Scope

- **Files Changed:** {N}
- **Lines Added:** ~{N}
- **Lines Removed:** ~{N}
- **New Tests:** {N}

---

### Questions for Clarification

1. {Any open questions that need user input}

---

**Ready to proceed?** Reply "yes" to begin implementation or provide feedback on the plan.
