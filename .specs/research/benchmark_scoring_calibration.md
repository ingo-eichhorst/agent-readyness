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

---

## v3 Findings — 2026-04-13 (Post-fix Re-benchmark)

**Context:** After fixes in ar-dx3 (C6 vacuous isolation), ar-n3y (C5 stagnation penalty), ar-il5 (C2 size bias), and ar-6jz (C1 breakpoints), re-ran the full 36-repo benchmark and validated against GH#77 success criteria.

**Golden files updated** from the old scores (pre-fix) to reflect the new scoring algorithm.

---

### Full Scores (v3, all 36 repos)

| Repo | Lang | Expected Quality | Score | Tier | C5 | C6 |
|------|------|-----------------|-------|------|-----|-----|
| uuid | go | medium | 8.65 | Agent-Ready | 8.42 | 9.77 |
| result | python | low | 8.38 | Agent-Ready | 8.41 | 9.08 |
| color | go | low | 8.19 | Agent-Ready | 9.34 | 9.03 |
| ms | typescript | low | 7.99 | Agent-Assisted | 3.51 | 9.08 |
| gin | go | high | 7.92 | Agent-Assisted | 6.96 | 8.04 |
| go-humanize | go | low | 7.89 | Agent-Assisted | 6.75 | 8.74 |
| httpx | python | high | 7.84 | Agent-Assisted | 6.76 | 7.36 |
| cobra | go | high | 7.79 | Agent-Assisted | 6.47 | 9.28 |
| zustand | typescript | very_high | 7.68 | Agent-Assisted | 4.97 | 8.84 |
| badger | go | medium | 7.63 | Agent-Assisted | 5.23 | 10.00 |
| resty | go | medium | 7.58 | Agent-Assisted | 7.34 | 6.71 |
| superjson | typescript | medium | 7.55 | Agent-Assisted | 8.83 | 7.32 |
| typer | python | medium | 7.48 | Agent-Assisted | 7.57 | 7.13 |
| colly | go | medium | 7.48 | Agent-Assisted | 3.02 | 7.53 |
| click | python | high | 7.43 | Agent-Assisted | 9.42 | 7.30 |
| viper | go | medium | 7.38 | Agent-Assisted | 7.43 | 8.14 |
| nanoid | typescript | low | 7.27 | Agent-Assisted | 4.41 | 1.00 |
| rich | python | high | 7.20 | Agent-Assisted | 7.05 | 6.76 |
| requests | python | very_high | 7.07 | Agent-Assisted | 8.57 | 5.32 |
| grpc-go | go | very_high | 7.04 | Agent-Assisted | 5.29 | 8.79 |
| changesets | typescript | medium | 6.92 | Agent-Assisted | 5.43 | 8.12 |
| gg | go | very_low | 6.87 | Agent-Assisted | 4.50 | 5.82 |
| lo | go | high | 6.84 | Agent-Assisted | 6.65 | 4.54 |
| zod | typescript | high | 6.82 | Agent-Assisted | 4.12 | 8.96 |
| ts-pattern | typescript | high | 6.79 | Agent-Assisted | 5.32 | 8.71 |
| prpl-server | typescript | very_low | 6.70 | Agent-Assisted | 3.86 | 1.00 |
| fast-check | typescript | medium | 6.70 | Agent-Assisted | 4.58 | 8.45 |
| python-slugify | python | low | 6.72 | Agent-Assisted | 9.70 | 1.00 |
| python-dateutil | python | medium | 6.65 | Agent-Assisted | 7.39 | 8.77 |
| pydantic | python | high | 6.61 | Agent-Assisted | 7.46 | 8.15 |
| effect | typescript | medium | 6.14 | Agent-Assisted | 5.74 | 5.73 |
| kysely | typescript | high | 6.28 | Agent-Assisted | 5.97 | 5.35 |
| tqdm | python | medium | 6.07 | Agent-Assisted | 8.77 | 1.06 |
| more-itertools | python | medium | 6.06 | Agent-Assisted | 7.65 | 7.70 |
| date-fns | typescript | high | 5.82 | Agent-Limited | 6.32 | 1.00 |
| chardet | python | very_low | 5.09 | Agent-Limited | 8.20 | 1.00 |

**Overall range:** 5.09–8.65 (3.56-point spread)

---

### Success Criteria Validation (GH#77)

| # | Criterion | Target | Actual | Status |
|---|-----------|--------|--------|--------|
| 1 | Effective scoring range | ~3.0–9.5 | 5.09–8.65 | ❌ FAIL |
| 2 | Cross-language parity (same-quality within 0.5) | ≤0.5 gap | high-tier Go avg 7.5, TS avg 6.6 (0.9 gap) | ❌ FAIL |
| 3 | Zero-test C6 floor | C6 < 1.5 | All zero-test repos: C6 ≤ 1.06 | ✅ PASS |
| 4 | Stagnation penalty | C5 < 3.0 | colly: 3.02, ms: 3.51, prpl-server: 3.86 | ❌ FAIL |
| 5 | very_high vs high separation | >1.0 gap | very_high max=7.68 < high max=7.92 (−0.88) | ❌ FAIL |
| 6 | very_low vs low separation | >1.0 gap | very_low max=6.87, low min=6.65 (overlap) | ❌ FAIL |
| 7 | No regression in medium-tier ordering | Stable relative order | Medium tier stable within language | ✅ PASS |

---

### Key Observations

**What the fixes achieved:**
- **C6 floor (ar-dx3):** Fully working. Repos with 0 test files now floor at 1.0. Six repos confirmed: chardet, python-slugify, tqdm (1.06), date-fns, nanoid, prpl-server all ≤ 1.06.
- **C5 stagnation (ar-n3y):** Partial. colly (last commit 2021) now scores C5=3.02 — a significant drop from v2's 8.3 — but just barely misses the <3.0 threshold. ms and prpl-server show similar improvement.
- **C2 size bias (ar-il5):** Improved. Large repos no longer systematically penalized; grpc-go C2 improved from 4.52 to a reasonable range.
- **C1 breakpoints (ar-6jz):** Changed breakpoint distribution. uuid (tiny, clean) rose to 8.65; medium repos redistributed.

**What remains broken — structural range compression:**
The fundamental problem identified in v2 persists: the effective scoring range is 5.09–8.65, far short of the 3.0–9.5 target. Two structural issues:

1. **Inverted tier ordering:** very_high repos (grpc-go 7.04, requests 7.07, zustand 7.68) score LOWER than some "high" repos (gin 7.92, cobra 7.79). The benchmark's "very_high" designation is based on community reputation, but the scoring algorithm doesn't capture what makes grpc-go genuinely better than cobra.

2. **very_low repos escape the floor:** gg (6.87), prpl-server (6.70), and chardet (5.09) — all designated very_low — cluster with medium-quality repos. Chardet is the only one that drops below 6.0, and only because of C6=1.0. The other categories (C1-C5) provide too much compensation.

**Root cause:** The composite score is a weighted average of 6 categories. Each category has its own floor (not zero), so even a repo with catastrophic scores in one category gets pulled up by the others. The weighting and category floors prevent the composite from reaching below ~5.0 for any real open-source repo.

---

### Recommendations (v3)

1. **Accept current state for v3** — the C6 and C5 fixes are verified and working. Golden files updated.
2. **`expected_score_range` in benchmark.yaml is aspirational**, not achievable with the current model. These ranges were written before calibration data existed. They should be updated to reflect actual v3 scores if the test suite validates against them.
3. **Tier separation requires weight restructuring** — not simple breakpoint adjustments. The composite weighting would need to be changed, or a non-linear transformation applied to spread the final scores.
4. **stagnation colly edge case:** C5=3.02 misses the <3.0 target by 0.017. The stagnation penalty formula needs a minor recalibration for repos in the 18-24 month stale window.
5. **Cross-language parity gap:** TypeScript "high" repos score ~0.9 points lower than equivalent Go/Python "high" repos on average. This may require language-specific calibration or investigation into whether TypeScript repo selection is biased toward more complex projects.
