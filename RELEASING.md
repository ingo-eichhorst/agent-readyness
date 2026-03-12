# Releasing

Release checklist for Agent Readiness Score (ARS).

## Pre-release

1. **All tests pass**
   ```bash
   go test ./... -count=1
   ```

2. **Build succeeds**
   ```bash
   go build ./cmd/ars
   ```

3. **`go install` works locally**
   ```bash
   go install ./cmd/ars
   ars --help
   ```

4. **CHANGELOG.md updated**
   - New version section added between `[Unreleased]` and previous version
   - Comparison links updated at bottom of file

5. **No uncommitted changes**
   ```bash
   git status
   ```

## Tag and push

1. **Create annotated tag**
   ```bash
   git tag -a v0.0.X -m "Release v0.0.X"
   ```

2. **Push tag**
   ```bash
   git push origin v0.0.X
   ```

3. **Verify CI** — the `release.yml` workflow runs automatically on tag push and checks:
   - `go build ./cmd/ars`
   - `go install ./cmd/ars`
   - `go test ./...`
   - `go vet ./...`

## Post-release verification

1. **Wait for Go module proxy** (up to 5 minutes)
   ```bash
   GOPROXY=https://proxy.golang.org go list -m github.com/ingo-eichhorst/agent-readyness@v0.0.X
   ```

2. **Verify `go install` from a clean environment**
   ```bash
   cd $(mktemp -d)
   go install github.com/ingo-eichhorst/agent-readyness/cmd/ars@v0.0.X
   ars --help
   ```

3. **Check GitHub release page** — confirm the tag appears at
   https://github.com/ingo-eichhorst/agent-readyness/releases

## What can go wrong

| Symptom | Cause | Fix |
|---------|-------|-----|
| `go install ...@vX` fails with "no matching versions" | Tag pushed before `cmd/ars/main.go` existed | Tag a new release that includes `cmd/ars/` (see #73) |
| CI release workflow fails | Build or test broken at tagged commit | Fix, re-tag with next patch version |
| Module proxy serves stale version | Caching delay | Wait 5 minutes, then `GONOSUMCHECK=* go install ...` |
