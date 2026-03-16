# Benchmark Scoring Calibration: Boundary Repo Findings

**Date:** 2026-03-16
**Context:** Added 6 "boundary" repos (3 expected very high ≥8.5, 3 expected very low ≤4) to stress-test the scoring range. Results revealed both selection errors and structural limitations in the scoring model.

---

## Results vs Expectations

| Repo | Lang | Expected | Actual | Delta |
|------|------|----------|--------|-------|
| `grpc/grpc-go` v1.63.0 | Go | 8–10 | **6.95** | −1.5 |
| `fogleman/gg` (main) | Go | 1–4 | **7.42** | +4.4 |
| `psf/requests` v2.32.3 | Python | 8–10 | **7.32** | −1.2 |
| `chardet/chardet` 5.2.0 | Python | 1–4 | **6.09** | +2.6 |
| `pmndrs/zustand` v4.5.5 | TypeScript | 8–10 | **7.60** | −0.9 |
| `Polymer/prpl-server` v3.1.0 | TypeScript | 1–4 | **7.11** | +3.6 |

None of the six repos fell outside the ~6–8 band that dominates the existing benchmark.

---

## Finding 1: Repo selection errors

The "very low" selections conflated **stagnant/archived** with **poorly written**:

- **`fogleman/gg`** (7.42): Clean, minimal 2D graphics library. Low duplication, readable code, straightforward structure. C1=8.5, C2=7.9. Stagnation doesn't degrade structural quality — a stable, finished library can be well-written.
- **`prpl-server`** (7.11): Small, professionally written Node.js/TypeScript server from the Polymer team. C3=9.4, C4=7.9. Being archived doesn't imply bad architecture.
- **`chardet`** (6.09): Lowest of the group as expected, but still "Agent-Assisted". The legacy ported code (from Java) is algorithmically complex but structurally coherent. C6=2.9 (near-zero tests) correctly scored low but other categories compensated.

**Lesson:** "Small + finished + no new features" ≠ low quality. Truly low-scoring repos need deliberately bad structure: monolithic files, heavy duplication, random naming, no tests, and no documentation. These are hard to find in well-known open-source projects since they get abandoned before accumulating community exposure.

---

## Finding 2: C2 (Semantics) systematically underscores large codebases

| Repo | C2 Score | Observation |
|------|----------|-------------|
| `grpc-go` | 4.52 | Large, complex Go — no type annotations |
| `requests` | 3.94 | Python without PEP 484 type hints |
| `zustand` | 4.28 | TypeScript, but C2 still mid-range |
| `gg` | 7.90 | Small Go library — clear naming, no complexity |
| `chardet` | 5.76 | Small Python — simple naming |

C2 appears to penalise **scale and complexity** rather than purely measuring naming quality. A well-named large project scores worse than a poorly-named small one. This is a calibration gap: C2 should distinguish naming quality from codebase size.

---

## Finding 3: C5 (Temporal) doesn't penalise stagnation as expected

- `fogleman/gg` last updated ~2021 → C5 = **8.2**
- `prpl-server` archived 2018 → C5 = **7.05**
- `requests` actively maintained → C5 = **8.92**
- `chardet` stagnant → C5 = **8.2**

C5 scores for stagnant repos are indistinguishable from actively maintained ones. The metric may be measuring **commit density over a short recent window** or interpreting low churn as stability rather than abandonment. A repo that hasn't changed in 3 years gets the same signal as one releasing monthly.

---

## Finding 4: Effective scoring range is ~5–8, not 1–10

Across all 36 repos now in the benchmark (30 original + 6 boundary):

- **Highest score:** ~8.5 (color)
- **Lowest score:** ~4.1 (ms, chardet)
- **Clustering:** ~80% of repos score between 6.0 and 8.0

The composite formula normalises individual category extremes into a band that lacks discrimination at both ends. With C7 (LLM) disabled, the maximum achievable composite is structurally capped below 9.

---

## Recommendations

### Short term
1. **Replace boundary repos** with repos that have genuinely pathological properties:
   - Low end: Large monorepo with >50% duplication, single 10k-line files, zero test files, no README sections, no function documentation
   - High end: Small focused library (<5k LOC) with >90% test coverage, full JSDoc/godoc, strict TypeScript, active releases
2. **Update `expected_score_range`** in benchmark.yaml to reflect the realistic 5–8 band rather than aspirational 1–10 ranges.

### Medium term
3. **C2 recalibration:** Decouple naming quality from codebase size. Large, well-named projects should not be penalised relative to small ones.
4. **C5 recalibration:** Introduce explicit stagnation detection (e.g. last commit > 18 months old = penalty, archived flag = hard penalty).
5. **C6 floor:** Near-zero test coverage should bottom out closer to 0–1, not 2.9. The current floor compresses the low end.
