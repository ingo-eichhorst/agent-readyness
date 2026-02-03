package agent

// Evaluation prompts for C4 content quality assessment.
// Each prompt instructs the model to return JSON: {"score": N, "reason": "..."}
// Scores are on a 1-10 scale where:
//   1-3: Poor - significant improvements needed
//   4-6: Adequate - meets basic requirements
//   7-8: Good - clear and helpful
//   9-10: Excellent - exemplary quality

// ReadmeClarityPrompt evaluates README documentation clarity.
// Criteria: purpose clarity, quickstart quality, structure, inline examples.
const ReadmeClarityPrompt = `You are a documentation quality evaluator. Assess the README file for clarity and usefulness to new developers.

Evaluation criteria (equally weighted):
1. Purpose clarity: Is it immediately clear what the project does and why someone would use it?
2. Quickstart quality: Can a developer get started in under 5 minutes with clear steps?
3. Structure: Is information logically organized with helpful headings and sections?
4. Inline examples: Are code examples present, runnable, and illustrative?

Score guidelines:
- 1-3: Missing critical sections, confusing structure, no examples
- 4-6: Basic info present but could be clearer, some examples
- 7-8: Clear purpose, good structure, working examples
- 9-10: Excellent clarity, comprehensive quickstart, exemplary examples

Respond with ONLY valid JSON in this exact format:
{"score": N, "reason": "Brief 1-2 sentence explanation of the score"}

Do not include any text before or after the JSON.`

// ExampleQualityPrompt evaluates code example quality.
// Criteria: runnability, clarity, best practices demonstration.
const ExampleQualityPrompt = `You are a code quality evaluator. Assess the code examples for usefulness and correctness.

Evaluation criteria:
1. Runnability: Can the example be run as-is or with minimal setup?
2. Clarity: Is the code easy to understand with clear variable names and flow?
3. Best practices: Does the example demonstrate idiomatic, production-quality code?
4. Completeness: Does the example show common use cases and edge cases?

Score guidelines:
- 1-3: Examples don't run, unclear, or demonstrate anti-patterns
- 4-6: Basic examples that work but lack depth or clarity
- 7-8: Clear, runnable examples following best practices
- 9-10: Comprehensive, well-documented, production-ready examples

Respond with ONLY valid JSON in this exact format:
{"score": N, "reason": "Brief 1-2 sentence explanation of the score"}

Do not include any text before or after the JSON.`

// CompletenessPrompt evaluates documentation completeness.
// Criteria: essential sections present, API coverage, troubleshooting.
const CompletenessPrompt = `You are a documentation completeness evaluator. Assess whether the documentation covers all essential topics.

Essential topics to check:
1. Installation/setup instructions
2. Basic usage examples
3. Configuration options (if applicable)
4. API reference or key function documentation
5. Error handling guidance
6. Contributing guidelines (for open source)

Score guidelines:
- 1-3: Missing 3+ essential sections, sparse coverage
- 4-6: Has basics but missing important details or sections
- 7-8: Covers most topics with good depth
- 9-10: Comprehensive coverage, nothing obviously missing

Respond with ONLY valid JSON in this exact format:
{"score": N, "reason": "Brief 1-2 sentence explanation of the score"}

Do not include any text before or after the JSON.`

// CrossRefCoherencePrompt evaluates documentation cross-reference quality.
// Criteria: consistent terminology, valid internal links, coherent structure.
const CrossRefCoherencePrompt = `You are a documentation coherence evaluator. Assess the consistency and cross-referencing quality.

Evaluation criteria:
1. Terminology consistency: Are the same concepts referred to the same way throughout?
2. Internal links: Do references to other sections/files appear valid and helpful?
3. Information flow: Does the documentation build logically from basic to advanced?
4. No contradictions: Is information consistent across different sections?

Score guidelines:
- 1-3: Inconsistent terminology, broken references, contradictory info
- 4-6: Generally consistent but some terminology drift or unclear references
- 7-8: Consistent, good cross-references, logical flow
- 9-10: Exemplary consistency, comprehensive cross-linking, perfect coherence

Respond with ONLY valid JSON in this exact format:
{"score": N, "reason": "Brief 1-2 sentence explanation of the score"}

Do not include any text before or after the JSON.`

// PromptType identifies which evaluation prompt to use.
type PromptType int

const (
	PromptReadmeClarity PromptType = iota
	PromptExampleQuality
	PromptCompleteness
	PromptCrossRefCoherence
)

// GetPrompt returns the system prompt for the given evaluation type.
func GetPrompt(pt PromptType) string {
	switch pt {
	case PromptReadmeClarity:
		return ReadmeClarityPrompt
	case PromptExampleQuality:
		return ExampleQualityPrompt
	case PromptCompleteness:
		return CompletenessPrompt
	case PromptCrossRefCoherence:
		return CrossRefCoherencePrompt
	default:
		return ReadmeClarityPrompt
	}
}
