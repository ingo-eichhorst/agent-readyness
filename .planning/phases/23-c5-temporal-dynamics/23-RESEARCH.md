# Phase 23: C5 Temporal Dynamics - Research

**Researched:** 2026-02-05
**Domain:** Academic citations for C5 Temporal Dynamics metrics (code churn, temporal coupling, author fragmentation, commit stability, hotspot concentration)
**Confidence:** HIGH

## Summary

This phase adds research-backed citations to all 5 C5 Temporal Dynamics metrics following the quality protocols established in Phase 18. The 5 C5 metrics (churn_rate, temporal_coupling_pct, author_fragmentation, commit_stability, hotspot_concentration) currently have minimal citations (Kim et al. 2007, Tornhill 2015) that need enhancement with both foundational change history research and AI-era empirical evidence.

Temporal dynamics research is well-established in software engineering, with seminal work from the 1990s-2000s establishing the predictive power of change history for defects. Key foundational sources include Graves et al. (2000) on change history and fault prediction, Nagappan & Ball (2005) on code churn, Gall et al. (1998) on logical coupling, and D'Ambros et al. (2009) on change coupling and defects. Bird et al. (2011) provides the definitive study on code ownership and defects. Tornhill's "Your Code as a Crime Scene" (2015) synthesizes this research into practitioner guidance.

The C5 category has strong foundational research but less direct AI-era validation compared to C1 code health metrics. Borg et al. (2026) confirms that code health (which encompasses structural metrics) predicts agent reliability, but does not specifically test temporal metrics. The connection is indirect but valid: temporal metrics predict defects, and defect-prone code is harder for AI agents.

**Primary recommendation:** Use Graves et al. (2000), Nagappan & Ball (2005), and Kim et al. (2007) as primary churn sources; Gall et al. (1998) and D'Ambros et al. (2009) for temporal coupling; Bird et al. (2011) for author fragmentation; Tornhill (2015) for practitioner synthesis across all metrics. Note that Tornhill is practitioner literature, not peer-reviewed research. Commit stability has the weakest dedicated research; use code decay literature (Eick et al. 2001) as closest proxy.

## Standard Stack

This phase requires **zero new Go dependencies**. All infrastructure exists from Phase 18.

### Core (Existing Infrastructure)

| Component | File | Purpose | Status |
|-----------|------|---------|--------|
| Citation struct | `internal/output/citations.go` | Stores Category, Title, Authors, Year, URL, Description | **Use as-is** |
| researchCitations | `internal/output/citations.go` | Array of Citation entries | **Expand** with C5 entries |
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
+-- citations.go          # Citation{} struct, researchCitations[]
+-- descriptions.go       # metricDescriptions{} with inline citations
+-- html.go              # buildHTMLCategories(), filterCitationsByCategory()
+-- templates/
    +-- report.html      # Per-category References sections
    +-- styles.css       # .citation class styling
```

### Pattern 1: Inline Citation Markup (Same as C1-C4)

**What:** Citations appear inline as `(Author, Year)` within metric descriptions
**When to use:** In the `Detailed` field of `MetricDescription` structs
**Example:**
```go
// Source: internal/output/descriptions.go (existing pattern)
Detailed: `...
<p>Research shows code churn strongly predicts defects <span class="citation">(Kim et al., 2007)</span>.</p>
...`,
```

### Pattern 2: Research Evidence Subsection (Same as C1-C4)

**What:** Dedicated "Research Evidence" subsection in detailed descriptions
**When to use:** All metrics with quantified claims
**Example:**
```go
Detailed: `<h4>Definition</h4>
<p>[Factual definition - no citations needed]</p>

<h4>Why It Matters for AI Agents</h4>
<p>[Explanation - 0-1 citations if specific claim]</p>

<h4>Research Evidence</h4>
<p>[2-3 citations here - primary evidence location]</p>

<h4>Recommended Thresholds</h4>
<ul><li>[Thresholds - cite if from specific research]</li></ul>

<h4>How to Improve</h4>
<ul><li>[Actionable guidance - no citations needed]</li></ul>`,
```

### Pattern 3: Balancing Academic and Practitioner Sources

**What:** C5 metrics bridge academic research and practitioner synthesis
**When to use:** All C5 citations should include both peer-reviewed research and label practitioner sources appropriately
**Guidelines:**
- **Peer-reviewed research (cite normally):** Graves et al. (2000), Nagappan & Ball (2005), Kim et al. (2007), D'Ambros et al. (2009), Bird et al. (2011)
- **Practitioner literature (label accordingly):** Tornhill (2015) - influential practitioner perspective synthesizing temporal analysis
- **AI-era (applicable with caveat):** Borg et al. (2026) - code health broadly, not temporal metrics specifically

### Anti-Patterns to Avoid

- **Over-citation:** Do NOT add 5+ citations per metric. Target 2-3 focused citations.
- **Citation in "How to Improve":** Actionable guidance needs no citations.
- **Citing Tornhill as peer-reviewed research:** Label as practitioner perspective.
- **Claiming AI-era validation for temporal metrics:** Borg et al. tests code health, not temporal dynamics specifically.

## Don't Hand-Roll

Problems with existing solutions that should NOT be rebuilt:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Citation storage | New database/JSON files | Existing `citations.go` slice | Simple, already works |
| URL verification | CI pipeline with Lychee | Manual `curl -I` checks | One-time task, overkill to automate |
| Citation formatting | BibTeX parser | Manual `(Author, Year)` strings | ~12 C5 citations don't justify tooling |

**Key insight:** This is a content expansion task, not infrastructure build. Phase 18 established all needed infrastructure.

## Common Pitfalls

### Pitfall 1: Conflating Temporal Coupling Research

**What goes wrong:** Using "temporal coupling" loosely to mean different things.
**Why it happens:** Term used differently by different researchers (logical coupling, change coupling, co-change patterns).
**How to avoid:**
- Gall et al. (1998) introduced "logical coupling" based on release history
- D'Ambros et al. (2009) studied "change coupling" and defect correlation
- Use consistent terminology: files that change together in commits
**Warning signs:** Mixing "logical coupling" and "structural coupling" concepts.

### Pitfall 2: Overstating Tornhill as Research

**What goes wrong:** Citing Tornhill (2015) as if it were peer-reviewed empirical research.
**Why it happens:** The book is well-known and frequently cited.
**How to avoid:**
- Cite Tornhill as practitioner perspective synthesizing temporal analysis
- For empirical claims, use academic sources (Graves, Nagappan, Kim, D'Ambros)
- Label appropriately: "influential practitioner perspective"
**Warning signs:** Tornhill citations for quantified claims without academic backup.

### Pitfall 3: Missing Foundational Change History Research

**What goes wrong:** Citing only Kim et al. (2007) without earlier foundational work.
**Why it happens:** Kim et al. is highly cited and accessible.
**How to avoid:**
- Include Graves et al. (2000) for process measures vs. product metrics
- Include Nagappan & Ball (2005) for relative code churn
- Kim et al. (2007) builds on and cites earlier work
**Warning signs:** C5 metrics lacking pre-2005 citations for established concepts.

### Pitfall 4: Claiming Direct AI-Era Validation

**What goes wrong:** Stating that temporal metrics have been empirically validated for AI agent performance.
**Why it happens:** Desire to make C5 equivalent to C1 in evidence strength.
**How to avoid:**
- Borg et al. (2026) validates CodeHealth (structural metrics), not temporal metrics
- Connection is indirect: temporal metrics predict defects; defect-prone code is harder for agents
- Use hedged language: "temporal metrics predict defect-prone areas, which correlate with agent difficulty"
**Warning signs:** Claims like "temporal coupling increases agent break rates by X%".

### Pitfall 5: Commit Stability Research Gap

**What goes wrong:** Presenting commit stability thresholds as research-backed when research is thin.
**Why it happens:** Desire for consistency with other well-researched metrics.
**How to avoid:**
- Commit stability as a specific metric has limited dedicated research
- Code decay literature (Eick et al. 2001) provides closest foundation
- Tornhill covers commit patterns as practitioner guidance
- Acknowledge research gap, present thresholds as practitioner consensus
**Warning signs:** Specific commit stability percentages attributed to research.

## C5 Metrics: Required Citations

### churn_rate

| Type | Source | Year | DOI/URL | Status | Key Finding |
|------|--------|------|---------|--------|-------------|
| Foundational | Graves et al., "Predicting Fault Incidence Using Software Change History" | 2000 | 10.1109/32.859533 | Verified | Process measures (change history) better predict faults than product metrics |
| Foundational | Nagappan & Ball, "Use of Relative Code Churn Measures to Predict System Defect Density" | 2005 | 10.1145/1062455.1062514 | Verified | Relative churn measures predict defect density with 89% accuracy |
| Foundational | Kim et al., "Predicting Faults from Cached History" | 2007 | 10.1109/ICSE.2007.66 | Verified | Change history predicts fault-prone files; caching strategy for predictions |
| Practitioner | Tornhill, "Your Code as a Crime Scene" | 2015 | ISBN 978-1-68050-038-7 | Verified | Churn analysis as forensic technique; high-churn = complexity hotspots |
| AI-Era | Borg et al., "Code for Machines" | 2026 | arxiv.org/abs/2601.02200 | Verified | Code health predicts agent reliability (indirect support) |

### temporal_coupling_pct

| Type | Source | Year | DOI/URL | Status | Key Finding |
|------|--------|------|---------|--------|-------------|
| Foundational | Gall et al., "Detection of Logical Coupling Based on Product Release History" | 1998 | 10.1109/ICSM.1998.738508 | Verified | Pioneered logical coupling detection from change history |
| Empirical | D'Ambros et al., "On the Relationship Between Change Coupling and Software Defects" | 2009 | 10.1109/WCRE.2009.19 | Verified | Change coupling correlates with defects; improves bug prediction models |
| Practitioner | Tornhill, "Your Code as a Crime Scene" | 2015 | ISBN 978-1-68050-038-7 | Verified | Temporal coupling reveals hidden dependencies not in code structure |
| AI-Era | Borg et al., "Code for Machines" | 2026 | arxiv.org/abs/2601.02200 | Verified | Code health predicts agent reliability (indirect support) |

### author_fragmentation

| Type | Source | Year | DOI/URL | Status | Key Finding |
|------|--------|------|---------|--------|-------------|
| Foundational | Bird et al., "Don't Touch My Code! Examining the Effects of Ownership on Software Quality" | 2011 | 10.1145/2025113.2025119 | Verified | Ownership measures relate to pre-release and post-release faults; low-expertise contributors increase defects |
| Foundational | Kim et al., "Predicting Faults from Cached History" | 2007 | 10.1109/ICSE.2007.66 | Verified | Includes developer contribution patterns in fault prediction |
| Practitioner | Tornhill, "Your Code as a Crime Scene" | 2015 | ISBN 978-1-68050-038-7 | Verified | Author fragmentation as indicator of knowledge silos |
| AI-Era | Borg et al., "Code for Machines" | 2026 | arxiv.org/abs/2601.02200 | Verified | Code health predicts agent reliability (indirect support) |

### commit_stability

| Type | Source | Year | DOI/URL | Status | Key Finding |
|------|--------|------|---------|--------|-------------|
| Foundational | Eick et al., "Does Code Decay? Assessing the Evidence from Change Management Data" | 2001 | 10.1109/32.895984 | Verified | Defines code decay; change patterns as symptoms/predictors of decay |
| Foundational | Graves et al., "Predicting Fault Incidence Using Software Change History" | 2000 | 10.1109/32.859533 | Verified | Modification patterns predict future defects |
| Practitioner | Tornhill, "Your Code as a Crime Scene" | 2015 | ISBN 978-1-68050-038-7 | Verified | Commit patterns as indicators of code maturity |
| AI-Era | Borg et al., "Code for Machines" | 2026 | arxiv.org/abs/2601.02200 | Verified | Code health predicts agent reliability (indirect support) |

### hotspot_concentration

| Type | Source | Year | DOI/URL | Status | Key Finding |
|------|--------|------|---------|--------|-------------|
| Foundational | Nagappan & Ball, "Use of Relative Code Churn Measures to Predict System Defect Density" | 2005 | 10.1145/1062455.1062514 | Verified | Churn concentration identifies high-defect-density components |
| Foundational | Hassan, "Predicting Faults Using the Complexity of Code Changes" | 2009 | 10.1109/ICSE.2009.5070510 | Verified | Change complexity as fault predictor; identifies change hotspots |
| Practitioner | Tornhill, "Your Code as a Crime Scene" | 2015 | ISBN 978-1-68050-038-7 | Verified | Hotspots as primary refactoring targets; Pareto principle in code changes |
| AI-Era | Borg et al., "Code for Machines" | 2026 | arxiv.org/abs/2601.02200 | Verified | Code health predicts agent reliability (indirect support) |

## Code Examples

### Citation Addition to descriptions.go (churn_rate)

```go
// Source: internal/output/descriptions.go
// Pattern for adding citations to churn_rate

"churn_rate": {
    Brief:     "Average code changes per file over time. Code churn strongly predicts defect-prone areas <span class=\"citation\">(Kim et al., 2007)</span>.",
    Threshold: 6.0,
    Detailed: `<h4>Definition</h4>
<p>Measures how frequently code changes over time, calculated from git history. High churn indicates files that are modified often, potentially due to instability, evolving requirements, or maintenance burden.</p>

<h4>Why It Matters for AI Agents</h4>
<p>High-churn code is more likely to change again soon, increasing the risk that agent modifications will conflict with ongoing work. Stable code provides a reliable foundation for agent changes. Churn also correlates with defect density, meaning high-churn areas are riskier for automated modification.</p>

<h4>Research Evidence</h4>
<p>Foundational research established that process measures derived from change history are more predictive of faults than product metrics like code size <span class="citation">(Graves et al., 2000)</span>. Nagappan and Ball demonstrated that relative code churn measures (churn normalized by component size) predict system defect density with 89% accuracy on Windows Server 2003 <span class="citation">(Nagappan & Ball, 2005)</span>.</p>
<p>Kim et al. extended this work, showing that change history patterns using a cache-based strategy effectively predict fault-prone files across seven software systems <span class="citation">(Kim et al., 2007)</span>. Tornhill synthesizes this research into practitioner guidance, identifying high-churn files as complexity hotspots requiring special attention <span class="citation">(Tornhill, 2015)</span>. Note: Tornhill is practitioner literature synthesizing academic research.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>0-2:</strong> Stable, low-risk modifications</li>
<li><strong>3-5:</strong> Moderate churn, normal development</li>
<li><strong>6-10:</strong> High churn, review stability</li>
<li><strong>11+:</strong> Very high, potential instability issues</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Stabilize high-churn code through refactoring</li>
<li>Identify and address root causes of frequent changes</li>
<li>Add tests to catch regressions earlier</li>
<li>Review if code is in appropriate abstraction layer</li>
</ul>`,
},
```

### Citation Addition to descriptions.go (temporal_coupling_pct)

```go
// Source: internal/output/descriptions.go
// Pattern for adding citations to temporal_coupling_pct

"temporal_coupling_pct": {
    Brief:     "Files that change together. Temporal coupling reveals hidden dependencies not visible in code structure <span class=\"citation\">(Gall et al., 1998)</span>.",
    Threshold: 6.0,
    Detailed: `<h4>Definition</h4>
<p>Percentage of file pairs that frequently change together in commits but have no direct import relationship. Indicates hidden coupling not visible in code structure but present in change patterns.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Temporal coupling reveals hidden dependencies that agents cannot see from code alone. When files are temporally coupled, changing one without the other often introduces bugs. Agents may miss these implicit relationships.</p>

<h4>Research Evidence</h4>
<p>Gall et al. pioneered the detection of "logical coupling" from product release history, demonstrating that change patterns reveal architectural dependencies not apparent from static code analysis <span class="citation">(Gall et al., 1998)</span>. This foundational work established that modules changing together often indicate design issues or restructuring opportunities.</p>
<p>D'Ambros et al. empirically validated that change coupling correlates with software defects across three large systems, and that incorporating change coupling information improves bug prediction models <span class="citation">(D'Ambros et al., 2009)</span>. Tornhill synthesizes this research, showing how temporal coupling analysis reveals hidden dependencies requiring attention <span class="citation">(Tornhill, 2015)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>0-10%:</strong> Low coupling, independent modules</li>
<li><strong>11-25%:</strong> Moderate, some hidden dependencies</li>
<li><strong>26-50%:</strong> High, architectural concerns</li>
<li><strong>51%+:</strong> Very high, significant hidden coupling</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Co-locate files that always change together</li>
<li>Extract shared functionality into explicit dependencies</li>
<li>Document cross-file dependencies that must exist</li>
<li>Review if module boundaries are correct</li>
</ul>`,
},
```

### Citation Addition to citations.go

```go
// Source: internal/output/citations.go
// Pattern for adding C5 reference entries

var researchCitations = []Citation{
    // Existing C5 entries (update descriptions)...
    {
        Category:    "C5",
        Title:       "Your Code as a Crime Scene",
        Authors:     "Tornhill",
        Year:        2015,
        URL:         "https://pragprog.com/titles/atcrime/your-code-as-a-crime-scene/",
        Description: "Practitioner synthesis of temporal analysis: churn, coupling, hotspots (ISBN 978-1-68050-038-7)",
    },
    {
        Category:    "C5",
        Title:       "Predicting Faults from Cached History",
        Authors:     "Kim et al.",
        Year:        2007,
        URL:         "https://doi.org/10.1109/ICSE.2007.66",
        Description: "Change history predicts fault-prone files using cache-based strategy",
    },

    // NEW: Additional C5 citations
    {
        Category:    "C5",
        Title:       "Predicting Fault Incidence Using Software Change History",
        Authors:     "Graves et al.",
        Year:        2000,
        URL:         "https://doi.org/10.1109/32.859533",
        Description: "Process measures from change history outperform product metrics for fault prediction",
    },
    {
        Category:    "C5",
        Title:       "Use of Relative Code Churn Measures to Predict System Defect Density",
        Authors:     "Nagappan & Ball",
        Year:        2005,
        URL:         "https://doi.org/10.1145/1062455.1062514",
        Description: "Relative churn measures predict defect density with 89% accuracy",
    },
    {
        Category:    "C5",
        Title:       "Detection of Logical Coupling Based on Product Release History",
        Authors:     "Gall et al.",
        Year:        1998,
        URL:         "https://doi.org/10.1109/ICSM.1998.738508",
        Description: "Pioneered logical coupling detection from change history; identifies restructuring opportunities",
    },
    {
        Category:    "C5",
        Title:       "On the Relationship Between Change Coupling and Software Defects",
        Authors:     "D'Ambros et al.",
        Year:        2009,
        URL:         "https://doi.org/10.1109/WCRE.2009.19",
        Description: "Change coupling correlates with defects; improves bug prediction models",
    },
    {
        Category:    "C5",
        Title:       "Don't Touch My Code! Examining the Effects of Ownership on Software Quality",
        Authors:     "Bird et al.",
        Year:        2011,
        URL:         "https://doi.org/10.1145/2025113.2025119",
        Description: "Ownership measures relate to faults; low-expertise contributors increase defects",
    },
    {
        Category:    "C5",
        Title:       "Does Code Decay? Assessing the Evidence from Change Management Data",
        Authors:     "Eick et al.",
        Year:        2001,
        URL:         "https://doi.org/10.1109/32.895984",
        Description: "Defines code decay; change patterns as symptoms and predictors of decay",
    },
    {
        Category:    "C5",
        Title:       "Predicting Faults Using the Complexity of Code Changes",
        Authors:     "Hassan",
        Year:        2009,
        URL:         "https://doi.org/10.1109/ICSE.2009.5070510",
        Description: "Change complexity predicts faults; identifies change hotspots",
    },
    {
        Category:    "C5",
        Title:       "Code for Machines, Not Just Humans: Quantifying AI-Friendliness with Code Health Metrics",
        Authors:     "Borg et al.",
        Year:        2026,
        URL:         "https://arxiv.org/abs/2601.02200",
        Description: "Code health metrics predict AI agent reliability (indirect support for temporal metrics)",
    },
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Product metrics (LOC, complexity) | Process metrics (change history) | 2000 (Graves) | Process measures more predictive than product metrics |
| Absolute churn measures | Relative churn measures | 2005 (Nagappan) | Normalized churn achieves 89% prediction accuracy |
| Static coupling analysis | Temporal/logical coupling | 1998 (Gall) | Reveals hidden dependencies not in code structure |
| Individual ownership | Ownership fragmentation effects | 2011 (Bird) | Multiple low-expertise contributors increase defects |

**Current best practices:**
- Temporal dynamics research has strong empirical foundation (1998-2011)
- Multiple independent studies confirm change history predicts defects
- Practitioner synthesis available (Tornhill 2015)
- AI-era validation indirect but logical (defect-prone code harder for agents)

## Temporal Research Landscape

The temporal dynamics field differs from code health (C1) and architecture (C3):

**Available high-quality sources for C5:**
- Strong foundational research from 1998-2011 (Gall, Graves, Nagappan, Kim, D'Ambros, Bird)
- Consistent findings across multiple studies and systems
- Well-established practitioner synthesis (Tornhill)
- IEEE/ACM peer-reviewed with DOIs

**Limitations:**
- AI-era research does not specifically test temporal metrics
- Commit stability as a specific metric has limited dedicated research
- Thresholds are practitioner consensus, not empirically derived
- Tornhill is practitioner literature, not peer-reviewed

**Recommendation:** Use foundational academic sources for empirical claims. Label Tornhill appropriately as practitioner perspective. Acknowledge that AI-agent validation is indirect: temporal metrics predict defect-prone areas, which are harder for agents to modify successfully.

## Open Questions

1. **Commit stability specific research**
   - What we know: Code decay research (Eick 2001) covers change patterns broadly
   - What's unclear: Specific commit stability ratio thresholds from empirical research
   - Recommendation: Present thresholds as practitioner consensus, cite code decay literature

2. **AI-agent validation for temporal metrics**
   - What we know: Borg et al. (2026) validates code health (structural metrics)
   - What's unclear: Whether temporal metrics specifically correlate with agent break rates
   - Recommendation: Use hedged language; connection is indirect but logical

3. **Hotspot concentration optimal thresholds**
   - What we know: Pareto principle (20/80) is common heuristic
   - What's unclear: Empirically validated thresholds for hotspot concentration
   - Recommendation: Cite Pareto as practitioner heuristic, not research-derived

4. **Author fragmentation optimal range**
   - What we know: Bird et al. found ownership effects on defects
   - What's unclear: Specific fragmentation thresholds beyond "clear ownership vs. fragmented"
   - Recommendation: Cite Bird for principle; thresholds are practitioner guidance

## Sources

### Primary (HIGH confidence)

**Foundational Change History Research:**
- [Graves et al., 2000 - Predicting Fault Incidence Using Software Change History](https://doi.org/10.1109/32.859533) - IEEE TSE; process measures outperform product metrics
- [Nagappan & Ball, 2005 - Use of Relative Code Churn Measures to Predict System Defect Density](https://doi.org/10.1145/1062455.1062514) - ICSE; 89% accuracy on Windows Server 2003
- [Kim et al., 2007 - Predicting Faults from Cached History](https://doi.org/10.1109/ICSE.2007.66) - ICSE; cache-based fault prediction

**Temporal Coupling Research:**
- [Gall et al., 1998 - Detection of Logical Coupling Based on Product Release History](https://doi.org/10.1109/ICSM.1998.738508) - ICSM; pioneered logical coupling
- [D'Ambros et al., 2009 - On the Relationship Between Change Coupling and Software Defects](https://doi.org/10.1109/WCRE.2009.19) - WCRE; change coupling correlates with defects

**Ownership Research:**
- [Bird et al., 2011 - Don't Touch My Code!](https://doi.org/10.1145/2025113.2025119) - FSE; ownership measures relate to faults

**Code Decay Research:**
- [Eick et al., 2001 - Does Code Decay?](https://doi.org/10.1109/32.895984) - IEEE TSE; defines and measures code decay
- [Hassan, 2009 - Predicting Faults Using the Complexity of Code Changes](https://doi.org/10.1109/ICSE.2009.5070510) - ICSE; change complexity as predictor

### Secondary (MEDIUM confidence)

- [Tornhill, 2015 - Your Code as a Crime Scene](https://pragprog.com/titles/atcrime/your-code-as-a-crime-scene/) - Practitioner synthesis (ISBN 978-1-68050-038-7)
- [Borg et al., 2026 - Code for Machines](https://arxiv.org/abs/2601.02200) - AI-era code health (indirect support for temporal)

### Tertiary (LOW confidence)

- Specific numerical thresholds for temporal metrics (practitioner consensus)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Existing infrastructure from Phase 18
- Architecture patterns: HIGH - Same patterns as C1-C4 apply
- Churn citations: HIGH - Multiple peer-reviewed empirical studies
- Temporal coupling citations: HIGH - Foundational Gall (1998) and empirical D'Ambros (2009)
- Author fragmentation citations: HIGH - Bird et al. (2011) is definitive study
- Commit stability citations: MEDIUM - Code decay literature as proxy; specific metric less researched
- Hotspot citations: HIGH - Nagappan (2005) and Hassan (2009) directly relevant
- AI-era coverage: MEDIUM - Borg et al. addresses code health broadly, not temporal specifically

**Research date:** 2026-02-05
**Valid until:** 90 days (stable content domain; foundational research is well-established)
