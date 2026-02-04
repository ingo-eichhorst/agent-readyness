# Project Research Summary

**Project:** Agent Readiness Score (ARS) - Academic Citations Implementation (v0.0.4)
**Domain:** Technical documentation enhancement with academic citations
**Researched:** 2026-02-04
**Confidence:** HIGH

## Executive Summary

The v0.0.4 milestone adds academic citations to ARS metric descriptions to establish credibility with engineering leaders who need research-backed justification for code quality investments. This is a **content expansion task, not an infrastructure build** — the existing architecture already supports citations through `citations.go`, `descriptions.go`, and HTML template patterns.

The recommended approach is manual citation curation using inline `(Author, Year)` format with per-category reference sections. The research strongly recommends against common pitfalls: no new dependencies needed, no JavaScript features, no citation generation tools, no numbered citation formats. The existing system uses a clean, CSS-only approach that is correct for technical documentation.

Key risk is balancing credibility with readability. Over-citation (academic paper style) will alienate developers, while under-citation undermines the credibility goal. The sweet spot is 2-6 citations per metric with a mix of foundational sources (McCabe 1976, Fowler 1999) and AI-era evidence (Borg et al. 2026). Critical pitfalls include broken URLs (23% average break rate), misattribution (25-54% error rate in academic papers), and citation clutter destroying readability. Mitigation strategies are straightforward: prefer DOIs over URLs, verify every claim against source text, and limit citations to research evidence sections.

## Key Findings

### Recommended Stack

The v0.0.4 milestone requires **zero new Go dependencies**. All necessary infrastructure exists in the codebase. The research validates that the current approach is architecturally sound and should be preserved.

**Core technologies (already in place):**
- `citations.go` Citation struct: Defines Category, Title, Authors, Year, URL, Description — structure is complete, no changes needed
- `descriptions.go` metric descriptions: Already contains inline `<span class="citation">` markup — extend with more citations
- `html/template` (stdlib): Per-category reference sections work correctly — no template changes needed
- CSS `.citation` class: Muted color styling for inline citations — appropriate for technical documentation
- Shields.io badge URLs: String formatting for badge generation — no SVG library needed

**What NOT to add:**
- Anthropic SDK: Being removed in v0.0.3, irrelevant to citation work
- BibTeX/CSL tooling: Over-engineering for ~100-150 citations that change rarely
- JavaScript for tooltips/popovers: Current CSS-only approach is correct and accessible
- SVG generation libraries: Shields.io URLs are simpler and more reliable
- Automated link checkers in CI: Overkill for one-time verification during addition
- Citation management tools (Zotero, Mendeley): Manual entry with quality control is more reliable

### Expected Features

The research identifies clear table stakes vs. differentiators for academic credibility in technical documentation.

**Must have (table stakes for v0.0.4):**
- Inline (Author, Year) format: Standard academic citation familiar to technical audiences, already specified in GitHub issues
- Complete References section per metric: Readers expect full citation details to locate sources, without them inline citations are placeholders
- Foundational source citations (pre-2021): Classic works (McCabe 1976, Fowler 1999, Martin 2003) establish theoretical foundations, 1-3 per metric
- AI/Agent era citations (2021+): Recent research (Borg et al. 2026, SWE-bench) validates metrics matter for AI agents specifically, 1-3 per metric
- Verified, accessible URLs: Broken links destroy trust instantly, all URLs must work at publication time
- Author attribution for claims: Every quantified claim needs a source, "36-44% agent break rate increase" must cite Borg et al. (2026)

**Should have (competitive differentiators):**
- Balanced foundational + recent blend: Papers citing "a mix of recent and classic literature" have higher quotation rates, target 40% foundational, 60% recent
- Primary source preference: Citing original research (McCabe's 1976 paper) over secondary summaries shows rigor
- DOIs for academic papers: Permanent addresses that survive URL changes when publishers reorganize
- Quantified impact statements: "Agents break 36% more often on complex code" (Borg et al., 2026) is more compelling than "complexity matters"
- Context-appropriate citation density: 2-6 citations per metric is the sweet spot, enough to establish credibility without cluttering readability
- Explicit methodology transparency: Brief notes like "based on 50+ LLM experiments" help readers assess evidence quality

**Defer (v1.0+ — anti-features for v0.0.4):**
- Exhaustive citation lists (10+ per metric): Over-citation signals insecurity, most increases slow after 15 references
- IEEE numbered citation style [1]: Obscures authorship, requires consulting reference list
- Footnote citations: Disrupts reading flow, works poorly in web documents
- Citation of every claim: Technical documentation becomes unreadable, cite only quantified claims and non-obvious assertions
- Mixing citation styles: Inconsistency signals sloppiness, normalize all to (Author, Year)
- Machine-generated citation lists: Auto-generated citations often have formatting issues, ALL CAPS titles

### Architecture Approach

The existing citation architecture is sound and requires no structural changes. The work is content expansion within established patterns.

**Current system works correctly:**

1. **Citation data layer** (`citations.go`): `Citation` struct with Category, Title, Authors, Year, URL, Description — 13 citations exist across 6 categories, expand to ~100-150 total
2. **Metric descriptions** (`descriptions.go`): 33 metrics with inline `<span class="citation">` markup — add more inline citations per metric
3. **HTML rendering** (`report.html`): Per-category reference sections with `{{range .Citations}}` — template is correct, no changes needed
4. **CSS styling** (`styles.css`): `.citation` class with muted color, normal font-style — appropriate styling, no changes needed

**Major components and responsibilities:**

1. **citations.go**: Citation data storage — expand with metric-level citations (category-level grouping recommended over per-metric structs)
2. **descriptions.go**: Metric description content with inline citations — add more `<span class="citation">(Author, Year)</span>` in Detailed field
3. **html.go**: Template data building with `buildHTMLCategories()` and `filterCitationsByCategory()` — existing functions work, possibly add `filterCitationsByMetric()` helper if needed
4. **report.html**: Citation rendering — per-category reference sections at bottom of each category, inline citations within metric tables
5. **styles.css**: Citation visual presentation — CSS-only rendering (no JavaScript), works offline, CSP-safe

**Data flow pattern:**
```
researchCitations[] → buildHTMLCategories() → HTMLCategory.Citations[] → report.html per-category sections
metricDescriptions{} → buildHTMLSubScores() → HTMLSubScore.DetailedDescription → report.html inline citations
```

**What NOT to change:**
- Do NOT add JavaScript-based features (tooltip popovers, dynamic loading) — current CSS-only approach is correct
- Do NOT use global bibliography — per-category references keep context together, easier to scan
- Do NOT use CSS counters for numbered citations — author-year format `(Borg, 2026)` is more informative than `[1]`
- Do NOT add per-metric citation structs — category-level grouping with inline citations is simpler and sufficient

### Critical Pitfalls

**1. Broken URLs and Link Rot (CRITICAL)**
- 23% of cited URLs are broken on average, rising to 50% for older articles
- URLs have a half-life of 4-14 years
- **Prevention:** Prefer DOIs over URLs (`https://doi.org/10.xxxx`), verify all URLs at submission, use stable source hierarchy (DOI > ArXiv > Publisher), include enough metadata to find sources manually
- **Phase:** Establish URL verification protocol in Phase 1 before adding any citations

**2. Citation-Reality Mismatch / Misattribution (CRITICAL)**
- 25-54% of citations in academic papers contain errors
- Only 20% of citing authors actually read the original paper they cite
- **Prevention:** Read the actual paper (not just abstracts), quote specific findings with page numbers, use hedged language ("research suggests" not "proves"), distinguish empirical findings from author opinions, cross-verify claims with original source
- **Phase:** ALL phases — every citation must be verified against source with claim-source matching in PR review

**3. Citing Retracted, Predatory, or Discredited Research (CRITICAL)**
- Over 15,000 predatory journals exist as of 2022
- AI tools recommend retracted papers without warnings
- **Prevention:** Check Retraction Watch Database, verify journal quality (Scopus/Web of Science/PubMed indexed), prefer established venues (ACM, IEEE, Springer), check citation count and context
- **Phase:** Establish source quality checklist in Phase 1, add Retraction Watch verification to citation process

**4. Citation Clutter Destroying Readability (MODERATE)**
- Over-citation makes documentation feel like academic paper rather than practical guidance
- Developers disengage when content feels like homework
- **Prevention:** Cite claims not facts (no citation for obvious facts), use consolidated citations for multiple supporting studies, target 1-3 citations per metric, place citations at end of evidence sections, distinguish primary vs. supporting citations
- **Phase:** Establish citation density guidelines (1-3 per metric) in Phase 1, review in Phase 7 for consistency

**5. Inconsistent Citation Formatting (MODERATE)**
- Multiple contributors using different academic backgrounds create visual inconsistency
- Mix of "(Author, Year)" and "[1]" and "Author (Year)" looks unprofessional
- **Prevention:** Establish one citation style (parenthetical author-year), create citation templates, use "et al." consistently for 3+ authors, standardize URL format (DOI preferred), run consistency check before merge
- **Phase:** Establish citation style guide in Phase 1, all subsequent phases follow standard

## Implications for Roadmap

Based on research, the work should be organized by **category** (C1-C7) with citation work per category. This is a 7-phase roadmap where each phase adds citations to one category's metrics.

### Phase 1: C1 Code Health Citations
**Rationale:** C1 has the most existing citations (complexity research is well-established), provides template for remaining categories, foundational complexity research (McCabe 1976) is readily available
**Delivers:** 6 metrics with complete citations (complexity_avg, complexity_max, function_length_avg, function_length_max, file_size_avg, duplication_rate)
**Establishes:** Citation style guide, URL verification protocol, Retraction Watch verification process, citation templates, source quality checklist
**Avoids:** Format inconsistency (#5), broken URLs (#1) — establishes prevention protocols for all subsequent phases
**Research flag:** Standard patterns (skip research-phase) — complexity metrics are well-documented

### Phase 2: C6 Testing Citations
**Rationale:** Well-researched domain with clear academic sources (TDD literature, coverage studies), builds on C1 citation patterns, relatively straightforward to find quality sources
**Delivers:** 5 metrics with complete citations (test_ratio, coverage, test_isolation, assertion_count, mock_usage)
**Uses:** C1 citation style guide, URL verification protocol from Phase 1
**Avoids:** Citation clutter (#4) — TDD literature is extensive, must prioritize seminal works
**Research flag:** Standard patterns (skip research-phase) — testing research is abundant

### Phase 3: C2 Semantic Explicitness Citations
**Rationale:** Type theory research readily available, logical progression after quality and testing metrics
**Delivers:** 5 metrics with complete citations (type_coverage, naming_consistency, magic_numbers, explicit_types, semantic_clarity)
**Uses:** Citation patterns from C1/C6
**Avoids:** Outdated research (#6) — distinguish timeless type theory from dated empirical findings
**Research flag:** Standard patterns (skip research-phase) — type annotation research is well-documented

### Phase 4: C3 Architecture Citations
**Rationale:** Classic software engineering references (Parnas 1972, Martin 2003), builds on established citation quality
**Delivers:** 5 metrics with complete citations (directory_depth, module_fanout, circular_dependencies, dead_exports, architecture_violations)
**Uses:** C1-C3 citation patterns
**Avoids:** Outdated research (#6) — Parnas 1972 is foundational but add modern replication studies
**Research flag:** Standard patterns (skip research-phase) — architecture patterns are well-documented

### Phase 5: C4 Documentation Citations
**Rationale:** Mix of academic and industry sources, medium complexity
**Delivers:** 7 metrics with complete citations (readme_quality, comment_ratio, api_docs, inline_docs, doc_freshness, example_coverage, architecture_docs)
**Uses:** C1-C4 citation patterns
**Avoids:** Paywalled sources (#7) — documentation research often in ACM/IEEE paywalls, find open-access versions
**Research flag:** Standard patterns (skip research-phase) — documentation research is well-documented

### Phase 6: C5 Temporal Dynamics Citations
**Rationale:** Primarily Tornhill's work (Your Code as a Crime Scene), fewer academic papers than other categories
**Delivers:** 5 metrics with complete citations (churn_rate, hotspot_count, temporal_coupling, change_frequency, age_distribution)
**Uses:** C1-C5 citation patterns
**Avoids:** Broken URLs (#1) — Tornhill's book may move, use ISBN backup reference
**Research flag:** Standard patterns (skip research-phase) — temporal coupling research is documented (Tornhill, D'Ambros)

### Phase 7: C7 Agent Evaluation Citations
**Rationale:** Nascent field, requires handling citation gaps explicitly, completes all categories
**Delivers:** 5 metrics with complete citations (task_completion, code_correctness, agent_efficiency, context_usage, tool_usage)
**Uses:** C1-C6 citation patterns
**Avoids:** Missing C7 citations (#8) — explicitly acknowledge novelty, use Borg et al. 2026 as primary evidence, cite SWE-bench and related work
**Research flag:** Needs research for novel metrics — AI agent code quality research is nascent, most work is in preprints

### Phase Ordering Rationale

- **Why this order:** Categories ordered by research availability and complexity — C1 has most existing work, C7 has least; each phase builds on citation patterns from previous phases
- **Why this grouping:** Per-category grouping keeps citations contextually relevant, matches existing HTML report structure (per-category reference sections)
- **How this avoids pitfalls:** Phase 1 establishes all prevention protocols (URL verification, Retraction Watch checks, style guide, source quality checklist) that subsequent phases inherit, reducing compound errors

### Research Flags

**Phases likely needing deeper research during planning:**
- **Phase 7 (C7 Agent Evaluation):** AI agent code quality research is nascent field, most relevant work is in preprints not peer-reviewed venues, requires citing adjacent research (LLM code generation, SWE-bench) and acknowledging gaps explicitly

**Phases with standard patterns (skip research-phase):**
- **Phase 1 (C1 Code Health):** Complexity metrics are well-documented (McCabe 1976, Borg et al. 2026), existing citations provide template
- **Phase 2 (C6 Testing):** TDD and coverage research is abundant, clear academic sources
- **Phase 3 (C2 Semantics):** Type theory research readily available
- **Phase 4 (C3 Architecture):** Classic SE references well-established (Parnas, Martin)
- **Phase 5 (C4 Documentation):** Documentation quality research well-documented
- **Phase 6 (C5 Temporal):** Tornhill's work and D'Ambros et al. cover domain

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All infrastructure exists in codebase, validated against existing citations.go and descriptions.go patterns |
| Features | HIGH | Citation practices are well-established in academic literature, clear distinction between table stakes and anti-features |
| Architecture | HIGH | Existing citation system is sound, verified against W3C Scholarly HTML and PubCSS patterns, no structural changes needed |
| Pitfalls | HIGH | Verified with academic citation error studies, library science guides, link rot research, retraction databases |

**Overall confidence:** HIGH

### Gaps to Address

The research was conclusive for infrastructure and patterns. Gaps are content-specific and will be resolved during each phase:

- **C7 citation scarcity:** AI agent code quality research is nascent, will handle by citing adjacent research (LLM code generation, SWE-bench, human factors in AI-assisted development) and explicitly acknowledging novelty in Phase 7
- **Paywalled source access:** Some research may be behind paywalls, will handle by finding open-access versions (ArXiv, author sites, Unpaywall), providing preprint links, and ensuring sufficient metadata for library lookup
- **DOI availability for older sources:** Pre-2000 papers may lack DOIs, will handle by using stable publisher URLs and including full bibliographic metadata (authors, year, title, venue, page numbers) to enable manual lookup
- **Citation verification workload:** ~100-150 citations require reading and verifying, will handle by establishing verification checklist in Phase 1 and applying consistently across all phases

## Sources

### Primary (HIGH confidence)

**Citation systems and formats:**
- [W3C Scholarly HTML](https://w3c.github.io/scholarly-html/) — Reference section structure, semantic markup for academic HTML
- [MDN `<cite>` Element](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/cite) — Semantic HTML for titles vs attributions
- [Purdue OWL - APA In-Text Citations](https://owl.purdue.edu/owl/research_and_citation/apa_style/apa_formatting_and_style_guide/in_text_citations_author_authors.html) — Standard inline citation format
- [UT Tyler Engineering Citations](https://libguides.uttyler.edu/citations/engineering) — APA-style as norm in engineering

**Link stability and DOIs:**
- [University of Maine Library: Link Rot](https://libguides.library.umaine.edu/power/day5) — 23% broken URL rate, verified link decay statistics
- [DOI Foundation](https://www.doi.org/) — Permanent identifier system, official documentation
- [Purdue OWL - DOIs vs URLs](https://owl.purdue.edu/owl/research_and_citation/conducting_research/internet_references/urls_vs_dois.html) — DOI permanence guidance

**Citation quality and errors:**
- [PMC: Citation Errors in Research](https://pmc.ncbi.nlm.nih.gov/articles/PMC10307651/) — Comprehensive study, 25-54% error rate, only 20% read original sources
- [Times Higher Education: Quarter of Citations Wrong](https://www.timeshighereducation.com/news/quarter-citations-top-journals-wrong-or-misleading) — 25% error rate in top journals
- [Retraction Watch Database](https://retractionwatch.com/) — Retraction verification database

**Citation density and best practices:**
- [Enago Academy: Citation Count](https://www.enago.com/academy/how-many-citations-do-you-need-finding-the-right-amount-of-references-for-your-research-paper/) — Quality over quantity, citation density research
- [Wikipedia: Citation Overkill](https://en.wikipedia.org/wiki/Wikipedia:Citation_overkill) — Over-citation disrupts flow

### Secondary (MEDIUM confidence)

**Academic HTML patterns:**
- [PubCSS: Formatting Academic Publications](https://thomaspark.co/2015/01/pubcss-formatting-academic-publications-in-html-css/) — HTML/CSS academic paper formatting patterns
- [Accessible Footnotes HTML](https://niquette.ca/articles/accessible-footnotes/) — ARIA attributes, CSS-only approaches

**Source quality:**
- [Enago: Predatory Journals](https://www.enago.com/academy/retractions-predatory-journals-crisis/) — 15,000+ predatory journals, journal quality issues
- [SCImago Journal Rank](https://www.scimagojr.com/) — Journal quality metrics

**Open access:**
- [Unpaywall](https://unpaywall.org/) — Free browser extension for legal open access
- [Ness Labs: Paywalled Research](https://nesslabs.com/paywalled-research-access) — Legal access strategies

### Tertiary (LOW confidence)

**Citation tools (evaluated but not recommended):**
- [Lychee Link Checker](https://github.com/lycheeverse/lychee) — Evaluated for link verification, determined overkill for ~150 one-time citations
- [W3Schools CSS Tooltip](https://www.w3schools.com/css/css_tooltip.asp) — Evaluated for citation previews, determined unnecessary complexity
- [BibGuru - CS Citation Style](https://www.bibguru.com/blog/citation-style-for-computer-science/) — Engineering citation standards overview

---
*Research completed: 2026-02-04*
*Ready for roadmap: yes*
