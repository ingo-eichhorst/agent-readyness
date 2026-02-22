# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

@agent.md

## Benchmark

The benchmark suite scores 30 pinned open-source repos (10 Go, 10 Python, 10 TypeScript) and detects scoring regressions via golden files.

**Run (read-only regression check):**
```bash
go test -tags benchmark -run TestBenchmarkRepos -timeout 45m ./benchmark/ -v
```

**Regenerate golden files** (after intentional scoring changes):
```bash
rm -rf benchmark/repos/
go test -tags benchmark -run TestBenchmarkRepos -timeout 45m ./benchmark/ -v -update
```

**Generate HTML report** from existing golden files:
```bash
go run ./benchmark/report   # writes benchmark/report.html
```

**Key files:**
- `benchmark/benchmark.yaml` — repo list with pinned commits
- `benchmark/golden/` — checked-in golden files (one JSON per repo)
- `benchmark/report/main.go` — report generator
- `benchmark/repos/` — cloned at runtime, gitignored

**Notes:**
- Build tag `benchmark` keeps this out of `go test ./...`
- Repos are shallow-cloned (`--depth=500`) for C5 temporal analysis
- C5 uses HEAD commit date as time reference, not wall-clock time (stable for pinned commits)
- C7 requires LLM and always scores -1 in the benchmark
