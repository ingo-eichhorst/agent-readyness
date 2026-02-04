# Citation Quality Guide

This guide establishes citation standards for ARS metric descriptions. All phases (C1-C7) follow these protocols to ensure consistent, verifiable, high-quality citations.

## Table of Contents

1. [Citation Style Guide](#1-citation-style-guide)
2. [Reference Entry Format](#2-reference-entry-format)
3. [URL Verification Protocol](#3-url-verification-protocol)
4. [Source Quality Checklist](#4-source-quality-checklist)

---

## 1. Citation Style Guide

### Inline Citation Format

All citations use the `(Author, Year)` format with HTML markup for styling:

```html
<span class="citation">(Author, Year)</span>
```

### Author Formatting Rules

| Authors | Format | Example |
|---------|--------|---------|
| Single author | Last name only | `(McCabe, 1976)` |
| Two authors | Last names with "&" | `(Gao & Ore, 2017)` |
| Three or more | First author + "et al." | `(Fowler et al., 1999)` |

### Multiple Citations

Combine multiple citations with semicolons inside a single span:

```html
<span class="citation">(McCabe, 1976; Fowler et al., 1999)</span>
```

### Citation Density Target

**Target:** 2-3 citations per metric

- **Minimum:** 1 foundational (pre-2021) + 1 AI-era (2021+)
- **Maximum:** 4 citations per metric (exceptional cases only)

**Rationale:** Engineering documentation should be evidence-based but not academic. Fewer, stronger citations are better than citation clutter.

### Placement Guidelines

| Section | Citations | Example |
|---------|-----------|---------|
| Brief description | Key citation only (if quantified claim) | "...break rates by 36-44% (Borg et al., 2026)" |
| Definition | None needed | Factual definitions don't require citations |
| Why It Matters | 0-1 citations | Only if making a specific, verifiable claim |
| **Research Evidence** | **1-3 citations** | Primary citation location |
| Recommended Thresholds | Citation if from specific research | "McCabe established complexity >10 as high-risk" |
| How to Improve | **None** | Actionable guidance, not claims |

**Good Example (from complexity_avg):**

```html
<h4>Research Evidence</h4>
<p>Empirical research quantifies the impact <span class="citation">(Borg et al., 2026)</span>:</p>
<!-- data table -->
<p>McCabe's foundational work established complexity above 10 as high-risk
<span class="citation">(McCabe, 1976)</span>, and Fowler identified high-complexity
functions as primary refactoring targets <span class="citation">(Fowler et al., 1999)</span>.</p>
```

**Bad Example:**

```html
<!-- Over-cited: Too many citations interrupt readability -->
<p>Complexity is bad <span class="citation">(McCabe, 1976)</span> and
causes problems <span class="citation">(Fowler et al., 1999)</span> and
agents struggle <span class="citation">(Borg et al., 2026)</span> and
research confirms <span class="citation">(Martin, 2003)</span> that
you should refactor <span class="citation">(Gamma et al., 1994)</span>.</p>
```

### Consistency Requirements

- **Always** use `<span class="citation">` markup (never bare parentheses)
- **Never** use numbered references like `[1]` or `[McCabe]`
- **Always** use "et al." (never "and others" or "&al.")
- **Always** include the comma before year: `(Author, Year)` not `(Author Year)`

---

## 2. Reference Entry Format

### Citation Struct (citations.go)

Each reference entry in `internal/output/citations.go` follows this structure:

```go
{
    Category:    "C1",                    // "C1" through "C7"
    Title:       "A Complexity Measure",  // Full paper/book title
    Authors:     "McCabe",                // Last names, "et al." for 3+
    Year:        1976,                    // Publication year (int)
    URL:         "https://doi.org/...",   // DOI preferred
    Description: "Original cyclomatic complexity metric definition",
}
```

### Field Guidelines

**Category**
- Use "C1" through "C7" to match metric categories
- Citations appear in the References section of their category

**Authors**
- Last names only, matching inline citation format
- Single author: `"McCabe"`
- Two authors: `"Gao & Ore"`
- Three or more: `"Fowler et al."`

**Year**
- Publication year as integer
- For preprints: Use arXiv upload year (when research became available)
- For books: Use first edition year unless citing a specific later edition

**URL**
- **DOI format preferred:** `https://doi.org/10.xxxx/xxxxx`
- **ArXiv format:** `https://arxiv.org/abs/XXXX.XXXXX`
- **Direct links:** Acceptable when DOI unavailable
- **No URL shorteners:** Use full canonical URLs

**Description**
- Brief (5-15 words) summary of what the citation supports
- Focus on the specific claim, not the paper's full scope
- Example: "Original cyclomatic complexity metric definition"

### Example Entries

```go
// Foundational source with DOI
{
    Category:    "C1",
    Title:       "A Complexity Measure",
    Authors:     "McCabe",
    Year:        1976,
    URL:         "https://doi.org/10.1109/TSE.1976.233837",
    Description: "Original cyclomatic complexity metric definition",
}

// AI-era source with ArXiv
{
    Category:    "C1",
    Title:       "Code for Machines, Not Just Humans: Quantifying AI-Friendliness",
    Authors:     "Borg et al.",
    Year:        2026,
    URL:         "https://arxiv.org/abs/2601.02200",
    Description: "Empirical study showing CodeHealth metrics predict AI agent break rates",
}

// Book with publisher URL
{
    Category:    "C1",
    Title:       "Refactoring: Improving the Design of Existing Code",
    Authors:     "Fowler et al.",
    Year:        1999,
    URL:         "https://martinfowler.com/books/refactoring.html",
    Description: "Code smells and refactoring patterns for maintainability",
}
```

---

## 3. URL Verification Protocol

### When to Verify

Verify URLs during the research phase, not after implementation. Catching broken links early saves rework.

### Verification Methods

**Method 1: HTTP Status Check (Primary)**

```bash
curl -I [URL]
```

Expected: `HTTP/1.1 200 OK` or `HTTP/2 200`

Acceptable redirects:
- `301 Moved Permanently` to valid destination
- `302 Found` (common for DOI resolution)
- `303 See Other` (common for arXiv)

**Method 2: DOI Verification**

DOIs resolve through the DOI system, which handles redirects automatically.

```bash
curl -I https://doi.org/10.1109/TSE.1976.233837
```

The DOI system will redirect to the publisher. Check that:
1. Redirect chain completes (no 404)
2. Final destination shows the paper/article

**Method 3: ArXiv Verification**

ArXiv URLs follow predictable patterns:

```bash
curl -I https://arxiv.org/abs/2601.02200
```

Verify:
- Paper exists at the URL
- Title and authors match citation

**Method 4: Manual Browser Check (Final)**

After automated checks, open the URL in a browser to confirm:
- Content matches the citation
- Paper/book is accessible (or accessible with institutional login)
- Page isn't a placeholder or "coming soon"

### Handling Paywalls

Many academic papers are behind publisher paywalls. This is acceptable.

**For paywalled sources:**
1. Use DOI (provides permanent link regardless of paywall)
2. Include sufficient metadata for manual lookup:
   - Author(s)
   - Year
   - Title
   - Journal/venue name
3. Check if open-access version exists:
   - Author's personal site
   - ArXiv preprint
   - Institutional repository
4. Note in Description if paywalled: `"(via IEEE)" or "(paywalled)"`

**Example for paywalled source:**

```go
{
    Category:    "C1",
    Title:       "A Complexity Measure",
    Authors:     "McCabe",
    Year:        1976,
    URL:         "https://doi.org/10.1109/TSE.1976.233837",
    Description: "Original cyclomatic complexity definition (IEEE TSE)",
}
```

### URL Red Flags

Avoid URLs with these characteristics (high link rot risk):
- Session IDs or authentication tokens in URL
- Query parameters like `?preview=true` or `?temp=1`
- Institutional proxy prefixes
- URL shorteners (bit.ly, tinyurl)
- "preview" or "temp" in the path

### Documentation

Document verification in the research phase notes, not in code:
- "All URLs verified [date]"
- Note any paywalled sources
- Note any sources with only ArXiv (no peer-reviewed version)

---

## 4. Source Quality Checklist

### Foundational Sources (pre-2021)

These establish timeless theory and principles. Use for:
- Metric definitions (McCabe for complexity, Parnas for coupling)
- Design principles (SOLID, code smells, refactoring patterns)
- Testing theory (TDD, coverage)

**Quality Criteria:**

- [ ] Seminal work in the field (widely cited, foundational)
- [ ] Author is recognized authority (McCabe, Parnas, Fowler, Martin, Beck)
- [ ] Peer-reviewed paper OR industry-standard book
- [ ] DOI available (for papers) or stable publisher URL (for books)
- [ ] Still relevant (theory hasn't been superseded)

**Accepted source types:**
- IEEE/ACM peer-reviewed papers
- Foundational books (Refactoring, Clean Code, Design Patterns)
- Well-cited technical reports

**Current foundational sources in ARS:**

| Author | Work | Year | Domain |
|--------|------|------|--------|
| McCabe | A Complexity Measure | 1976 | Cyclomatic complexity |
| Parnas | Decomposing Systems into Modules | 1972 | Module design, coupling |
| Fowler et al. | Refactoring | 1999 | Code smells, function length |
| Martin | Agile Software Development | 2003 | Coupling metrics (Ca/Ce) |
| Gamma et al. | Design Patterns | 1994 | Cohesion, module design |
| Beck | TDD By Example | 2002 | Testing practices |

### AI-Era Sources (2021+)

These provide empirical evidence specific to AI/LLM code generation and agent behavior. Use for:
- Quantified impact on AI agents (break rates, success rates)
- LLM-specific recommendations
- Agent behavior studies

**Quality Criteria:**

- [ ] ArXiv preprint acceptable (standard in AI/ML research)
- [ ] Published in reputable venue if peer-reviewed (ICSE, FSE, NeurIPS, ICLR)
- [ ] Research methodology is sound (clear experimental setup)
- [ ] Claims are specific and verifiable
- [ ] Not retracted (check if suspicious)

**Accepted source types:**
- ArXiv preprints (standard for AI/ML)
- Peer-reviewed conference papers (ICSE, FSE, ASE)
- Journal papers (TOSEM, TSE, EMSE)
- Technical reports from reputable organizations

**Current AI-era sources in ARS:**

| Author | Work | Year | Domain |
|--------|------|------|--------|
| Borg et al. | Code for Machines, Not Just Humans | 2026 | CodeHealth impact on agents |

### Retraction Watch Check Process

**When to check:**
- Source seems suspicious (extraordinary claims, unknown venue)
- Author is unfamiliar and paper lacks established citations
- Claims seem too strong or absolute
- Found via unreliable secondary source

**How to check:**
1. Visit [retractionwatch.com](https://retractionwatch.com)
2. Search for author name + paper title
3. Check the Retraction Watch Database if available

**Documentation:**
- If checked: Note "Retraction status verified [date]" in research notes
- If not checked: No documentation needed (trust reputable sources)

**Default trust levels:**

| Source Type | Trust Level | Check Retraction? |
|-------------|-------------|-------------------|
| IEEE/ACM papers | High | Only if suspicious |
| ArXiv preprints | Medium-High | Only if suspicious |
| Major publisher books | High | No |
| Unknown venue | Medium | Yes, verify |
| Blog posts, gray literature | Low | Verify claims independently |

### Source Selection Decision Tree

```
Is the claim foundational theory (complexity, coupling, testing)?
├── YES → Use pre-2021 seminal source (McCabe, Parnas, Fowler, etc.)
│         └── Is there AI-era validation?
│             ├── YES → Include both: foundational + AI-era
│             └── NO → Foundational alone is acceptable
└── NO (AI-specific claim: break rates, LLM behavior)
    └── Must use AI-era source (2021+)
        └── Is ArXiv preprint available?
            ├── YES → Use ArXiv (standard for AI/ML)
            └── NO → Wait for research or acknowledge gap
```

### Avoiding Common Mistakes

**Mistake: Citing based on abstract only**
- Always read the relevant section of the paper
- Verify the claim matches what the paper actually says
- Use hedged language ("research suggests" not "proves")

**Mistake: Citation chaining**
- Don't cite secondary sources that cite the original
- Go to the original source
- Exception: If original is inaccessible, cite review paper and note it

**Mistake: Outdated AI claims**
- Pre-2021 research doesn't apply to modern LLMs
- GPT-2 era findings may not apply to GPT-4/Claude
- Use most recent empirical research available

**Mistake: Over-citation**
- 2-3 citations per metric is the target
- More citations doesn't mean better
- Choose the strongest, most relevant sources

---

## Appendix: Existing Citation Examples

### Good: complexity_avg (from descriptions.go)

```html
<h4>Research Evidence</h4>
<p>Empirical research quantifies the impact of code complexity on AI agent
performance <span class="citation">(Borg et al., 2026)</span>:</p>
<!-- Evidence table with specific numbers -->
<p>The study identifies a maximum nesting depth threshold of <strong>4 levels</strong>
...McCabe's foundational work established complexity above 10 as high-risk
<span class="citation">(McCabe, 1976)</span>, and Fowler identified high-complexity
functions as primary refactoring targets <span class="citation">(Fowler et al., 1999)</span>.</p>
```

**Why it's good:**
- AI-era source (Borg) provides quantified impact (36-44% break rates)
- Foundational sources (McCabe, Fowler) establish theoretical basis
- Citations in Research Evidence section, not scattered
- Specific claims tied to specific sources

### Good: Citation Entry (from citations.go)

```go
{
    Category:    "C1",
    Title:       "Code for Machines, Not Just Humans: Quantifying AI-Friendliness",
    Authors:     "Borg et al.",
    Year:        2026,
    URL:         "https://arxiv.org/abs/2601.02200",
    Description: "Empirical study showing CodeHealth metrics predict AI agent break rates",
}
```

**Why it's good:**
- Consistent author format with inline citations
- ArXiv URL (stable, accessible)
- Description focuses on specific claim being supported

---

*Guide version: 1.0*
*Established: Phase 18 (C1 Code Health)*
*Applies to: All metric descriptions (C1-C7)*
