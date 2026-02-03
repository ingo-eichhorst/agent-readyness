# GitHub Issue Workflow Checklist

Use this checklist to track progress through the workflow phases.

---

## Phase 1: Environment Setup
- [ ] Verified working directory is clean (`git status`)
- [ ] Checked out main branch
- [ ] Pulled latest changes from origin
- [ ] Parsed issue identifier from arguments
- [ ] Fetched issue details with `gh issue view`
- [ ] Created feature branch with proper naming

## Phase 2: Research & Understanding
- [ ] Parsed requirements from issue body
- [ ] Identified acceptance criteria
- [ ] Researched best practices for the problem domain
- [ ] Explored affected codebase areas
- [ ] Identified related files and patterns
- [ ] Documented potential risks and edge cases

## Phase 3: Implementation Plan
- [ ] Created structured plan document
- [ ] Listed all requirements from issue
- [ ] Described technical approach
- [ ] Identified files to modify/create
- [ ] Outlined test strategy
- [ ] Documented risks and mitigations
- [ ] **USER APPROVED THE PLAN**

## Phase 4: TDD Implementation
For each requirement:
- [ ] **RED:** Wrote failing test
- [ ] **RED:** Verified test fails for right reason
- [ ] **GREEN:** Implemented minimum code to pass
- [ ] **GREEN:** All tests passing
- [ ] **REFACTOR:** Cleaned up code
- [ ] **REFACTOR:** Tests still passing

Overall:
- [ ] All requirements have corresponding tests
- [ ] All tests are passing
- [ ] Code follows project conventions (see CLAUDE.md)

## Phase 5: Code Review
- [ ] Ran go-code-reviewer agent
- [ ] Addressed all Critical issues
- [ ] Addressed all Important issues
- [ ] Evaluated and addressed Suggestions
- [ ] Re-ran tests after fixes
- [ ] Re-ran reviewer if significant changes made

## Phase 6: QA Validation
- [ ] Ran qa-requirements-validator agent
- [ ] All requirements marked as FULFILLED
- [ ] Edge cases verified
- [ ] If gaps found: returned to Phase 4, fixed, re-validated

## Phase 7: Commit & PR
- [ ] Final test run passed
- [ ] Staged specific files (not `git add -A`)
- [ ] Created descriptive commit message
- [ ] Referenced issue number in commit
- [ ] Pushed branch to origin
- [ ] Created pull request with `gh pr create`
- [ ] PR includes summary, changes, test plan
- [ ] Reported PR URL to user

---

## Quality Gates Summary

| Gate | Criteria | Status |
|------|----------|--------|
| Plan Approval | User said "yes" | [ ] |
| Tests Pass | `go test ./...` exits 0 | [ ] |
| Code Review | No Critical/Important issues | [ ] |
| QA Validation | All requirements FULFILLED | [ ] |
| PR Created | PR URL obtained | [ ] |

---

## Troubleshooting Quick Reference

### Git Issues
- Dirty working directory: Stash or commit first
- Merge conflicts: Report to user, don't auto-resolve
- Push rejected: Pull and rebase, or report to user

### Test Issues
- Compilation error: Check syntax and imports
- Wrong failure: Review test logic
- Flaky test: Check for race conditions or timing

### API Issues
- `gh` auth error: Run `gh auth login`
- Rate limited: Wait and retry
- Network error: Check connectivity
