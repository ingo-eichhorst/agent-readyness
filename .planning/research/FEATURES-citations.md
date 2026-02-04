# Feature Research: Academic Citations in Technical Documentation

**Domain:** Scientific citations for metric descriptions in technical documentation
**Researched:** 2026-02-04
**Confidence:** HIGH (citation practices are well-established; the challenge is applying them appropriately to technical documentation context)

---

## Feature Landscape

### Table Stakes (Academic Credibility Requires These)

Features that engineering leaders expect when documentation claims scientific backing. Missing these undermines credibility.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Inline (Author, Year) format** | Standard academic citation format familiar to technical audiences. Enables quick source identification without interrupting reading flow. APA-style parenthetical citations are "the norm in scientific fields including engineering" ([UT Tyler Engineering](https://libguides.uttyler.edu/citations/engineering)). | LOW | Already specified in GitHub issues. Format: `(McCabe, 1976)` or `(Borg et al., 2026)` for 3+ authors. |
| **Complete References section per metric** | Readers expect full citation details to locate sources. IEEE and APA both require complete reference lists. Without them, inline citations are "placeholders without substance" ([Purdue OWL](https://owl.purdue.edu/owl/research_and_citation/apa_style/apa_formatting_and_style_guide/in_text_citations_author_authors.html)). | LOW | Format: Author (Year). Title. Publisher/Journal. URL/DOI. Include DOIs when available. |
| **Foundational source citations (pre-2021)** | Classic works (McCabe 1976, Fowler 1999, Martin 2003) establish theoretical foundations. "Seminal sources tend to be the major studies that initially presented an idea of great importance" ([Elon University](https://elon.libguides.com/c.php?g=1334811&p=9830885)). Citing only recent work appears shallow. | MEDIUM | 1-3 foundational sources per metric. These are often books, not papers (e.g., "Refactoring", "Clean Code"). |
| **AI/Agent era citations (2021+)** | Recent research validates that classic metrics matter for AI agents specifically. Without current citations, readers question "is this still relevant?" Research showing 15% error reduction is more compelling than theoretical arguments alone. | MEDIUM | 1-3 AI-era sources per metric. Focus on empirical studies (Borg et al. 2026, CrossCodeEval, SWE-bench). |
| **Verified, accessible URLs** | "Over 50% of cited links in Supreme Court opinions no longer point to the intended page" ([Wikipedia: Link rot](https://en.wikipedia.org/wiki/Link_rot)). Broken links destroy trust instantly. All URLs must work at publication time. | LOW | Test every URL before inclusion. Prefer DOIs for academic papers (permanent identifiers). |
| **Author attribution for claims** | Every quantified claim needs a source. "36-44% agent break rate increase" must cite Borg et al. (2026). Claims without attribution read as unsubstantiated opinion. "Proper documentation strengthens scientific argument by referring to work others have produced" ([U Toronto ECP](https://ecp.engineering.utoronto.ca/resources/online-handbook/accurate-documentation/)). | LOW | Apply rule: "No number without a name." Every statistic gets a citation. |

### Differentiators (What Separates Good from Mediocre Citations)

Features that elevate citation quality above baseline academic credibility.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Balanced foundational + recent blend** | Papers citing "a mix of recent and classic literature tend to have higher quotation rates" ([PLoS Medicine via Enago](https://www.enago.com/academy/how-many-citations-do-you-need-finding-the-right-amount-of-references-for-your-research-paper/)). The 1-3 foundational + 1-3 AI-era requirement creates this balance naturally. Demonstrates both theoretical grounding AND current relevance. | MEDIUM | Target ratio: approximately 40% foundational, 60% recent for AI-focused tool. |
| **Primary source preference** | Citing original research (McCabe's 1976 paper) over secondary summaries (blog post explaining McCabe) shows rigor. "Primary sources are the most direct evidence" ([UMN Crookston](https://crk.umn.edu/library/primary-secondary-and-tertiary-sources)). Engineering leaders notice when you cite the original rather than Wikipedia. | MEDIUM | Locate original papers, not citations-of-citations. Use Google Scholar's "Cited by" to find seminal works. |
| **DOIs for academic papers** | "A DOI is a permanent address for an article you're citing -- it will always refer to that article, and only that one" ([Scribbr](https://www.scribbr.com/citing-sources/what-is-a-doi/)). DOIs survive URL changes when publishers reorganize. Shows attention to long-term maintainability. | LOW | Format: `https://doi.org/10.xxxx/xxxxx`. Prefer over direct publisher URLs. |
| **Quantified impact statements** | "Agents break 36% more often on complex code" (Borg et al., 2026) is more compelling than "complexity matters." Specific numbers with citations create memorable, shareable findings. | LOW | Extract specific numbers from research. Include confidence intervals or study conditions when relevant. |
| **Context-appropriate citation density** | Neither over-cited (academic paper) nor under-cited (blog post). Technical documentation targets 2-6 citations per metric -- enough to establish credibility without cluttering readability. "Consider whether each citation supports a claim... quality over filling space" ([Enago Academy](https://www.enago.com/academy/how-many-citations-do-you-need-finding-the-right-amount-of-references-for-your-research-paper/)). | MEDIUM | 2-6 citations per metric is the sweet spot. More for complex metrics (complexity, coverage), fewer for simple ones. |
| **Explicit methodology transparency** | When citing studies, briefly note methodology: "based on 50+ LLM experiments" or "industry survey of 500 developers." Helps readers assess evidence quality without reading the full paper. | LOW | One-phrase methodology notes for key citations. Not required for foundational texts (McCabe's paper is the definition, not a study). |

### Anti-Features (Things to Deliberately NOT Do)

Features that seem valuable but create problems in technical documentation context.

| Anti-Feature | Why Requested | Why Problematic | Alternative |
|--------------|---------------|-----------------|-------------|
| **Exhaustive citation lists (10+ per metric)** | "More citations = more credible" assumption. Academic papers often have 50+ references. | Overwhelming for technical documentation. Readers want actionable insights, not literature reviews. "Most increases seemed to slow down after approximately 15 references per paper" ([Meadows 1974 via PMC](https://pmc.ncbi.nlm.nih.gov/articles/PMC8345841/)). Over-citation signals insecurity, not expertise. | 2-6 focused citations per metric. Quality curation demonstrates expertise. |
| **IEEE numbered citation style [1]** | Common in engineering journals. Compact inline references. | Obscures authorship while reading. "(Borg et al., 2026)" communicates more than "[3]" without consulting reference list. Engineering leaders scanning reports want quick attribution. | Author-year format for all citations. More readable for technical documentation audience. |
| **Footnote citations** | Keeps main text clean. Traditional in legal/historical writing. | Disrupts reading flow by forcing reader to scroll. Technical documentation is read non-linearly; footnotes get lost. Web documents work poorly with footnotes. | Inline parenthetical citations. References section at metric end. |
| **Citation of every claim** | Academic papers require citation for even obvious facts. | Technical documentation becomes unreadable: "Functions (definition: named code blocks [1]) that are long (over 25 lines [2]) are harder (subjective term [3]) to read." Obvious facts need no citation. | Cite only: (1) quantified claims, (2) non-obvious assertions, (3) methodology origins. Skip citations for widely accepted definitions. |
| **Mixing citation styles** | Different sources use different formats. Preserving original format shows "accuracy." | Inconsistency signals sloppiness. "(McCabe, 1976)" mixed with "McCabe [1976]" and "McCabe 1976" looks unprofessional. | Normalize all citations to (Author, Year) format. Consistency trumps source fidelity. |
| **Citation-only claims (no explanation)** | "As demonstrated by Borg et al. (2026)" without summarizing finding. Assumes reader has read source. | Readers cannot assess relevance without explanation. Creates false credibility through name-dropping. | Always include finding summary: "Agents break 36% more often on complex code (Borg et al., 2026)." |
| **Archiving all URLs via Wayback Machine** | Prevent link rot by creating permanent archives. Recommended practice for legal citations. | Maintenance burden for 33 metrics x 3-6 citations = 100-200 URLs. Wayback links are ugly and long. ARS is a living document that can update broken links. | Prefer DOIs (permanent by design). Use direct URLs for non-academic sources. Fix broken links as discovered. |
| **Machine-generated citation lists** | Tools like Zotero, Mendeley auto-generate references. Fast, reduces manual errors. | "If you use citation software... be sure to review the references it generates for any errors. These programs are not foolproof" ([Open Oregon Technical Writing](https://openoregon.pressbooks.pub/technicalwriting/chapter/5-1-citations/)). Auto-generated citations often have formatting issues, wrong years, ALL CAPS titles. | Manual citation formatting with template. Quality control through human review. |

---

## Feature Dependencies

```
[Inline Citation Format (Author, Year)]
    |
    +--requires--> [Foundational Source Research]
    |                   |
    |                   +--requires--> [Source Verification]
    |                                       |
    |                                       +--requires--> [URL Testing]
    |
    +--requires--> [AI-Era Source Research]
                        |
                        +--requires--> [Source Verification]
                                            |
                                            +--requires--> [URL Testing]

[References Section]
    |
    +--requires--> [All Inline Citations Complete]
    +--requires--> [DOI Resolution] (where available)
    +--requires--> [Full Bibliographic Data]

[Quantified Impact Statements]
    |
    +--enhances--> [Brief Descriptions]
    +--enhances--> [Detailed Descriptions]
    +--requires--> [Primary Source Access]
```

### Dependency Notes

- **All citations require source verification:** Cannot write references until sources are located and verified accessible.
- **DOIs enhance URL stability:** Finding DOIs should happen during source research, not as a separate step.
- **Inline citations and References sections are coupled:** Cannot have one without the other. Write together.
- **Quantified claims require primary sources:** Cannot cite "36% break rate" from a blog summarizing Borg et al.; must cite Borg et al. directly.

---

## MVP Definition

### Launch With (v0.0.4)

Minimum viable citation quality -- what's needed to claim "research-backed" credibility.

- [x] **Inline (Author, Year) citations** -- Core credibility signal
- [x] **References section per metric** -- Enables source verification
- [x] **1-3 foundational sources per metric** -- Establishes theoretical grounding
- [x] **1-3 AI-era sources per metric** -- Demonstrates current relevance
- [x] **All URLs verified accessible** -- No broken links at release
- [x] **Quantified claims attributed** -- Every number has a source

### Add After Validation (v0.0.5+)

Features to add once core citations are working.

- [ ] **DOI prioritization** -- Replace direct URLs with DOIs where available
- [ ] **Methodology transparency notes** -- Add study context for key citations
- [ ] **Citation count per category summary** -- Track research coverage

### Future Consideration (v1.0+)

Features to defer until product-market fit is established.

- [ ] **Citation database export** -- BibTeX or similar for users citing ARS research
- [ ] **Automated link checking in CI** -- Catch link rot before release
- [ ] **Interactive references (expandable)** -- Show/hide full citation on click

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Inline (Author, Year) format | HIGH | LOW | P1 |
| Complete References section | HIGH | LOW | P1 |
| Foundational sources (1-3/metric) | HIGH | MEDIUM | P1 |
| AI-era sources (1-3/metric) | HIGH | MEDIUM | P1 |
| Verified URLs | HIGH | LOW | P1 |
| Quantified impact statements | HIGH | LOW | P1 |
| DOI preference | MEDIUM | LOW | P2 |
| Primary source preference | MEDIUM | MEDIUM | P2 |
| Methodology transparency | MEDIUM | LOW | P2 |
| Balanced citation density (2-6) | MEDIUM | LOW | P2 |

**Priority key:**
- P1: Must have for v0.0.4 launch (core credibility)
- P2: Should have, add when time permits (quality polish)
- P3: Nice to have, future consideration

---

## Quality Criteria for Citation Work

### Per-Citation Checklist

- [ ] URL verified accessible (click-tested)
- [ ] DOI used if available (format: `https://doi.org/10.xxxx`)
- [ ] Author name(s) correct (check paper directly, not secondary sources)
- [ ] Year matches actual publication (not blog post year for papers)
- [ ] Title matches exactly (copy from source)
- [ ] Inline citation format: `(Author, Year)` or `(Author et al., Year)`

### Per-Metric Checklist

- [ ] 1-3 foundational sources (pre-2021) cited
- [ ] 1-3 AI/agent era sources (2021+) cited
- [ ] Total 2-6 citations (not under-cited, not over-cited)
- [ ] All quantified claims have attribution
- [ ] References section complete with full bibliographic data
- [ ] No broken URLs

### Category-Level Checklist

- [ ] Citation style consistent across all metrics in category
- [ ] No duplicate citations within metric (cite once, reference once)
- [ ] Foundational/AI-era ratio approximately 40%/60%
- [ ] Key sources not overused (e.g., Borg et al. cited at most 3-4 times per category)

---

## Competitor Feature Analysis

| Feature | Academic Papers | SonarQube | CodeClimate | ARS (Our Approach) |
|---------|-----------------|-----------|-------------|-------------------|
| Citation format | IEEE/APA | None | None | (Author, Year) inline |
| Research backing | Required | Claims without sources | Blog-level | Full citations with URLs |
| Foundational sources | Expected | N/A | N/A | 1-3 per metric |
| AI-era sources | Field-specific | N/A | N/A | 1-3 per metric |
| References section | Yes (Bibliography) | No | No | Per-metric References |
| URL verification | Author responsibility | N/A | N/A | Pre-release testing |
| Quantified claims | Sourced | Unsourced thresholds | Unsourced thresholds | All sourced |

### Key Competitive Insight

**No competitor provides research-backed metric explanations.** SonarQube documents *what* metrics measure but not *why* the thresholds matter. CodeClimate provides recommendations without empirical backing. ARS will be the only tool where engineering leaders can trace metric significance to peer-reviewed research.

This differentiation aligns with target user: "Engineering leaders prioritizing investment" who need justification for improvement efforts. A claim like "keep complexity under 10" is more compelling when backed by "(McCabe, 1976; Borg et al., 2026)" than when stated as product opinion.

---

## Sources

### Citation Format Standards
- [Technical Writing Citations (Open Oregon)](https://openoregon.pressbooks.pub/technicalwriting/chapter/5-1-citations/)
- [APA Style for Engineering (UT Tyler)](https://libguides.uttyler.edu/citations/engineering)
- [IEEE Citation Format Guide (Sourcely)](https://www.sourcely.net/post/ieee-citation-format-a-complete-guide-for-engineering-and-computer-science-students)
- [Purdue OWL APA In-Text Citations](https://owl.purdue.edu/owl/research_and_citation/apa_style/apa_formatting_and_style_guide/in_text_citations_author_authors.html)

### Citation Density and Quality
- [How Many Citations Do You Need? (Enago Academy)](https://www.enago.com/academy/how-many-citations-do-you-need-finding-the-right-amount-of-references-for-your-research-paper/)
- [Reference Density Growth 1980-2019 (PMC)](https://pmc.ncbi.nlm.nih.gov/articles/PMC8345841/)

### Link Stability and DOIs
- [DOIs vs URLs (Purdue OWL)](https://owl.purdue.edu/owl/research_and_citation/conducting_research/internet_references/urls_vs_dois.html)
- [What is a DOI? (Scribbr)](https://www.scribbr.com/citing-sources/what-is-a-doi/)
- [Link Rot (Wikipedia)](https://en.wikipedia.org/wiki/Link_rot)

### Primary vs Secondary Sources
- [Foundational Works and Primary Sources (Elon University)](https://elon.libguides.com/c.php?g=1334811&p=9830885)
- [Primary vs Secondary Sources (UMN Crookston)](https://crk.umn.edu/library/primary-secondary-and-tertiary-sources)

### Engineering Documentation Credibility
- [Accurate Documentation (U Toronto ECP)](https://ecp.engineering.utoronto.ca/resources/online-handbook/accurate-documentation/)
- [Backing Claims with Sources (CollabWriting)](https://blog.collabwriting.com/creditbiliti-essential-tips-for-backing-your-claims/)

### Balancing Old and New Citations
- [Old Classics Being Crowded Out (Nature Index)](https://www.nature.com/nature-index/news/the-growth-of-papers-is-crowding-out-old-classics)
- [Older Papers Increasingly Remembered (Science)](https://www.science.org/content/article/older-papers-are-increasingly-remembered-and-cited)

---
*Feature research for: Academic Citations in Technical Documentation*
*Researched: 2026-02-04*
