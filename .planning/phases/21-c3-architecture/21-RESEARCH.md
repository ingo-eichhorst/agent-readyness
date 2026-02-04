# Phase 21: C3 Architecture - Research

**Researched:** 2026-02-04
**Domain:** Academic citations for C3 Architecture metrics in technical documentation
**Confidence:** HIGH

## Summary

This phase adds research-backed citations to all 5 C3 Architecture metrics following the quality protocols established in Phase 18. The 5 C3 metrics (max_dir_depth, module_fanout_avg, circular_deps, import_complexity_avg, dead_exports) currently have minimal citations (Parnas 1972, Gamma et al. 1994, Martin 2003) that need significant enhancement with both foundational software architecture research and AI-era empirical evidence.

Architecture metrics represent a mature research domain with strong foundational sources from the 1970s-1990s. Parnas (1972) remains the seminal work on module decomposition and information hiding. Stevens, Myers & Constantine (1974) formalized coupling and cohesion concepts that directly map to C3 metrics. Chidamber & Kemerer (1994) extended these to object-oriented systems with the CBO (Coupling Between Objects) metric. For circular dependencies, Martin's Acyclic Dependencies Principle (2003) is widely cited, though it represents practitioner guidance rather than peer-reviewed research. Lakos (1996) provides practical techniques for eliminating cyclic dependencies in large systems.

AI-era research for architecture metrics is less abundant than for C1/C6, but recent empirical studies provide valuable evidence. Oyetoyan et al. (2015) empirically studied circular dependencies and change-proneness. MacCormack et al. (2006) validated modularity benefits through design structure matrix analysis. Pisch et al. (2024) introduced M-score as an empirically-validated modularity metric. For dead exports, Romano et al. (2018) conducted a multi-study investigation establishing that dead code harms comprehensibility and maintainability. Borg et al. (2026) provides the primary AI-era evidence linking code health metrics to AI agent reliability.

**Primary recommendation:** Use Parnas (1972) and Stevens/Myers/Constantine (1974) as foundational sources for modularity and coupling; Chidamber & Kemerer (1994) for OO coupling metrics; Martin (2003) for practitioner principles (labeled as such); Borg et al. (2026) for AI-era relevance. Handle Martin's work by noting it represents influential practitioner perspective rather than peer-reviewed research.

## Standard Stack

This phase requires **zero new Go dependencies**. All infrastructure exists from Phase 18.

### Core (Existing Infrastructure)

| Component | File | Purpose | Status |
|-----------|------|---------|--------|
| Citation struct | `internal/output/citations.go` | Stores Category, Title, Authors, Year, URL, Description | **Use as-is** |
| researchCitations | `internal/output/citations.go` | Array of Citation entries | **Expand** with C3 entries |
| metricDescriptions | `internal/output/descriptions.go` | Metric descriptions with inline HTML | **Expand** citations in Detailed field |
| CSS .citation class | `templates/styles.css` | Muted styling for inline citations | **Use as-is** |
| HTML template | `templates/report.html` | Per-category References sections | **Use as-is** |

### Supporting (Documentation Only)

| Tool | Purpose | When to Use |
|------|---------|-------------|
| `curl -I [URL]` | Verify URL accessibility | During citation addition |
| Browser verification | Confirm content matches citation | Final validation |
| DOI.org resolver | Permanent academic links | Prefer for academic papers |

**Installation:** No installation needed. All infrastructure exists.

## Architecture Patterns

### Existing Citation Architecture (Preserve)

The patterns established in Phase 18 apply directly:

```
internal/output/
├── citations.go          # Citation{} struct, researchCitations[]
├── descriptions.go       # metricDescriptions{} with inline citations
├── html.go              # buildHTMLCategories(), filterCitationsByCategory()
└── templates/
    ├── report.html      # Per-category References sections
    └── styles.css       # .citation class styling
```

### Pattern 1: Inline Citation Markup (Same as C1)

**What:** Citations appear inline as `(Author, Year)` within metric descriptions
**When to use:** In the `Detailed` field of `MetricDescription` structs
**Example:**
```go
// Source: internal/output/descriptions.go (existing pattern)
Detailed: `...
<p>Research shows modules with high coupling are harder to maintain <span class="citation">(Stevens et al., 1974)</span>.</p>
...`,
```

### Pattern 2: Research Evidence Subsection (Same as C1)

**What:** Dedicated "Research Evidence" subsection in detailed descriptions
**When to use:** All metrics with quantified claims
**Example:**
```go
Detailed: `<h4>Definition</h4>
<p>[Factual definition - no citations needed]</p>

<h4>Why It Matters for AI Agents</h4>
<p>[Explanation - 0-1 citations if specific claim]</p>

<h4>Research Evidence</h4>
<p>[2-4 citations here - primary evidence location]</p>

<h4>Recommended Thresholds</h4>
<ul><li>[Thresholds - cite if from specific research]</li></ul>

<h4>How to Improve</h4>
<ul><li>[Actionable guidance - no citations needed]</li></ul>`,
```

### Pattern 3: Distinguishing Research vs Practitioner Guidance

**What:** Clearly label practitioner opinions vs peer-reviewed research
**When to use:** C3 citations span both categories (Martin = practitioner, Parnas = research)
**Guidelines:**
- **Peer-reviewed research (cite normally):** Parnas (1972), Stevens et al. (1974), Chidamber & Kemerer (1994)
- **Practitioner perspective (label explicitly):** Martin (2003) - "influential practitioner perspective"
- **Empirical studies (strong evidence):** Oyetoyan et al. (2015), MacCormack et al. (2006), Romano et al. (2018)
- **AI-era (directly applicable):** Borg et al. (2026), Pisch et al. (2024)

### Anti-Patterns to Avoid

- **Over-citation:** Do NOT add 5+ citations per metric. Target 2-4 focused citations.
- **Citation in "How to Improve":** Actionable guidance needs no citations.
- **Presenting Martin as peer-reviewed:** Label as influential practitioner perspective.
- **Ignoring foundational sources:** Always include 1970s foundations (Parnas, Stevens et al.) for architecture metrics.

## Don't Hand-Roll

Problems with existing solutions that should NOT be rebuilt:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Citation storage | New database/JSON files | Existing `citations.go` slice | Simple, already works |
| URL verification | CI pipeline with Lychee | Manual `curl -I` checks | One-time task, overkill to automate |
| Citation formatting | BibTeX parser | Manual `(Author, Year)` strings | ~15 C3 citations don't justify tooling |

**Key insight:** This is a content expansion task, not infrastructure build. Phase 18 established all needed infrastructure.

## Common Pitfalls

### Pitfall 1: Treating Martin's Principles as Peer-Reviewed Research

**What goes wrong:** Citing Martin (2003) for specific thresholds or empirical claims.
**Why it happens:** Martin's work is influential and widely referenced in industry.
**How to avoid:**
- Label Martin's work as "influential practitioner perspective"
- Use for principle statements (e.g., Acyclic Dependencies Principle)
- Do NOT cite for quantified claims without empirical backing
- Pair with actual empirical research when possible
**Warning signs:** Specific numbers attributed to Martin, treating Clean Architecture as empirical study.

### Pitfall 2: Missing Foundational Architecture Research

**What goes wrong:** Citing only recent sources without foundational papers.
**Why it happens:** Tendency to prefer newer research; older papers less accessible.
**How to avoid:**
- Always include Parnas (1972) for module decomposition concepts
- Include Stevens et al. (1974) for coupling/cohesion foundations
- Reference Chidamber & Kemerer (1994) for OO metric definitions
- These foundations are timeless and widely validated
**Warning signs:** C3 metrics lacking pre-2000 citations, no mention of information hiding.

### Pitfall 3: Circular Dependencies Lack of Empirical Evidence

**What goes wrong:** Claiming specific defect rates for circular dependencies without research backing.
**Why it happens:** The principle is widely accepted, but empirical studies are limited.
**How to avoid:**
- Cite Oyetoyan et al. (2015) for empirical change-proneness evidence
- Cite Lakos (1996) for practical techniques (industry experience)
- Note that Martin's ADP is principle-based, not empirically derived
- Use hedged language: "correlates with" not "causes"
**Warning signs:** Specific percentages claimed for cycle impact without citation.

### Pitfall 4: Conflating Coupling Metrics

**What goes wrong:** Using afferent/efferent coupling terminology inconsistently with fanout.
**Why it happens:** Multiple overlapping metric definitions exist (CBO, Ca, Ce, fanout).
**How to avoid:**
- Be precise: module_fanout_avg relates to efferent coupling (Ce) / outgoing dependencies
- CBO (Chidamber & Kemerer) counts bidirectional coupling
- Ca (afferent) = incoming dependencies, Ce (efferent) = outgoing dependencies
- Use consistent terminology within each metric description
**Warning signs:** Mixing Ca/Ce with CBO without distinction.

### Pitfall 5: Dead Code Research Gaps

**What goes wrong:** Claiming strong empirical evidence for dead export impact when research is limited.
**Why it happens:** Dead code research is less developed than other SE areas.
**How to avoid:**
- Cite Romano et al. (2018) for the most comprehensive dead code study
- Note that research shows dead code harms comprehensibility
- Acknowledge that dead exports specifically have less direct research
- Reference Fowler (1999) for code smell classification
**Warning signs:** Overstating dead export impact with thin empirical backing.

## C3 Metrics: Required Citations

### max_dir_depth

| Type | Source | Year | DOI/URL | Status | Key Finding |
|------|--------|------|---------|--------|-------------|
| Foundational | Parnas, "Decomposing Systems into Modules" | 1972 | 10.1145/361598.361623 | Verified | Module decomposition and information hiding principles |
| Empirical | MacCormack et al., "Exploring the Structure of Complex Software Designs" | 2006 | 10.1287/mnsc.1060.0552 | Verified | Design structure matrix analysis shows modularity benefits |
| AI-Era | Borg et al., "Code for Machines" | 2026 | arxiv.org/abs/2601.02200 | Verified | Code health (including structure) predicts agent reliability |

### module_fanout_avg

| Type | Source | Year | DOI/URL | Status | Key Finding |
|------|--------|------|---------|--------|-------------|
| Foundational | Stevens, Myers & Constantine, "Structured Design" | 1974 | 10.1147/sj.132.0115 | Verified | Original coupling/cohesion definitions; low coupling improves quality |
| Foundational | Chidamber & Kemerer, "Metrics Suite for OO Design" | 1994 | 10.1109/32.295895 | Verified | CBO metric: excessive coupling detrimental to modular design |
| Practitioner | Martin, "Agile Software Development" | 2003 | Pearson ISBN | Verified | Stable Dependencies Principle (influential practitioner perspective) |
| AI-Era | Borg et al., "Code for Machines" | 2026 | arxiv.org/abs/2601.02200 | Verified | Highly-coupled code increases agent break rates |

### circular_deps

| Type | Source | Year | DOI/URL | Status | Key Finding |
|------|--------|------|---------|--------|-------------|
| Foundational | Parnas, "Decomposing Systems into Modules" | 1972 | 10.1145/361598.361623 | Verified | Modular systems should have clear dependency direction |
| Practitioner | Martin, "Agile Software Development" | 2003 | Pearson ISBN | Verified | Acyclic Dependencies Principle (influential practitioner perspective) |
| Practitioner | Lakos, "Large-Scale C++ Software Design" | 1996 | Pearson ISBN | Verified | Techniques for eliminating cyclic dependencies; acyclic designs easier to test |
| Empirical | Oyetoyan et al., "Circular Dependencies and Change-Proneness" | 2015 | 10.1109/SANER.2015.7081834 | Verified | Cycles impact change frequency of classes near cycles |
| AI-Era | Borg et al., "Code for Machines" | 2026 | arxiv.org/abs/2601.02200 | Verified | Architectural complexity impacts agent reliability |

### import_complexity_avg

| Type | Source | Year | DOI/URL | Status | Key Finding |
|------|--------|------|---------|--------|-------------|
| Foundational | Parnas, "Decomposing Systems into Modules" | 1972 | 10.1145/361598.361623 | Verified | Clear module boundaries improve comprehension |
| Empirical | Sangal et al., "Using Dependency Models to Manage Architecture" | 2005 | 10.1145/1094855.1094915 | Verified | DSM approach for managing complex dependencies; simpler is better |
| Empirical | Pisch et al., "M-score: Empirically Derived Modularity Metric" | 2024 | 10.1145/3674805.3686697 | Verified | Dependency density predicts maintainability |
| AI-Era | Borg et al., "Code for Machines" | 2026 | arxiv.org/abs/2601.02200 | Verified | Structural complexity impacts agent comprehension |

### dead_exports

| Type | Source | Year | DOI/URL | Status | Key Finding |
|------|--------|------|---------|--------|-------------|
| Foundational | Fowler et al., "Refactoring" | 1999 | martinfowler.com/books/refactoring.html | Verified | Dead Code as code smell; unused code increases cognitive load |
| Empirical | Romano et al., "Multi-Study Investigation into Dead Code" | 2018 | 10.1109/TSE.2018.2842781 | Verified | Dead code harms comprehensibility; developers should avoid this smell |
| Empirical | Malavolta et al., "JavaScript Dead Code" | 2023 | arxiv.org/abs/2308.16729 | Verified | Dead code removal improves performance and reduces resource usage |
| AI-Era | Borg et al., "Code for Machines" | 2026 | arxiv.org/abs/2601.02200 | Verified | Clean, well-organized code improves agent reliability |

## Code Examples

### Citation Addition to descriptions.go (max_dir_depth)

```go
// Source: internal/output/descriptions.go
// Pattern for adding citations to max_dir_depth

"max_dir_depth": {
    Brief:     "Deepest directory nesting level. Clear module boundaries and shallow hierarchies improve comprehensibility <span class=\"citation\">(Parnas, 1972)</span>.",
    Threshold: 6.0,
    Detailed: `<h4>Definition</h4>
<p>The maximum depth of directory nesting in the source tree, counting from the project root. Measures how deeply files are organized into subdirectories (e.g., src/api/v2/handlers/auth/utils.go = depth 6).</p>

<h4>Why It Matters for AI Agents</h4>
<p>Deep directory hierarchies make it harder for agents to locate related code and understand project organization. Long import paths consume context space and are prone to errors. Shallower structures provide clearer boundaries and easier navigation.</p>

<h4>Research Evidence</h4>
<p>Parnas's foundational work on module decomposition established that well-structured systems with clear boundaries are fundamentally easier to understand and maintain <span class="citation">(Parnas, 1972)</span>. This principle directly applies to directory organization: shallow, well-named hierarchies communicate structure more effectively than deep nesting.</p>
<p>Empirical studies using design structure matrices confirm that modular architectures with clear boundaries have measurable quality benefits <span class="citation">(MacCormack et al., 2006)</span>. For AI agents, structural clarity is essential: agents working with well-organized code experience significantly lower break rates <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>1-3:</strong> Flat, easy to navigate</li>
<li><strong>4-5:</strong> Moderate depth, acceptable</li>
<li><strong>6-7:</strong> Deep, review if necessary</li>
<li><strong>8+:</strong> Very deep, likely over-organized</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Flatten unnecessary intermediate directories</li>
<li>Group by feature rather than layer when depth grows</li>
<li>Use package/module naming instead of deep nesting</li>
<li>Consider monorepo tools if managing many packages</li>
</ul>`,
},
```

### Citation Addition to descriptions.go (circular_deps)

```go
// Source: internal/output/descriptions.go
// Pattern for adding citations to circular_deps

"circular_deps": {
    Brief:     "Number of circular dependencies. Acyclic dependency structures are easier to understand, test, and maintain <span class=\"citation\">(Lakos, 1996)</span>.",
    Threshold: 7.0,
    Detailed: `<h4>Definition</h4>
<p>Counts the number of circular dependency chains where module A imports B which imports A (directly or transitively). Circular dependencies create ordering problems and make it impossible to understand modules in isolation.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Circular dependencies mean agents cannot understand one module without understanding all modules in the cycle. This creates reasoning complexity that scales with cycle size. Breaking cycles allows agents to analyze and modify modules independently.</p>

<h4>Research Evidence</h4>
<p>Parnas established that modular systems should have clear dependency direction—each module's design decisions should be hidden from others <span class="citation">(Parnas, 1972)</span>. Circular dependencies violate this principle by creating mutual knowledge requirements.</p>
<p>The Acyclic Dependencies Principle, articulated by Martin, states that the dependency graph of packages should have no cycles <span class="citation">(Martin, 2003)</span>. Note: This represents an influential practitioner perspective widely adopted in industry, though not derived from empirical research.</p>
<p>Lakos demonstrated practical techniques for eliminating cyclic dependencies in large systems, showing that acyclic physical dependencies dramatically reduce link-time costs and improve testability <span class="citation">(Lakos, 1996)</span>. Empirical research on 31 open-source Java systems found that circular dependencies correlate with higher change frequency in affected classes <span class="citation">(Oyetoyan et al., 2015)</span>.</p>
<p>For AI agents, architectural complexity directly impacts reliability: agents experience higher break rates when working with poorly-structured code <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>0:</strong> No cycles, excellent architecture</li>
<li><strong>1-2:</strong> Minor cycles, should be addressed</li>
<li><strong>3-5:</strong> Moderate, architectural debt</li>
<li><strong>6+:</strong> Significant cycles, major refactoring needed</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Introduce interface packages that both modules can depend on</li>
<li>Move shared functionality to a separate module</li>
<li>Use dependency injection to break compile-time cycles</li>
<li>Consider if modules should be merged if tightly coupled</li>
</ul>`,
},
```

### Citation Addition to citations.go

```go
// Source: internal/output/citations.go
// Pattern for adding reference entries

var researchCitations = []Citation{
    // Existing C3 entries (enhance, don't replace)...

    // NEW: Additional C3 citations
    {
        Category:    "C3",
        Title:       "Structured Design",
        Authors:     "Stevens, Myers & Constantine",
        Year:        1974,
        URL:         "https://doi.org/10.1147/sj.132.0115",
        Description: "Original coupling and cohesion definitions; low coupling improves software quality",
    },
    {
        Category:    "C3",
        Title:       "A Metrics Suite for Object Oriented Design",
        Authors:     "Chidamber & Kemerer",
        Year:        1994,
        URL:         "https://doi.org/10.1109/32.295895",
        Description: "CBO metric: excessive coupling detrimental to modular design and reuse",
    },
    {
        Category:    "C3",
        Title:       "Agile Software Development: Principles, Patterns, and Practices",
        Authors:     "Martin",
        Year:        2003,
        URL:         "https://www.pearson.com/en-us/subject-catalog/p/agile-software-development-principles-patterns-and-practices/P200000009452",
        Description: "Acyclic Dependencies Principle and Stable Dependencies Principle (influential practitioner perspective)",
    },
    {
        Category:    "C3",
        Title:       "Large-Scale C++ Software Design",
        Authors:     "Lakos",
        Year:        1996,
        URL:         "https://www.pearson.com/store/p/large-scale-c-software-design/P200000009117",
        Description: "Techniques for eliminating cyclic dependencies; acyclic designs easier to test and maintain",
    },
    {
        Category:    "C3",
        Title:       "Circular Dependencies and Change-Proneness: An Empirical Study",
        Authors:     "Oyetoyan et al.",
        Year:        2015,
        URL:         "https://doi.org/10.1109/SANER.2015.7081834",
        Description: "Empirical evidence that circular dependencies correlate with higher change frequency",
    },
    {
        Category:    "C3",
        Title:       "Exploring the Structure of Complex Software Designs",
        Authors:     "MacCormack et al.",
        Year:        2006,
        URL:         "https://doi.org/10.1287/mnsc.1060.0552",
        Description: "Design structure matrix analysis validates modularity benefits for maintainability",
    },
    {
        Category:    "C3",
        Title:       "Using Dependency Models to Manage Complex Software Architecture",
        Authors:     "Sangal et al.",
        Year:        2005,
        URL:         "https://doi.org/10.1145/1094855.1094915",
        Description: "DSM approach for specifying and enforcing architectural patterns like layering",
    },
    {
        Category:    "C3",
        Title:       "A Multi-Study Investigation into Dead Code",
        Authors:     "Romano et al.",
        Year:        2018,
        URL:         "https://doi.org/10.1109/TSE.2018.2842781",
        Description: "Dead code harms comprehensibility and maintainability; developers should avoid this smell",
    },
    {
        Category:    "C3",
        Title:       "M-score: An Empirically Derived Software Modularity Metric",
        Authors:     "Pisch et al.",
        Year:        2024,
        URL:         "https://doi.org/10.1145/3674805.3686697",
        Description: "Dependency density metrics correlate with project maintainability",
    },
    {
        Category:    "C3",
        Title:       "Code for Machines, Not Just Humans: Quantifying AI-Friendliness with Code Health Metrics",
        Authors:     "Borg et al.",
        Year:        2026,
        URL:         "https://arxiv.org/abs/2601.02200",
        Description: "Code health metrics including architecture predict AI agent reliability",
    },
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Intuitive modularity | Empirically-validated metrics | 2024 (M-score) | Quantified maintainability correlation |
| Principle-based architecture | DSM-validated structures | 2006 (MacCormack) | Objective architecture comparison |
| Assumed cycle harm | Empirically-measured change-proneness | 2015 (Oyetoyan) | Quantified cycle impact |
| Human-centric code quality | AI-agent-aware quality | 2026 (Borg et al.) | Agent break rates as quality indicator |

**Current best practices:**
- Architecture metrics have strong foundational backing (Parnas 1972, Stevens et al. 1974)
- Martin's principles are influential but should be labeled as practitioner perspective
- Empirical studies provide quantified evidence for coupling impact
- AI-era research (Borg et al. 2026) validates that architecture impacts agent reliability

## AI-Era Architecture Research

The AI-era (2021+) research landscape for architecture metrics differs from traditional research:

**Available AI-era sources for C3:**
- Borg et al. (2026): Code health metrics predict AI agent reliability. Architectural complexity is a component of code health.
- Pisch et al. (2024): M-score empirically validates that modularity metrics correlate with maintainability.

**Limited direct AI-era sources:**
- Most AI-era research focuses on code generation and LLM capabilities, not architecture metrics specifically
- No equivalent to Borg et al. specifically studying directory depth or circular dependencies and AI agent behavior
- Architecture research continues but without direct AI agent performance studies

**Recommendation:** Use Borg et al. (2026) for general AI relevance across all C3 metrics, noting it addresses code health broadly rather than architecture specifically. Rely on traditional empirical studies (Oyetoyan 2015, MacCormack 2006) for architecture-specific evidence.

## Open Questions

1. **Martin's principles vs empirical evidence**
   - What we know: Martin's principles (ADP, SDP) are widely adopted
   - What's unclear: Specific thresholds derived from empirical research
   - Recommendation: Present Martin as practitioner perspective; use Oyetoyan et al. for empirical backing

2. **Directory depth specific research**
   - What we know: Module decomposition principles (Parnas) apply
   - What's unclear: Specific depth thresholds from empirical studies
   - Recommendation: Cite Parnas for principle; note thresholds are practitioner consensus

3. **Import complexity metrics**
   - What we know: DSM research validates that simpler structures are better
   - What's unclear: Specific import complexity thresholds from research
   - Recommendation: Cite Sangal et al. for DSM approach; mark thresholds as practitioner consensus

4. **Dead exports vs dead code**
   - What we know: Romano et al. studied dead code comprehensively
   - What's unclear: Whether findings apply equally to dead exports specifically
   - Recommendation: Cite Romano et al. with note that study covers dead code broadly

## Sources

### Primary (HIGH confidence)

**Foundational Architecture Research:**
- [Parnas, 1972 - On the Criteria To Be Used in Decomposing Systems into Modules](https://doi.org/10.1145/361598.361623) - Seminal module decomposition and information hiding paper
- [Stevens, Myers & Constantine, 1974 - Structured Design](https://doi.org/10.1147/sj.132.0115) - Original coupling and cohesion definitions
- [Chidamber & Kemerer, 1994 - A Metrics Suite for Object Oriented Design](https://doi.org/10.1109/32.295895) - CBO and OO coupling metrics

**Empirical Architecture Studies:**
- [MacCormack et al., 2006 - Exploring the Structure of Complex Software Designs](https://doi.org/10.1287/mnsc.1060.0552) - DSM analysis validating modularity benefits
- [Oyetoyan et al., 2015 - Circular Dependencies and Change-Proneness](https://doi.org/10.1109/SANER.2015.7081834) - Empirical study of cycle impact on 31 Java systems
- [Romano et al., 2018 - A Multi-Study Investigation into Dead Code](https://doi.org/10.1109/TSE.2018.2842781) - Dead code harms comprehensibility

**AI-Era Sources:**
- [Borg et al., 2026 - Code for Machines, Not Just Humans](https://arxiv.org/abs/2601.02200) - Code health metrics predict AI agent reliability
- [Pisch et al., 2024 - M-score: Empirically Derived Modularity Metric](https://doi.org/10.1145/3674805.3686697) - Modularity metrics correlate with maintainability

### Secondary (MEDIUM confidence)

- [Martin, 2003 - Agile Software Development](https://www.pearson.com/en-us/subject-catalog/p/agile-software-development-principles-patterns-and-practices/P200000009452) - ADP, SDP principles (influential practitioner perspective, not peer-reviewed)
- [Lakos, 1996 - Large-Scale C++ Software Design](https://www.pearson.com/store/p/large-scale-c-software-design/P200000009117) - Practical techniques for acyclic designs (industry experience)
- [Sangal et al., 2005 - Using Dependency Models to Manage Architecture](https://doi.org/10.1145/1094855.1094915) - DSM approach for architecture management
- [Fowler et al., 1999 - Refactoring](https://martinfowler.com/books/refactoring.html) - Dead Code as code smell

### Tertiary (LOW confidence)

- [Malavolta et al., 2023 - JavaScript Dead Code](https://arxiv.org/abs/2308.16729) - Dead code removal improves performance (web-specific, may not generalize)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Existing infrastructure from Phase 18
- Architecture patterns: HIGH - Same patterns as C1 apply
- Foundational citations: HIGH - Classic papers well-established and DOI verified
- Empirical citations: HIGH - Peer-reviewed studies with DOIs
- Martin/Lakos practitioner sources: MEDIUM - Influential but not peer-reviewed research
- AI-era coverage: MEDIUM - Borg et al. addresses code health broadly, not architecture specifically

**Research date:** 2026-02-04
**Valid until:** 90 days (stable content domain, architecture research is mature)
