# Pitfalls Research: Academic Citations in Technical Documentation

**Domain:** Adding scientific citations to metric descriptions (ARS v0.0.4)
**Researched:** 2026-02-04
**Confidence:** HIGH (verified with academic literature, library guides, and citation error studies)

This document catalogs pitfalls specific to adding academic citations to existing technical documentation, with a focus on:
1. Maintaining credibility for developer audience
2. Balancing citation density with readability
3. Ensuring URL/reference longevity
4. Mixing foundational (pre-2021) and AI-era (2021+) research

---

## Critical Pitfalls

These mistakes undermine credibility or cause user-facing issues.

### 1. Broken URLs and Link Rot

**What goes wrong:**
URLs in citations become inaccessible over time. Studies show 23% of cited URLs are broken on average, rising to 50% for older articles. Academic URLs have a half-life of approximately 4-14 years. Users clicking citation links encounter 404 errors, damaging trust in the documentation.

**Why it happens:**
- Publishers reorganize their websites
- Institutional repositories change URL structures
- Personal websites and blogs disappear
- Conference proceedings move or are taken offline
- ArXiv papers may be retracted or superseded

**How to avoid:**
1. **Prefer DOIs over URLs** - DOIs (Digital Object Identifiers) are designed to be permanent. Use `https://doi.org/10.xxxx/xxxxx` format, which resolves even if the publisher changes URLs
2. **Verify all URLs at submission time** - Check every link actually loads the expected content
3. **Use stable sources hierarchy:**
   - Best: DOI links (IEEE, ACM, Springer, Nature)
   - Good: ArXiv permanent URLs (`https://arxiv.org/abs/xxxx.xxxxx`)
   - Acceptable: Publisher landing pages (not deep PDFs)
   - Avoid: Blog posts, personal sites, Medium articles
4. **Archive critical non-DOI sources** - Use Perma.cc or Wayback Machine for sources without DOIs
5. **Include enough metadata to find sources manually** - Authors, year, title, publication venue allows lookup even if URL breaks

**Warning signs:**
- URL contains session IDs, timestamps, or query parameters
- URL points to a PDF directly instead of landing page
- Source is on a personal website, blog, or social media
- URL contains "preview" or "temp" in path
- No DOI exists for a peer-reviewed publication

**Phase to address:**
Phase 1 (first category citations) - Establish URL verification protocol before adding any citations. Create verification checklist. Every subsequent phase inherits this protocol.

**Sources:**
- [University of Maine Library: Link Rot Study](https://libguides.library.umaine.edu/power/day5) - 23% broken URL rate
- [Link Rot Wikipedia](https://en.wikipedia.org/wiki/Link_rot) - 14-year half-life research
- [DOI Foundation](https://www.doi.org/) - Permanent identifier system

---

### 2. Citation-Reality Mismatch (Misattribution)

**What goes wrong:**
The documentation claims a study says X, but the actual study says Y (or nothing about X). This is the most credibility-damaging error. Studies show 25-54% of citations in academic papers contain errors, and only 20% of citing authors actually read the original paper they cite.

**Why it happens:**
- Citing based on abstracts without reading full paper
- "Citation chaining" - copying citations from other papers without verification
- Misremembering or misinterpreting findings
- Overgeneralizing specific findings (e.g., "study shows X" when study shows "X in specific conditions")
- Confusing correlation with causation claims

**How to avoid:**
1. **Read the actual paper** - Download and read relevant sections, not just abstracts
2. **Quote specific findings with page numbers** - "...reduces defects by 15% (Ore et al., 2018, p. 7)"
3. **Use hedged language** - "Research suggests..." not "Research proves..."
4. **Distinguish empirical findings from author opinions** - The methodology section findings vs. discussion speculation
5. **Cross-verify claims** - If citing a claim, verify with the original source, not secondary citations
6. **Mark scope limitations** - "In their study of 12 JavaScript projects..." not "Studies show..."

**Warning signs:**
- Citing a famous work for general credibility without specific relevance
- Claim seems too strong or absolute
- No page number or section reference for specific claims
- Claim contradicts what you remember from the paper
- Multiple sources cite the same finding but none cite the original

**Phase to address:**
ALL phases - Every citation must be verified against source. Create citation verification checklist. No citation should be added without confirming the source actually supports the claim.

**Sources:**
- [Times Higher Education](https://www.timeshighereducation.com/news/quarter-citations-top-journals-wrong-or-misleading) - 25% error rate study
- [PMC Citation Errors](https://pmc.ncbi.nlm.nih.gov/articles/PMC10307651/) - Only 20% read original sources

---

### 3. Citing Retracted, Predatory, or Discredited Research

**What goes wrong:**
A citation points to a paper that has been retracted, appeared in a predatory journal, or has been widely discredited. This undermines the credibility of all citations in the documentation. Users who investigate the source find warnings about the paper's validity.

**Why it happens:**
- Not checking retraction status before citing
- Citing from secondary sources that cited before retraction
- Not recognizing predatory journal characteristics
- Assuming all published research is equally valid
- Over 15,000 predatory journals exist as of 2022
- AI tools (ChatGPT, Perplexity) recommend retracted papers without warnings

**How to avoid:**
1. **Check Retraction Watch Database** - Search papers before citing: https://retractionwatch.com/
2. **Verify journal quality:**
   - Check if indexed in Scopus, Web of Science, or PubMed
   - Use SCImago Journal Rank: https://www.scimagojr.com/
   - Avoid journals not in major indexes
3. **Prefer established venues:**
   - ACM, IEEE, Springer, Elsevier publications
   - Top-tier conferences (ICSE, FSE, ESEC, etc.)
   - Well-known preprint servers (ArXiv) for recent work
4. **Be skeptical of too-good findings** - Extraordinary claims require extraordinary evidence
5. **Check citation count and context** - Highly-cited but with many "disputed" citations is a red flag

**Warning signs:**
- Journal has no impact factor or unverifiable metrics
- Unrealistic turnaround times advertised
- Paper makes extraordinary claims without rigorous methodology
- Cannot find other papers citing this work
- Journal charges high fees but lacks prestige

**Phase to address:**
Phase 1 - Establish source quality checklist. Add Retraction Watch verification to citation process. Document acceptable publication venues.

**Sources:**
- [Retraction Watch Database](https://retractionwatch.com/) - Comprehensive retraction tracker
- [Enago Academy: Retractions and Predatory Journals](https://www.enago.com/academy/retractions-predatory-journals-crisis/) - 15,000+ predatory journals
- [scite Reference Check](https://scite.ai/blog/reference-check-an-easy-way-to-check-the-reliability-of-your-references-b2afcd64abc6) - Citation reliability tool

---

### 4. Citation Clutter Destroying Readability

**What goes wrong:**
Over-citation makes the documentation feel like an academic paper rather than practical developer documentation. Every sentence has 3-4 citations, creating visual noise that obscures the actual guidance. Developers disengage because the content feels like homework.

**Why it happens:**
- Treating every statement as needing citation
- Academic writing habits inappropriate for technical docs
- Fear of credibility attacks leading to defensive over-citation
- Copy-pasting from academic papers without adapting style
- Not distinguishing "common knowledge in the field" from novel claims

**How to avoid:**
1. **Cite claims, not facts** - "Cyclomatic complexity above 10 is high-risk" needs citation; "Go uses goroutines" does not
2. **Use consolidated citations** - Group multiple supporting studies: "(Fowler et al., 1999; Martin, 2003; Tornhill, 2015)"
3. **Target 1-3 citations per metric** - Enough for credibility, not overwhelming
4. **Place citations at end of evidence sections** - Not after every sentence
5. **Use "see also" sections for additional reading** - Separate from inline citations
6. **Distinguish primary vs. supporting citations:**
   - Primary: The key study backing your claim (inline)
   - Supporting: Additional evidence (in references section only)

**Warning signs:**
- More than 2 citations in a single sentence
- Citations appearing in "How to Improve" or threshold lists
- Every paragraph has 5+ citations
- Reader must scroll past citations to find actionable content
- Documentation reads like an academic literature review

**Phase to address:**
Phase 1 - Establish citation density guidelines (1-3 per metric). Create template showing appropriate placement. Review in Phase 7 for consistency.

**Recommended citation placement:**
```
Brief: [No citations - this is the hook]
Definition: [No citations - factual definition]
Why It Matters: [0-1 citations if needed]
Research Evidence: [1-3 citations - THIS IS WHERE CITATIONS BELONG]
Recommended Thresholds: [0-1 citations for threshold origins]
How to Improve: [No citations - actionable guidance]
```

**Sources:**
- [Wikipedia: Citation Overkill](https://en.wikipedia.org/wiki/Wikipedia:Citation_overkill) - Guidelines on citation balance
- [ResearchGate: Optimum References](https://www.researchgate.net/post/What-is-the-optimum-number-of-references-to-be-quoted-in-a-research-paper-for-its-quality-and-effectiveness) - 40-60 references for full papers
- [yomu.ai: Citation Errors](https://www.yomu.ai/blog/10-common-citation-errors-in-academic-writing) - Overcitation disrupts flow

---

## Moderate Pitfalls

These mistakes cause delays or technical debt but are recoverable.

### 5. Inconsistent Citation Formatting

**What goes wrong:**
Citations follow different formats throughout the documentation: sometimes "(Author, Year)", sometimes "[1]", sometimes "Author (Year)", sometimes full titles inline. This creates visual inconsistency and makes the documentation look unprofessional.

**Why it happens:**
- Multiple contributors using different academic backgrounds
- Copying citation formats from original sources
- No established style guide
- Mixing citation styles from different fields (CS uses IEEE/ACM; SE uses APA)
- Not updating old citations when adding new ones

**How to avoid:**
1. **Establish one citation style** - Recommend parenthetical author-year for technical docs: `(Fowler et al., 1999)`
2. **Create citation templates:**
   - Inline: `<span class="citation">(LastName et al., YEAR)</span>`
   - Multiple: `(Author1, YEAR; Author2, YEAR)`
   - Same author multiple: `(Author, YEAR1, YEAR2)`
3. **Use "et al." consistently** - For 3+ authors
4. **Standardize URL format in citations.go** - DOI preferred, then arxiv, then publisher
5. **Run consistency check before merge** - Search for citation patterns and verify uniformity

**Warning signs:**
- Same author cited differently in different metrics
- Mix of numbered [1] and author-year (Smith, 2020) styles
- Inconsistent use of "et al." vs. listing all authors
- Year sometimes inside parentheses, sometimes outside
- Different capitalization patterns

**Phase to address:**
Phase 1 - Establish citation style guide. Document in CONTRIBUTING.md or inline. All subsequent phases follow the standard.

**Existing format to maintain (from descriptions.go):**
```html
<span class="citation">(Author et al., YEAR)</span>
```

---

### 6. Citing Outdated Research as Current

**What goes wrong:**
Using a 2005 study to support claims about modern development practices, or citing pre-Git research about version control. The findings may be obsolete due to technological or methodological changes. This is especially problematic for AI-era claims using pre-AI research.

**Why it happens:**
- Not checking publication date relevance
- Over-relying on "classic" citations
- Not distinguishing foundational theory (timeless) from empirical findings (dated)
- Assuming all research is evergreen
- Not updating citations when newer, better studies exist

**How to avoid:**
1. **Distinguish citation types:**
   - **Foundational (pre-2015):** Theoretical frameworks that remain valid (McCabe, Parnas, Fowler)
   - **Empirical (within 5 years):** Findings that may change with technology
   - **AI-era (2021+):** Specific to AI agent behavior
2. **Context-date empirical claims:** "In 2017, Gao et al. found..." not "Research shows..."
3. **Prefer recent replications** - If a classic finding has been replicated recently, cite the replication
4. **Check for superseding studies** - Google Scholar "cited by" can reveal updates
5. **Balance old and new** - Aim for mix: foundational theory + recent empirical evidence

**Warning signs:**
- Empirical claims supported only by 10+ year old studies
- AI agent claims supported by pre-2021 research
- Technology-specific claims from different technology era
- No citations newer than 2020 for rapidly evolving topic
- Citing "best practices" from pre-modern tooling era

**Phase to address:**
Each category phase (1-7) - Review citation dates for appropriateness. Ensure each metric has at least one post-2020 citation for AI relevance where applicable.

**Sources:**
- [APA Style: Outdated Sources Myth](https://apastyle.apa.org/blog/outdated-sources-myth) - Older sources valid for foundational work
- [PMC: Knowledge Half-Life](https://pmc.ncbi.nlm.nih.gov/articles/PMC10231019/) - 20-year citation decay in medical literature

---

### 7. Paywalled Sources Without Alternatives

**What goes wrong:**
Citations point to research behind expensive paywalls ($30-$40 per article). Developers cannot verify claims without institutional access or payment. This creates a trust barrier where readers must accept claims on faith.

**Why it happens:**
- Academic research is predominantly paywalled
- Not checking accessibility from non-academic network
- Not providing alternative access points
- Not aware of open-access versions

**How to avoid:**
1. **Check for open-access versions:**
   - Author's personal website often has preprint
   - ArXiv, SSRN, ResearchGate preprints
   - Unpaywall browser extension
   - Semantic Scholar "Open Access" filter
2. **Prefer open-access publications** - PLOS, IEEE Open Access, ACM Open
3. **Provide multiple access points:**
   ```go
   {
       Title: "Paper Title",
       URL: "https://doi.org/xxx", // Official DOI
       PrePrintURL: "https://arxiv.org/abs/xxx", // Free version
   }
   ```
4. **Use Google Scholar** - Often links to free PDF versions
5. **Include sufficient metadata** - Title, authors, venue allows library lookup

**Warning signs:**
- URL leads directly to paywall without summary
- No ArXiv or preprint version exists
- Publisher is known for aggressive paywalling
- Paper is from pre-open-access era (pre-2000)

**Phase to address:**
ALL phases - Check accessibility of every citation. Prefer open-access versions. Add preprint links where available.

**Sources:**
- [Unpaywall](https://unpaywall.org/) - Free browser extension for legal open access
- [Ness Labs: Paywalled Research Access](https://nesslabs.com/paywalled-research-access) - Legal access strategies

---

### 8. Missing C7 Agent Evaluation Citations

**What goes wrong:**
The C7 (Agent Evaluation) category lacks citations because AI agent research is new and sparse. Leaving C7 uncited creates inconsistency with other well-cited categories, making it appear less rigorous.

**Why it happens:**
- AI agent code quality research is a nascent field
- Most relevant work is in preprints, not peer-reviewed venues
- Studies specifically measuring what ARS measures don't exist yet
- Temptation to skip citations for novel metrics

**How to avoid:**
1. **Cite adjacent research:**
   - LLM code generation studies (Codex, GitHub Copilot evaluations)
   - SWE-bench task completion studies
   - Human factors in AI-assisted development
2. **Cite foundational AI evaluation** - LLM-as-judge approaches, evaluation methodologies
3. **Acknowledge novelty explicitly** - "This metric adapts traditional evaluation approaches..."
4. **Cite your own methodology paper** - If ARS team publishes methodology, cite it
5. **Use quality preprints** - ArXiv from reputable institutions is acceptable for cutting-edge work
6. **Document the gap** - "Empirical validation of this specific metric is ongoing research"

**Warning signs:**
- C7 section has zero citations while others have 3-4
- Only citing blog posts or marketing materials
- Vague references like "recent studies show..."
- Claims about AI agent behavior with no source

**Phase to address:**
Phase 7 (C7 Agent Evaluation) - Explicitly address citation challenges. Use Borg et al. (2026) as primary evidence. Cite SWE-bench and related work.

**Relevant sources for C7:**
- Borg et al., 2026: "Code for Machines, Not Just Humans" - Direct agent break rate study
- SWE-bench (Jimenez et al., 2024) - Task completion benchmarks
- RepoGraph (Zhang et al., 2024) - Graph-based architecture metrics

---

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Citing Wikipedia instead of primary source | Easy to find, always accessible | Not authoritative, can change | Never for empirical claims; OK for definitions |
| Using blog posts as citations | Accessible, explains concepts well | Not peer-reviewed, may disappear | Never as primary source; OK as "see also" |
| Citing only abstracts without reading paper | Saves time | High misattribution risk | Never |
| Skipping DOI lookup | Faster to paste URL | Link rot in 5+ years | Only for sources without DOIs |
| One citation per category | Quick to implement | Insufficient evidence depth | Never - minimum 2-3 per category |
| Copying citations from other tools | Fast, already vetted | May not support your specific claims | Verify each citation independently |

---

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical pieces.

- [ ] **Citation Added:** Verify URL actually resolves to expected content (not just that it loads)
- [ ] **DOI Format:** Verify DOI follows `https://doi.org/10.xxxx` format (not old `dx.doi.org`)
- [ ] **Author Format:** Verify "et al." used consistently for 3+ authors
- [ ] **Year Accuracy:** Verify cited year matches actual publication year (not ArXiv upload date)
- [ ] **Claim Match:** Re-read cited paper section to confirm it supports the specific claim
- [ ] **Open Access:** Verify at least one free-access path exists (preprint, author site)
- [ ] **citations.go Entry:** Verify citation is added to both inline text AND citations.go reference list
- [ ] **HTML Rendering:** Verify citation displays correctly in HTML output (check class="citation")
- [ ] **No Retraction:** Check Retraction Watch for each paper
- [ ] **Consistent Style:** Verify citation matches established format

---

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Broken URL discovered | LOW | Find DOI or archive.org version; update URL; add note if source is truly lost |
| Misattribution discovered | MEDIUM | Read original source; rewrite claim to match evidence; may need to remove claim entirely |
| Retraction discovered | MEDIUM | Remove citation; find alternative source; review if claim is still supported |
| Over-citation complaints | LOW | Consolidate citations; move supporting refs to "see also"; keep only primary sources inline |
| Style inconsistency | LOW | Run search-and-replace for patterns; update all to match standard |
| Paywalled source complaints | LOW | Find and add preprint/open-access link |

---

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls.

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Broken URLs (#1) | Phase 1 (establish protocol) | URL check script runs before each merge |
| Misattribution (#2) | ALL phases | Claim-source matching in PR review checklist |
| Retracted sources (#3) | Phase 1 (establish protocol) | Retraction Watch check for each citation |
| Citation clutter (#4) | Phase 1 (establish guidelines) | Citation count per metric review |
| Format inconsistency (#5) | Phase 1 (establish style) | Linting/grep for citation patterns |
| Outdated research (#6) | ALL phases | Date balance check per category |
| Paywalled sources (#7) | ALL phases | Accessibility check from non-academic IP |
| Missing C7 citations (#8) | Phase 7 | Explicit acknowledgment of novelty |

---

## Phase-Specific Risk Summary

| Phase | Category | Highest Risk Pitfall | Mitigation |
|-------|----------|---------------------|------------|
| Phase 1 | C1 Code Quality | Format inconsistency (#5) | Establish style guide; existing citations set precedent |
| Phase 2 | C2 Semantics | Citation clutter (#4) | Type annotation research is abundant; resist over-citing |
| Phase 3 | C3 Architecture | Outdated research (#6) | Parnas (1972) is foundational but add modern replication |
| Phase 4 | C4 Documentation | Paywalled sources (#7) | Documentation research often in ACM/IEEE paywalls |
| Phase 5 | C5 Temporal | Broken URLs (#1) | Tornhill's book may move; use ISBN backup |
| Phase 6 | C6 Testing | Citation clutter (#4) | TDD literature is extensive; prioritize seminal works |
| Phase 7 | C7 Agent | Missing citations (#8) | Nascent field; use preprints, acknowledge gaps |

---

## Verification Checklist (Per Phase)

Before completing each phase, verify:

### Setup (Phase 1)
- [ ] Citation style guide documented
- [ ] URL verification script working
- [ ] Retraction Watch check process documented
- [ ] Citation template established (`<span class="citation">`)
- [ ] citations.go structure supports all needed fields

### Each Category Phase (1-7)
- [ ] All URLs verified accessible
- [ ] All claims verified against source text
- [ ] Citation count per metric is 1-3 (not excessive)
- [ ] Mix of foundational + recent citations where appropriate
- [ ] At least one open-access path per citation
- [ ] citations.go updated with all new references
- [ ] HTML rendering verified for all new citations
- [ ] Consistent format with existing citations

### Final Review (After Phase 7)
- [ ] Cross-category citation consistency
- [ ] No duplicate citations with inconsistent metadata
- [ ] All categories have appropriate citation density
- [ ] README or CONTRIBUTING documents citation standards
- [ ] Total citation count reasonable (target ~50-60 total)

---

## Sources

### Academic Citation Research
- [PMC: Citation Errors in Research](https://pmc.ncbi.nlm.nih.gov/articles/PMC10307651/) - Comprehensive study of citation error types
- [Times Higher Education: Quarter of Citations Wrong](https://www.timeshighereducation.com/news/quarter-citations-top-journals-wrong-or-misleading) - 25% error rate finding
- [Zenodo: Software Citation Pitfalls](https://zenodo.org/records/4263762) - Software-specific citation issues

### Link Rot and Persistence
- [University of Maine: Link Rot](https://libguides.library.umaine.edu/power/day5) - Link decay statistics
- [Harvard Law Review: Perma.cc](https://harvardlawreview.org/forum/vol-127/perma-scoping-and-addressing-the-problem-of-link-and-reference-rot-in-legal-citations/) - Archive solution
- [Crossref: URLs and DOIs](https://www.crossref.org/blog/urls-and-dois-a-complicated-relationship/) - DOI persistence

### Source Quality
- [Retraction Watch](https://retractionwatch.com/) - Retraction database
- [Enago: Predatory Journals](https://www.enago.com/academy/retractions-predatory-journals-crisis/) - Journal quality issues
- [SCImago Journal Rank](https://www.scimagojr.com/) - Journal quality metrics

### Citation Best Practices
- [Wikipedia: Citation Overkill](https://en.wikipedia.org/wiki/Wikipedia:Citation_overkill) - Over-citation guidelines
- [APA Style: Outdated Sources](https://apastyle.apa.org/blog/outdated-sources-myth) - When old sources are appropriate
- [Unpaywall](https://unpaywall.org/) - Open access finder

---
*Pitfalls research for: Academic citations in technical documentation*
*Researched: 2026-02-04*
*Confidence: HIGH - Verified against citation error studies, library science guides, and existing ARS codebase patterns*
