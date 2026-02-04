package output

import "html/template"

// MetricDescription contains brief and detailed descriptions for a metric.
type MetricDescription struct {
	Brief     string        // 1-2 sentences, always visible when expanded
	Detailed  template.HTML // Full HTML content for expanded section
	Threshold float64       // Score below which to auto-expand (typically 6.0)
}

// metricDescriptions maps metric names to their descriptions.
// These are used in HTML reports to help users understand what each metric measures,
// why it matters for AI agents, and how to improve.
var metricDescriptions = map[string]MetricDescription{
	// ============================================================================
	// C1: Code Health Metrics
	// ============================================================================
	"complexity_avg": {
		Brief:     "Average cyclomatic complexity per function. High complexity increases AI agent break rates by 36-44%. Keep under 10.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Cyclomatic complexity counts the number of independent paths through a function's control flow graph. Each decision point (if, for, while, case, &&, ||) adds one to the count. A function with no branches has complexity 1.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Agents must mentally simulate all possible execution paths when reasoning about code changes. High complexity causes <strong>"state drift"</strong>—LLMs lose track of variable states and scope as they traverse deeply nested conditionals. The <strong>"Bumpy Road"</strong> pattern (multiple sequential nested conditionals) is the single most detrimental property for agent reliability.</p>
<p>Attention mechanisms in LLMs scale O(n²) with sequence length, making long complex functions disproportionately harder to reason about. Weaker models are especially sensitive: the same code health issues that cause a 36% increase in break rates for Claude cause 44% more failures for less capable models.</p>

<h4>Research Evidence</h4>
<p>Empirical research quantifies the impact of code complexity on AI agent performance <span class="citation">(Borg et al., 2026)</span>:</p>
<table class="evidence-table">
<tr><th>Model</th><th>Healthy Code</th><th>Unhealthy Code</th><th>Increase</th></tr>
<tr><td>Claude</td><td>3.81%</td><td>5.19%</td><td>+36%</td></tr>
<tr><td>Qwen</td><td>19.28%</td><td>27.84%</td><td>+44%</td></tr>
<tr><td>GPT</td><td>35.87%</td><td>47.02%</td><td>+31%</td></tr>
</table>
<p>The study identifies a maximum nesting depth threshold of <strong>4 levels</strong>—the "pyramid of doom"—beyond which agent reliability drops sharply. McCabe's foundational work established complexity above 10 as high-risk <span class="citation">(McCabe, 1976)</span>, and Fowler identified high-complexity functions as primary refactoring targets <span class="citation">(Fowler et al., 1999)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>1-5:</strong> Simple, easy for agents to reason about</li>
<li><strong>6-10:</strong> Moderate, agents can handle with care</li>
<li><strong>11-20:</strong> Complex, expect 30-40% higher agent break rates</li>
<li><strong>21+:</strong> Very high risk, refactor before agent use</li>
</ul>
<p><em>Note: Keep nesting depth ≤4 levels regardless of overall complexity score.</em></p>

<h4>How to Improve</h4>
<ul>
<li>Replace nested conditionals with guard clauses (early returns)—this "resets the context window" for agents</li>
<li>Extract conditional logic into well-named helper functions</li>
<li>Add nesting depth linting (e.g., <code>max-depth</code> ESLint rule, <code>nestif</code> for Go)</li>
<li>Prioritize "Bumpy Road" functions—those with multiple sequential nested blocks</li>
<li>Use polymorphism or strategy pattern instead of switch statements</li>
</ul>`,
	},

	"func_length_avg": {
		Brief:     "Average lines per function. Shorter functions (under 25 lines) are easier for agents to understand atomically <span class=\"citation\">(Chowdhury et al., 2022)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Measures the average number of lines of code per function across the codebase. Includes all executable statements, comments within functions, and blank lines within function bodies.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Agents process code within context windows with limited capacity. Long functions consume more context, leaving less room for related code, documentation, and reasoning. Shorter functions allow agents to see complete units of behavior, understand purpose quickly, and make targeted modifications.</p>

<h4>Research Evidence</h4>
<p>Fowler identified "Long Method" as a primary code smell, recommending functions be short enough to understand at a glance <span class="citation">(Fowler et al., 1999)</span>. Empirical research on Java methods found that functions under 24 SLOC have significantly lower maintenance burden and defect rates <span class="citation">(Chowdhury et al., 2022)</span>.</p>
<p>Recent studies on AI agents confirm that function length directly impacts agent performance. Agents working with unhealthy code (including long functions) experience 36-44% higher break rates depending on model capability <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>1-15:</strong> Excellent, functions fit easily in agent context</li>
<li><strong>16-25:</strong> Good, still comprehensible as units</li>
<li><strong>26-50:</strong> Moderate, consider splitting</li>
<li><strong>51+:</strong> Long, likely doing too much</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Apply Extract Method refactoring to isolate logical sections</li>
<li>Move setup/teardown into separate functions</li>
<li>Use composition over inline implementation</li>
<li>Ensure each function has a single, clear purpose</li>
</ul>`,
	},

	"file_size_avg": {
		Brief:     "Average lines per file. Smaller files (under 300 lines) help agents navigate and understand module scope <span class=\"citation\">(Parnas, 1972)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>The average number of lines per source file in the codebase, including code, comments, and blank lines. Measures overall file organization and module granularity.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Large files often indicate poor separation of concerns, making it harder for agents to locate relevant code and understand module boundaries. When agents need to modify code in large files, they risk unintended side effects due to hidden dependencies between distant sections.</p>

<h4>Research Evidence</h4>
<p>Parnas's foundational work on information hiding established that well-decomposed modules with clear boundaries are essential for maintainability <span class="citation">(Parnas, 1972)</span>. Design patterns literature reinforces this principle, emphasizing cohesion: code that changes together should live together, but in manageable units <span class="citation">(Gamma et al., 1994)</span>.</p>
<p>AI agent research confirms these principles apply to automated code modification. Agents experience significantly higher break rates (36-44%) when working with large, poorly-structured files compared to well-decomposed codebases <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>1-200:</strong> Well-organized, focused modules</li>
<li><strong>201-400:</strong> Acceptable, may benefit from splitting</li>
<li><strong>401-800:</strong> Large, review module boundaries</li>
<li><strong>801+:</strong> Very large, likely multiple responsibilities</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Split files by logical domain or feature</li>
<li>Extract utility functions into dedicated modules</li>
<li>Move related types and constants into focused files</li>
<li>Consider one public type/class per file as a guideline</li>
</ul>`,
	},

	"afferent_coupling_avg": {
		Brief:     "Incoming dependencies per module (Ca). Lower coupling means modules can be modified more safely <span class=\"citation\">(Martin, 2003)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Afferent coupling (Ca) counts how many other modules depend on a given module. High afferent coupling means the module is heavily used throughout the codebase, making changes to it potentially far-reaching.</p>

<h4>Why It Matters for AI Agents</h4>
<p>When agents modify highly-coupled modules, changes ripple to all dependents. Agents must understand and account for all usages, which may exceed context window capacity. Lower coupling allows agents to make confident, isolated changes.</p>

<h4>Research Evidence</h4>
<p>Parnas established that information hiding and module interfaces are fundamental to maintainability; modules with many incoming dependencies become change-resistant <span class="citation">(Parnas, 1972)</span>. Martin formalized the afferent coupling metric (Ca) as part of the Stable Dependencies Principle: modules with high Ca should be stable since changes affect many dependents <span class="citation">(Martin, 2003)</span>.</p>
<p>Empirical research on AI agents shows that highly-coupled code significantly increases agent break rates. Claude experiences 36% more failures and Qwen 44% more failures when working with unhealthy (highly-coupled) code <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>0-3:</strong> Low coupling, safe to modify</li>
<li><strong>4-7:</strong> Moderate, changes need careful review</li>
<li><strong>8-15:</strong> High, consider if module is doing too much</li>
<li><strong>16+:</strong> Very high, likely a utility or core module needing special care</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Introduce interfaces to decouple implementations</li>
<li>Use dependency injection instead of direct imports</li>
<li>Extract shared functionality into well-defined API contracts</li>
<li>Consider whether widely-used code should be in a stable core library</li>
</ul>`,
	},

	"efferent_coupling_avg": {
		Brief:     "Outgoing dependencies per module (Ce). Modules depending on too many others become fragile <span class=\"citation\">(Martin, 2003)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Efferent coupling (Ce) counts how many other modules a given module depends on. High efferent coupling means the module relies on many external components, making it vulnerable to changes elsewhere.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Modules with high efferent coupling require agents to understand many dependencies before making changes. This increases cognitive load and the risk of missing interactions. Agents work best with self-contained modules having minimal external dependencies.</p>

<h4>Research Evidence</h4>
<p>Martin's Stable Dependencies Principle states that modules should depend only on modules more stable than themselves; high efferent coupling (Ce) indicates a module is vulnerable to ripple effects from its many dependencies <span class="citation">(Martin, 2003)</span>. This principle builds on foundational work showing that explicit, minimal interfaces are essential for maintainability <span class="citation">(Parnas, 1972)</span>.</p>
<p>AI agent studies confirm these principles: agents working with highly-coupled code experience 36-44% higher break rates, as understanding dependency chains exceeds context window capacity <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>0-5:</strong> Well-isolated, focused module</li>
<li><strong>6-10:</strong> Moderate, review if all dependencies are necessary</li>
<li><strong>11-20:</strong> High, likely an orchestration or integration point</li>
<li><strong>21+:</strong> Very high, consider decomposition</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Apply Facade pattern to simplify dependency interfaces</li>
<li>Use dependency injection to make dependencies explicit</li>
<li>Split modules that import too many unrelated packages</li>
<li>Consider if functionality should be closer to its dependencies</li>
</ul>`,
	},

	"duplication_rate": {
		Brief:     "Percentage of duplicated code. Less duplication means fewer places to update when agents make changes <span class=\"citation\">(Fowler et al., 1999)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>The percentage of code that appears multiple times in the codebase, typically measured as duplicate sequences of 6+ lines or tokens. Includes exact duplicates and near-duplicates with minor variations.</p>

<h4>Why It Matters for AI Agents</h4>
<p>When agents identify a bug or make an improvement, duplicated code requires the same change in multiple locations. Agents may miss some instances, leading to inconsistent behavior. Additionally, duplicates consume context window space without adding new information.</p>

<h4>Research Evidence</h4>
<p>Fowler identified "Duplicated Code" as a fundamental code smell, noting that identical or similar code sequences indicate missing abstractions and create maintenance burden <span class="citation">(Fowler et al., 1999)</span>. The DRY (Don't Repeat Yourself) principle follows directly: every piece of knowledge should have a single, unambiguous representation.</p>
<p>AI agent research confirms that duplication significantly impacts automated code modification. When agents must propagate changes across duplicated code, break rates increase substantially—studies show 36-44% higher failure rates on unhealthy codebases <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>0-3%:</strong> Minimal duplication, excellent</li>
<li><strong>4-7%:</strong> Low, some duplication is natural</li>
<li><strong>8-15%:</strong> Moderate, review largest duplicates</li>
<li><strong>16%+:</strong> High, significant refactoring opportunity</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Extract duplicated code into shared functions or methods</li>
<li>Use parameterization instead of copy-paste with modifications</li>
<li>Create utility libraries for common patterns</li>
<li>Apply DRY (Don't Repeat Yourself) principle during code review</li>
</ul>`,
	},

	// ============================================================================
	// C2: Semantic Explicitness Metrics
	// ============================================================================
	"type_annotation_coverage": {
		Brief:     "Percentage of values with explicit type annotations. Type annotations catch 15% of bugs <span class=\"citation\">(Gao et al., 2017)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>The percentage of function parameters, return values, and variables that have explicit type annotations. In Go, this is inherent; in TypeScript and Python, it measures type hint usage.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Type annotations serve as machine-readable documentation of programmer intent. Agents use types to understand what data flows through the system, validate their changes are type-safe, and navigate codebases efficiently. Without types, agents must infer intent from usage patterns, which is error-prone.</p>

<h4>Research Evidence</h4>
<p>Pierce's foundational work established that type systems ensure "well-typed programs do not go wrong"—they prevent entire categories of runtime errors <span class="citation">(Pierce, 2002)</span>. Empirical studies confirm these theoretical benefits: TypeScript and Flow detect approximately 15% of bugs that would otherwise reach production <span class="citation">(Gao et al., 2017)</span>.</p>
<p>Industry adoption validates these findings. A 2024 Meta survey found that 88% of Python developers consistently use type hints, with 49.8% citing bug prevention as a primary benefit <span class="citation">(Meta, 2024)</span>. For AI agents, type annotations are especially valuable: type-constrained decoding reduces LLM compilation errors by 52% in code generation tasks. Code health metrics including type coverage predict AI agent reliability <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>90-100%:</strong> Fully typed, excellent for agents</li>
<li><strong>70-89%:</strong> Good coverage, some gaps</li>
<li><strong>50-69%:</strong> Partial typing, agents may struggle</li>
<li><strong>0-49%:</strong> Minimal typing, high agent error risk</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Enable strict mode in TypeScript or mypy --strict in Python</li>
<li>Add return type annotations to all public functions</li>
<li>Annotate function parameters, especially those accepting multiple types</li>
<li>Use generics instead of any/object types</li>
</ul>`,
	},

	"naming_consistency": {
		Brief:     "Adherence to naming conventions. Flawed identifiers correlate with low-quality code <span class=\"citation\">(Butler et al., 2009)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Measures how consistently the codebase follows naming conventions: camelCase for functions and variables, PascalCase for types and classes, UPPER_SNAKE_CASE for constants. Also checks for descriptive names over abbreviations.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Agents learn patterns from training data that follow common conventions. Inconsistent naming breaks these patterns, causing agents to generate code that clashes with local style. Consistent naming also helps agents infer purpose from names and maintain coherent code generation.</p>

<h4>Research Evidence</h4>
<p>Butler et al. conducted empirical studies correlating identifier naming quality with code quality. Their initial work found that flawed identifiers (poor grammar, single letters, abbreviations) correlate with lower-quality code as measured by static analysis tools <span class="citation">(Butler et al., 2009)</span>. A follow-up study extended these findings to method identifiers, confirming that consistent, descriptive naming associates with higher code quality <span class="citation">(Butler et al., 2010)</span>.</p>
<p>Note: These studies focused on Java codebases; the naming conventions differ across languages, but the principle that naming quality correlates with code quality appears language-agnostic. Well-structured code with clear naming improves AI agent comprehension and reliability <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>90-100%:</strong> Highly consistent, agents can follow patterns</li>
<li><strong>75-89%:</strong> Good, minor inconsistencies</li>
<li><strong>50-74%:</strong> Mixed conventions, may confuse agents</li>
<li><strong>0-49%:</strong> Inconsistent, high risk of style clashes</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Adopt and enforce a style guide (Google, Airbnb, etc.)</li>
<li>Configure linters to check naming conventions</li>
<li>Use IDE refactoring tools to rename inconsistent identifiers</li>
<li>Prefer descriptive names over abbreviations (userCount vs uc)</li>
</ul>`,
	},

	"magic_number_ratio": {
		Brief:     "Unexplained numeric literals per 1,000 lines. Magic Number is a classic code smell <span class=\"citation\">(Fowler et al., 1999)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Counts numeric literals in code (excluding 0, 1, and common values) that are not defined as named constants. Reported as occurrences per 1,000 lines of code. Magic numbers are unexplained values embedded directly in logic.</p>

<h4>Why It Matters for AI Agents</h4>
<p>When agents encounter magic numbers, they cannot determine the value's purpose or whether it can be safely changed. Named constants like MAX_RETRIES = 3 communicate intent; the literal 3 does not. Agents may incorrectly reuse or modify magic numbers without understanding their significance.</p>

<h4>Research Evidence</h4>
<p>Fowler identified "Magic Number" as a canonical code smell, recommending replacement with named constants that communicate intent <span class="citation">(Fowler et al., 1999)</span>. From a type-theoretic perspective, named constants with appropriate types help prevent category errors—using a timeout value where a retry count is expected <span class="citation">(Pierce, 2002)</span>.</p>
<p>Semantic clarity, including meaningful constant names, aids AI agent reasoning. Agents working with well-structured code experience significantly lower break rates <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>0-2:</strong> Few magic numbers, well-documented</li>
<li><strong>3-5:</strong> Low, some explanations may be needed</li>
<li><strong>6-10:</strong> Moderate, review critical values</li>
<li><strong>11+:</strong> High, significant documentation debt</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Replace literals with named constants that describe purpose</li>
<li>Group related constants in configuration files or modules</li>
<li>Add comments when context-specific values must be inline</li>
<li>Use enums for related sets of values</li>
</ul>`,
	},

	"type_strictness": {
		Brief:     "Use of strict type checking features. Type systems rule out untrapped errors <span class=\"citation\">(Cardelli, 1996)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Measures whether the codebase uses strict type checking features: strict mode in TypeScript, strict mypy settings in Python, or equivalent. A binary metric (enabled or not) that significantly impacts type safety.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Strict type checking catches type errors at compile time rather than runtime. When agents generate code, strict checking provides immediate feedback on type mismatches, allowing agents to self-correct. Without strict mode, type errors may only surface during execution.</p>

<h4>Research Evidence</h4>
<p>Cardelli's foundational work defines type safety as ruling out "untrapped errors"—runtime failures that can corrupt program state without immediate detection <span class="citation">(Cardelli, 1996)</span>. Wright and Felleisen formalized this through the progress and preservation theorems: well-typed programs either evaluate to a value or continue evaluating (progress), and evaluation preserves types (preservation) <span class="citation">(Wright & Felleisen, 1994)</span>.</p>
<p>Empirical validation confirms these theoretical benefits. Gao et al. found that TypeScript and Flow detect 15% of bugs that would escape untyped JavaScript—this detection requires strict mode to achieve full coverage <span class="citation">(Gao et al., 2017)</span>. For AI agents, type constraints provide immediate feedback on generated code correctness, reducing LLM compilation errors by 52% in code generation tasks.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>Enabled:</strong> Strong type safety, recommended</li>
<li><strong>Disabled:</strong> Looser checking, higher risk of type errors</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Enable "strict": true in tsconfig.json</li>
<li>Enable all mypy strict flags in Python</li>
<li>Fix existing type errors before enabling strict mode</li>
<li>Use strict mode from project start for new codebases</li>
</ul>`,
	},

	"null_safety": {
		Brief:     "Handling of null/undefined values. Null references were called a \"billion-dollar mistake\" <span class=\"citation\">(Hoare, 2009)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Measures usage of null-safe patterns: optional chaining (?.), nullish coalescing (??), null assertions, and proper Optional/Maybe types. Also detects unsafe patterns like unchecked null dereferences.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Null reference errors are among the most common runtime failures. When agents generate code, they may not anticipate all cases where values could be null. Explicit null handling in the codebase establishes patterns that agents can follow, reducing null-related bugs in generated code.</p>

<h4>Research Evidence</h4>
<p>Tony Hoare, inventor of the null reference, called it his "billion-dollar mistake" in a 2009 presentation <span class="citation">(Hoare, 2009)</span>. Note: This is a practitioner acknowledgment, not peer-reviewed research, but it carries weight as a reflection from the language designer who introduced the concept.</p>
<p>Type theory provides the formal solution: Optional/Maybe types make nullability explicit in the type system, ensuring that potentially-absent values must be handled before use <span class="citation">(Pierce, 2002)</span>. Empirical research confirms that type annotations, including null-related annotations, help catch bugs that would otherwise reach production <span class="citation">(Gao et al., 2017)</span>. Languages like Kotlin demonstrate industry validation—language-level null safety largely eliminates the NullPointerException class of bugs.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>90-100%:</strong> Excellent null safety practices</li>
<li><strong>70-89%:</strong> Good coverage with some gaps</li>
<li><strong>50-69%:</strong> Partial, null errors likely</li>
<li><strong>0-49%:</strong> Poor, high null error risk</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Enable strictNullChecks in TypeScript</li>
<li>Use Optional types instead of nullable types where appropriate</li>
<li>Apply optional chaining (?.) and nullish coalescing (??)</li>
<li>Validate input at API boundaries</li>
</ul>`,
	},

	// ============================================================================
	// C3: Architecture Metrics
	// ============================================================================
	"max_dir_depth": {
		Brief:     "Deepest directory nesting level. Clear module boundaries and shallow hierarchies improve comprehensibility <span class=\"citation\">(Parnas, 1972)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>The maximum depth of directory nesting in the source tree, counting from the project root. Measures how deeply files are organized into subdirectories (e.g., src/api/v2/handlers/auth/utils.go = depth 6).</p>

<h4>Why It Matters for AI Agents</h4>
<p>Deep directory hierarchies make it harder for agents to locate related code and understand project organization. Long import paths consume context space and are prone to errors. Shallower structures provide clearer boundaries and easier navigation.</p>

<h4>Research Evidence</h4>
<p>Parnas's foundational work on module decomposition established that well-structured systems with clear boundaries are fundamentally easier to understand and maintain <span class="citation">(Parnas, 1972)</span>. The principle of information hiding means each module should encapsulate design decisions, with directory structure reflecting logical module boundaries.</p>
<p>Empirical studies using design structure matrices confirm that modular architectures with clear boundaries have measurable quality benefits. MacCormack et al. analyzed open-source and proprietary systems, finding that well-decomposed architectures enable independent component evolution <span class="citation">(MacCormack et al., 2006)</span>.</p>
<p>For AI agents, structural clarity is essential: agents working with well-organized code experience significantly lower break rates. Code health metrics including organizational structure predict agent reliability <span class="citation">(Borg et al., 2026)</span>.</p>

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

	"module_fanout_avg": {
		Brief:     "Average imports per module. High coupling is detrimental to modular design <span class=\"citation\">(Stevens et al., 1974)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>The average number of imports per source file. Counts both internal and external dependencies, measuring how widely each module reaches into the rest of the codebase or ecosystem.</p>

<h4>Why It Matters for AI Agents</h4>
<p>High fanout means agents must understand many dependencies to reason about a single file. Each import brings potential side effects and API contracts into scope. Lower fanout creates more self-contained modules that agents can modify with confidence.</p>

<h4>Research Evidence</h4>
<p>Stevens, Myers, and Constantine's foundational work on structured design established that coupling and cohesion are primary determinants of software quality <span class="citation">(Stevens et al., 1974)</span>. Low coupling between modules—measured by import count—improves maintainability and reduces change propagation.</p>
<p>Chidamber and Kemerer formalized this with the Coupling Between Objects (CBO) metric, demonstrating that excessive coupling is detrimental to modular design, prevents reuse, and increases testing complexity <span class="citation">(Chidamber & Kemerer, 1994)</span>.</p>
<p>The Stable Dependencies Principle advises that modules should depend only on modules more stable than themselves <span class="citation">(Martin, 2003)</span>. Note: This is an influential practitioner perspective widely adopted in industry.</p>
<p>For AI agents, highly-coupled code significantly increases break rates. Agents experience 36-44% higher failure rates when working with unhealthy (highly-coupled) code <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>1-5:</strong> Low fanout, focused modules</li>
<li><strong>6-10:</strong> Moderate, typical for feature modules</li>
<li><strong>11-15:</strong> High, review if all imports are needed</li>
<li><strong>16+:</strong> Very high, consider decomposition</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Remove unused imports</li>
<li>Extract code that requires different dependencies into separate modules</li>
<li>Use dependency injection to reduce direct coupling</li>
<li>Create focused interfaces instead of importing full modules</li>
</ul>`,
	},

	"circular_deps": {
		Brief:     "Number of circular dependencies. Acyclic dependency structures are easier to understand, test, and maintain <span class=\"citation\">(Lakos, 1996)</span>.",
		Threshold: 7.0,
		Detailed: `<h4>Definition</h4>
<p>Counts the number of circular dependency chains where module A imports B which imports A (directly or transitively). Circular dependencies create ordering problems and make it impossible to understand modules in isolation.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Circular dependencies mean agents cannot understand one module without understanding all modules in the cycle. This creates reasoning complexity that scales with cycle size. Breaking cycles allows agents to analyze and modify modules independently.</p>

<h4>Research Evidence</h4>
<p>Parnas established that modular systems should have clear dependency direction—each module's design decisions should be hidden from others <span class="citation">(Parnas, 1972)</span>. Circular dependencies violate this principle by creating mutual knowledge requirements.</p>
<p>The Acyclic Dependencies Principle states that the dependency graph of packages should have no cycles <span class="citation">(Martin, 2003)</span>. Note: This represents an influential practitioner perspective widely adopted in industry, though not derived from empirical research.</p>
<p>Lakos demonstrated practical techniques for eliminating cyclic dependencies in large systems, showing that acyclic physical dependencies dramatically reduce link-time costs and improve testability <span class="citation">(Lakos, 1996)</span>.</p>
<p>Empirical research on 31 open-source Java systems found that circular dependencies correlate with higher change frequency in affected classes <span class="citation">(Oyetoyan et al., 2015)</span>. This supports the principle that cycles create maintenance burden.</p>
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

	"import_complexity_avg": {
		Brief:     "Average complexity of import statements. Dependency structure impacts maintainability <span class=\"citation\">(Sangal et al., 2005)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Measures the complexity of import patterns: deep submodule imports, aliased imports, re-exports, and barrel files. Higher scores indicate more complex import structures that are harder to trace and understand.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Complex import patterns obscure where code actually lives. Agents may struggle to locate the true source of an import, especially with re-exports and barrel files. Simpler imports create clearer dependency graphs that agents can navigate.</p>

<h4>Research Evidence</h4>
<p>Parnas established that clear module boundaries with explicit interfaces improve comprehension and enable independent development <span class="citation">(Parnas, 1972)</span>. Complex import patterns violate this principle by obscuring true dependencies.</p>
<p>Sangal et al. developed the Design Structure Matrix (DSM) approach for managing complex software dependencies, demonstrating that simpler, well-organized dependency structures enable clearer architectural reasoning <span class="citation">(Sangal et al., 2005)</span>.</p>
<p>Recent empirical work on the M-score metric found that dependency density correlates with project maintainability—projects with simpler, sparser dependency graphs are easier to maintain <span class="citation">(Pisch et al., 2024)</span>.</p>
<p>For AI agents, structural complexity impacts comprehension: agents working with well-organized code experience significantly lower break rates <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>1-2:</strong> Simple, direct imports</li>
<li><strong>3-4:</strong> Moderate complexity</li>
<li><strong>5-7:</strong> High, consider simplification</li>
<li><strong>8+:</strong> Very complex, significant refactoring opportunity</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Prefer direct imports over barrel file re-exports</li>
<li>Limit import depth (import from package, not deep submodules)</li>
<li>Use consistent import aliasing conventions</li>
<li>Organize exports at clear API boundaries</li>
</ul>`,
	},

	"dead_exports": {
		Brief:     "Exported symbols not used elsewhere. Dead Code is an established code smell that harms comprehensibility <span class=\"citation\">(Fowler et al., 1999)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Counts exported/public symbols (functions, types, constants) that are never imported or used outside their defining module. These create noise in the public API without providing value.</p>

<h4>Why It Matters for AI Agents</h4>
<p>When agents explore a module's API, dead exports appear as valid options but lead to confusion when used. Agents may incorrectly incorporate unused functionality or spend context window space understanding code that serves no purpose.</p>

<h4>Research Evidence</h4>
<p>Fowler identified Dead Code as a canonical code smell, noting that unused code increases cognitive load and should be removed <span class="citation">(Fowler et al., 1999)</span>. Dead exports are a specific form of dead code that pollutes the public API surface.</p>
<p>Romano et al. conducted a multi-study investigation into dead code across multiple systems, finding that dead code harms comprehensibility and maintainability <span class="citation">(Romano et al., 2018)</span>. Note: This study covers dead code broadly; dead exports specifically have less direct research, but the comprehensibility impact applies equally.</p>
<p>Clean, well-organized code improves AI agent reliability. Agents working with minimal cognitive noise—including clean API surfaces—experience lower break rates <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>0-2:</strong> Clean API, minimal dead exports</li>
<li><strong>3-5:</strong> Low, some cleanup needed</li>
<li><strong>6-10:</strong> Moderate, review public API surface</li>
<li><strong>11+:</strong> High, significant cleanup opportunity</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Make unused exports private/internal</li>
<li>Delete truly dead code after confirming no external usage</li>
<li>Use IDE tools to identify unused exports</li>
<li>Review API surface during code review</li>
</ul>`,
	},

	// ============================================================================
	// C4: Documentation Quality Metrics
	// ============================================================================
	"readme_word_count": {
		Brief:     "README length in words. README organization and maintenance correlate with project success <span class=\"citation\">(Prana et al., 2019)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>The word count of the project's README file. Measures documentation completeness for the primary entry point that developers (and agents) encounter when exploring a project.</p>

<h4>Why It Matters for AI Agents</h4>
<p>The README is the first documentation agents read when given a task. A comprehensive README helps agents understand project purpose, architecture, conventions, and how to contribute. Without this context, agents make incorrect assumptions about project structure and practices.</p>

<h4>Research Evidence</h4>
<p>An empirical study of 4,226 GitHub README files identified eight content categories that well-documented projects include: what the project does, how to install/use it, contribution guidelines, and examples <span class="citation">(Prana et al., 2019)</span>. This research provides an empirical foundation for README completeness assessment.</p>
<p>Research on the correlation between README files and project popularity found that README organization and update frequency positively associate with GitHub stars—projects with well-maintained READMEs attract more users and contributors <span class="citation">(Wang et al., 2023)</span>. For AI agents, documentation quality is a component of overall code health that predicts agent reliability <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>500+:</strong> Comprehensive, excellent for agents</li>
<li><strong>200-499:</strong> Good coverage of basics</li>
<li><strong>100-199:</strong> Minimal, may lack important details</li>
<li><strong>0-99:</strong> Sparse, agents will struggle with context</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Include project purpose, installation, and quickstart sections</li>
<li>Document architecture and key decisions</li>
<li>Add examples of common use cases</li>
<li>Include contribution guidelines and code conventions</li>
</ul>`,
	},

	"comment_density": {
		Brief:     "Percentage of lines that are comments. Comment quality matters more than quantity <span class=\"citation\">(Rani et al., 2022)</span>.",
		Threshold: 5.0,
		Detailed: `<h4>Definition</h4>
<p>The percentage of source lines that are comments. Measures how much inline documentation exists to explain code purpose, assumptions, and non-obvious behavior.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Comments explain intent that cannot be derived from code alone. Agents use comments to understand why code exists, what edge cases it handles, and what assumptions it makes. Well-commented code helps agents generate appropriate modifications that preserve intent.</p>

<h4>Research Evidence</h4>
<p>Knuth's foundational work on literate programming established that programs should be written primarily for humans to read, with code secondary <span class="citation">(Knuth, 1984)</span>. This philosophy underlies the value of meaningful comments.</p>
<p>A systematic literature review of comment quality research identified 21 distinct quality attributes, with consistency between comments and code being the predominant factor <span class="citation">(Rani et al., 2022)</span>. Code-comment inconsistencies are common: a large-scale study analyzing 1.3 billion AST changes identified 13 types of inconsistencies between comments and the code they describe <span class="citation">(Wen et al., 2019)</span>.</p>
<p>For AI agents, code health metrics including documentation quality predict agent reliability <span class="citation">(Borg et al., 2026)</span>. Comments that explain "why" rather than "what" are especially valuable for agent comprehension.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>10-25%:</strong> Well-balanced documentation</li>
<li><strong>5-9%:</strong> Moderate, may miss important context</li>
<li><strong>26-40%:</strong> High, may indicate over-complex code</li>
<li><strong>0-4% or 41%+:</strong> Extreme, review documentation strategy</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Add comments explaining "why" for non-obvious code</li>
<li>Document assumptions and edge cases</li>
<li>Use doc comments for public APIs</li>
<li>Remove outdated or obvious comments</li>
</ul>`,
	},

	"api_doc_coverage": {
		Brief:     "Percentage of public APIs with documentation. Incomplete API documentation is a major obstacle to effective code reuse <span class=\"citation\">(Robillard, 2011)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>The percentage of public functions, methods, classes, and types that have documentation comments (doc strings, JSDoc, GoDoc). Measures formal API documentation coverage.</p>

<h4>Why It Matters for AI Agents</h4>
<p>API documentation is the contract between modules. Agents rely on doc comments to understand function purposes, parameter meanings, return values, and error conditions. Without API docs, agents must infer behavior from implementation, which is error-prone.</p>

<h4>Research Evidence</h4>
<p>A multi-phased study of over 440 Microsoft developers found that documentation-related obstacles are among the most severe faced when learning new APIs <span class="citation">(Robillard, 2011)</span>. The study identified five critical factors for API documentation design, including documentation of intent and provision of examples.</p>
<p>Systematic analysis of 323 developers and 179 API documentation units revealed that the three most severe documentation problems are ambiguity, incompleteness, and incorrectness <span class="citation">(Uddin & Robillard, 2015)</span>. Six of the ten studied problems were mentioned as "blockers" that forced developers to abandon an API entirely.</p>
<p>Industrial case studies confirm that documentation quality varies significantly by task type—implementation tasks require different documentation than maintenance tasks <span class="citation">(Garousi et al., 2013)</span>. For AI agents, code health metrics including API documentation predict agent reliability <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>80-100%:</strong> Well-documented APIs</li>
<li><strong>60-79%:</strong> Good coverage with gaps</li>
<li><strong>40-59%:</strong> Partial, many undocumented APIs</li>
<li><strong>0-39%:</strong> Poor, agents will struggle to use APIs correctly</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Add doc comments to all public functions and types</li>
<li>Document parameter meanings and valid ranges</li>
<li>Describe return values and error conditions</li>
<li>Include examples in doc comments for complex APIs</li>
</ul>`,
	},

	"changelog_present": {
		Brief:     "Whether a CHANGELOG exists. Release note content varies by system but serves critical communication function <span class=\"citation\">(Abebe et al., 2016)</span>.",
		Threshold: 5.0,
		Detailed: `<h4>Definition</h4>
<p>Binary metric indicating whether the project has a CHANGELOG file documenting version history, notable changes, and migration guides. Common formats include CHANGELOG.md, HISTORY.md, or NEWS.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Changelogs help agents understand how the project has evolved and what changes exist between versions. When agents work on upgrades or migrations, changelogs provide crucial context about breaking changes and deprecated features.</p>

<h4>Research Evidence</h4>
<p>An empirical study of software release notes identified six types of content: new features, fixed bugs, changes, known issues, technical details, and other information <span class="citation">(Abebe et al., 2016)</span>. The study found that content varies between systems and even between versions of the same system, but structured release documentation serves a critical communication function.</p>
<p>Note: Changelog-specific research is sparse; release notes studies provide the closest proxy for understanding version history documentation value.</p>
<p>For AI agents, comprehensive project documentation including version history is a component of code health that predicts agent reliability <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>Present:</strong> Project tracks version history</li>
<li><strong>Absent:</strong> No formal version documentation</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Create CHANGELOG.md following keepachangelog.com format</li>
<li>Document notable changes for each release</li>
<li>Include breaking changes, deprecations, and security fixes</li>
<li>Automate changelog generation from commit messages</li>
</ul>`,
	},

	"examples_present": {
		Brief:     "Whether example code exists. Examples are critical for API learning and reduce developer mistakes <span class=\"citation\">(Robillard, 2011)</span>.",
		Threshold: 5.0,
		Detailed: `<h4>Definition</h4>
<p>Binary metric indicating whether the project includes example code in an examples/, demo/, or similar directory, or inline examples in documentation. Also counts example functions in tests (ExampleXxx in Go).</p>

<h4>Why It Matters for AI Agents</h4>
<p>Examples are the most effective way to communicate intended usage. Agents can pattern-match against examples to generate code that follows project conventions. Without examples, agents may use APIs in unintended ways.</p>

<h4>Research Evidence</h4>
<p>Research on API learning obstacles found that examples are a critical factor for API learning—developers rely heavily on examples for initial understanding and problem-solving <span class="citation">(Robillard, 2011)</span>.</p>
<p>A study of REST API documentation effectiveness found that examples reduce developer mistakes, improve task success rate, and increase developer satisfaction <span class="citation">(Sohan et al., 2017)</span>. Examples serve as a critical design factor that helps developers understand how to correctly use APIs <span class="citation">(Uddin & Robillard, 2015)</span>.</p>
<p>For AI agents, well-documented code including examples improves agent reliability and reduces errors in generated code <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>Present:</strong> Project provides usage examples</li>
<li><strong>Absent:</strong> No example code available</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Create an examples/ directory with runnable code</li>
<li>Add Example functions to Go tests</li>
<li>Include code samples in README and API docs</li>
<li>Document common use cases with complete examples</li>
</ul>`,
	},

	"contributing_present": {
		Brief:     "Whether CONTRIBUTING guide exists. Contribution guidelines are one of eight essential README categories <span class=\"citation\">(Prana et al., 2019)</span>.",
		Threshold: 5.0,
		Detailed: `<h4>Definition</h4>
<p>Binary metric indicating whether the project has a CONTRIBUTING file explaining how to contribute: code style, testing requirements, pull request process, and development setup.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Contributing guidelines tell agents how to make changes that will be accepted. This includes code style, testing requirements, commit message formats, and review processes. Agents following these guidelines produce higher-quality contributions.</p>

<h4>Research Evidence</h4>
<p>An empirical study of GitHub README files identified eight content categories found in well-documented projects, with contribution guidelines being one of these essential categories <span class="citation">(Prana et al., 2019)</span>. Projects that document how to contribute tend to receive higher-quality contributions.</p>
<p>Note: Dedicated contributing file research is emerging; current research covers contribution guidelines within README files. The principle applies equally to standalone CONTRIBUTING.md files.</p>
<p>For AI agents, clear contribution guidelines are part of comprehensive documentation that predicts agent reliability <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>Present:</strong> Project documents contribution process</li>
<li><strong>Absent:</strong> No contribution guidelines</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Create CONTRIBUTING.md with development setup instructions</li>
<li>Document code style and linting requirements</li>
<li>Explain testing expectations for contributions</li>
<li>Describe the pull request review process</li>
</ul>`,
	},

	"diagrams_present": {
		Brief:     "Whether architecture diagrams exist. Visual notation aids comprehension of object-oriented designs <span class=\"citation\">(Gamma et al., 1994)</span>.",
		Threshold: 5.0,
		Detailed: `<h4>Definition</h4>
<p>Binary metric indicating whether the project includes architecture diagrams, flow charts, or other visual documentation. Detects common formats: .svg, .png, .mermaid in docs/, or diagram blocks in markdown.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Diagrams communicate system structure more effectively than text for certain relationships. While current agents primarily process text, diagram descriptions in alt-text or accompanying text help agents understand high-level architecture.</p>

<h4>Research Evidence</h4>
<p>The Design Patterns book established that visual notation—class diagrams, object diagrams, and interaction diagrams—aids comprehension of object-oriented designs and relationships <span class="citation">(Gamma et al., 1994)</span>. Diagrams make abstract patterns concrete and navigable.</p>
<p>Note: Diagram effectiveness for AI agents is indirect since agents primarily process text. However, diagrams with descriptive alt-text and accompanying textual explanations contribute to overall documentation quality.</p>
<p>For AI agents, comprehensive documentation including architectural visualization is a component of code health that predicts agent reliability <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>Present:</strong> Project includes visual documentation</li>
<li><strong>Absent:</strong> No diagrams available</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Create architecture overview diagrams</li>
<li>Document data flow with sequence diagrams</li>
<li>Use Mermaid for version-controlled diagrams</li>
<li>Include component relationship diagrams</li>
</ul>`,
	},

	// ============================================================================
	// C5: Temporal Dynamics Metrics
	// ============================================================================
	"churn_rate": {
		Brief:     "Average code changes per file over time. Code churn strongly predicts defect-prone areas <span class=\"citation\">(Kim et al., 2007)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Measures how frequently code changes over time, calculated from git history. High churn indicates files that are modified often, potentially due to instability, evolving requirements, or maintenance burden.</p>

<h4>Why It Matters for AI Agents</h4>
<p>High-churn code is more likely to change again soon, increasing the risk that agent modifications will conflict with ongoing work. Stable code provides a reliable foundation for agent changes. Churn also correlates with defect density, meaning high-churn areas are riskier for automated modification.</p>

<h4>Research Evidence</h4>
<p>Foundational research established that process measures derived from change history are more predictive of faults than product metrics like code size <span class="citation">(Graves et al., 2000)</span>. Nagappan and Ball demonstrated that relative code churn measures (churn normalized by component size) predict system defect density with 89% accuracy on Windows Server 2003 <span class="citation">(Nagappan & Ball, 2005)</span>.</p>
<p>Kim et al. extended this work, showing that change history patterns using a cache-based strategy effectively predict fault-prone files across seven software systems <span class="citation">(Kim et al., 2007)</span>. Tornhill synthesizes this research into practitioner guidance, identifying high-churn files as complexity hotspots requiring special attention <span class="citation">(Tornhill, 2015)</span>. Note: Tornhill is influential practitioner literature synthesizing academic research.</p>
<p>While AI-era research has not specifically tested temporal metrics, code health broadly predicts agent reliability <span class="citation">(Borg et al., 2026)</span>. The connection is indirect but logical: temporal metrics predict defect-prone areas, which are harder for AI agents to modify successfully.</p>

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

	"temporal_coupling_pct": {
		Brief:     "Files that change together. Temporal coupling reveals hidden dependencies not visible in code structure <span class=\"citation\">(Gall et al., 1998)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Percentage of file pairs that frequently change together in commits but have no direct import relationship. Indicates hidden coupling not visible in code structure but present in change patterns.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Temporal coupling reveals hidden dependencies that agents cannot see from code alone. When files are temporally coupled, changing one without the other often introduces bugs. Agents may miss these implicit relationships.</p>

<h4>Research Evidence</h4>
<p>Gall et al. pioneered the detection of "logical coupling" from product release history, demonstrating that change patterns reveal architectural dependencies not apparent from static code analysis <span class="citation">(Gall et al., 1998)</span>. This foundational work established that modules changing together often indicate design issues or restructuring opportunities.</p>
<p>D'Ambros et al. empirically validated that change coupling correlates with software defects across three large systems, and that incorporating change coupling information improves bug prediction models <span class="citation">(D'Ambros et al., 2009)</span>. Tornhill synthesizes this research into practitioner guidance, showing how temporal coupling analysis reveals hidden dependencies requiring attention <span class="citation">(Tornhill, 2015)</span>. Note: Tornhill represents influential practitioner literature.</p>
<p>While AI-era research focuses on structural code health rather than temporal metrics specifically, the principle applies: hidden dependencies that surprise developers also surprise AI agents <span class="citation">(Borg et al., 2026)</span>.</p>

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

	"author_fragmentation": {
		Brief:     "Number of distinct authors per file. Ownership fragmentation increases defect rates <span class=\"citation\">(Bird et al., 2011)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Average number of distinct commit authors per file, measured from git history. High fragmentation indicates code touched by many developers without clear ownership.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Files with many authors often have inconsistent styles and patterns from different contributors. This inconsistency makes it harder for agents to match existing conventions. Clear ownership typically correlates with consistent, maintainable code.</p>

<h4>Research Evidence</h4>
<p>Bird et al. conducted a definitive study on code ownership at Microsoft, finding that ownership measures relate to both pre-release and post-release faults <span class="citation">(Bird et al., 2011)</span>. Components with many low-expertise contributors had significantly higher defect rates than those with clear ownership. The study quantified that minor contributors (those with less than 5% of changes) increase defect risk.</p>
<p>Kim et al.'s work on fault prediction from change history also incorporates developer contribution patterns, showing that author metrics contribute to prediction accuracy <span class="citation">(Kim et al., 2007)</span>. Tornhill synthesizes this research, identifying author fragmentation as an indicator of knowledge silos and potential quality issues <span class="citation">(Tornhill, 2015)</span>. Note: Tornhill represents influential practitioner literature.</p>
<p>For AI agents, code with inconsistent patterns from multiple authors is harder to modify in a style-consistent way. Code health metrics broadly predict agent reliability <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>1-2:</strong> Clear ownership, consistent style</li>
<li><strong>3-4:</strong> Moderate, shared responsibility</li>
<li><strong>5-7:</strong> High fragmentation, style variance likely</li>
<li><strong>8+:</strong> Very high, review code consistency</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Assign code owners via CODEOWNERS file</li>
<li>Enforce consistent style with automated formatters</li>
<li>Review pull requests for consistency with existing code</li>
<li>Consolidate fragmented code into focused modules</li>
</ul>`,
	},

	"commit_stability": {
		Brief:     "Ratio of additions to modifications. Code decay manifests through increasing modification patterns <span class=\"citation\">(Eick et al., 2001)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Measures the balance between new code additions and modifications to existing code. High stability indicates more additive development; low stability suggests significant rework or refactoring.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Stable code that mostly receives additions is safer for agent modifications. Frequently rewritten code may change again soon, potentially conflicting with agent work. Stability indicates mature, settled designs.</p>

<h4>Research Evidence</h4>
<p>Eick et al. defined and studied "code decay"—the phenomenon where code becomes increasingly difficult to change over time <span class="citation">(Eick et al., 2001)</span>. Their research identified change patterns as both symptoms and predictors of decay, establishing that high modification rates relative to additions indicate code under stress.</p>
<p>Graves et al. demonstrated that modification patterns from change history predict future defects, with recently and frequently modified code being more fault-prone <span class="citation">(Graves et al., 2000)</span>. Tornhill extends this into practitioner guidance, using commit patterns as indicators of code maturity and stability <span class="citation">(Tornhill, 2015)</span>. Note: Tornhill represents influential practitioner literature.</p>
<p>Note: Commit stability as a specific ratio metric has limited dedicated research. The thresholds represent practitioner consensus rather than empirically derived values. The underlying principle—that modification patterns indicate instability—has strong research support.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>0.7-1.0:</strong> Very stable, mostly additions</li>
<li><strong>0.4-0.69:</strong> Stable, normal evolution</li>
<li><strong>0.2-0.39:</strong> Moderate, significant modifications</li>
<li><strong>0-0.19:</strong> Unstable, heavy rework</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Complete refactoring before adding new features</li>
<li>Stabilize design through architecture review</li>
<li>Improve requirements gathering to reduce rework</li>
<li>Add tests to prevent regression-driven rewrites</li>
</ul>`,
	},

	"hotspot_concentration": {
		Brief:     "How concentrated changes are in a few files. Churn concentration identifies high-defect-density components <span class=\"citation\">(Nagappan & Ball, 2005)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Measures how concentrated code changes are in a small number of "hotspot" files. High concentration means most changes happen in few files; low concentration indicates changes are distributed evenly.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Hotspot files are high-risk modification targets. When agents must modify hotspots, the risk of conflicts and regressions is higher. Distributed changes across many files indicate healthier architecture with single-purpose modules.</p>

<h4>Research Evidence</h4>
<p>Nagappan and Ball's research on code churn showed that churn concentration—where changes cluster in specific components—identifies high-defect-density areas <span class="citation">(Nagappan & Ball, 2005)</span>. Components with concentrated changes had disproportionately higher defect rates.</p>
<p>Hassan extended this work, demonstrating that the complexity of changes (measured by entropy across files) predicts faults <span class="citation">(Hassan, 2009)</span>. Hotspots with high change entropy are particularly fault-prone. Tornhill synthesizes this research into practitioner guidance, identifying hotspots as primary refactoring targets following the Pareto principle: 20% of files often account for 80% of bugs and changes <span class="citation">(Tornhill, 2015)</span>. Note: Tornhill represents influential practitioner literature; the 20/80 ratio is a common heuristic, not an empirically derived constant.</p>
<p>For AI agents, hotspots represent high-risk modification targets. Code health metrics broadly predict agent reliability, with defect-prone areas being harder for agents to modify successfully <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>0-20%:</strong> Well-distributed changes</li>
<li><strong>21-40%:</strong> Some hotspots, normal</li>
<li><strong>41-60%:</strong> Concentrated, review hotspots</li>
<li><strong>61%+:</strong> Very concentrated, architectural concern</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Refactor hotspot files into smaller modules</li>
<li>Apply Single Responsibility Principle to large files</li>
<li>Review if hotspots have too many responsibilities</li>
<li>Prioritize hotspots for testing and documentation</li>
</ul>`,
	},

	// ============================================================================
	// C6: Testing Metrics
	// ============================================================================
	"test_to_code_ratio": {
		Brief:     "Ratio of test code to production code. TDD teams see 40-90% fewer defects with comprehensive testing <span class=\"citation\">(Nagappan et al., 2008)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>The ratio of test lines of code to production lines of code. A ratio of 1.0 means equal amounts of test and production code. Higher ratios indicate more comprehensive testing.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Tests are the safety net that catches agent mistakes. With good test coverage, agents can make changes and immediately verify they haven't broken existing functionality. Without tests, agent changes may introduce silent regressions.</p>

<h4>Research Evidence</h4>
<p>Beck established the test-first methodology that makes systematic testing practical <span class="citation">(Beck, 2002)</span>. Industrial studies at Microsoft and IBM found that teams using TDD experienced 40-90% lower pre-release defect density, with the trade-off of 15-35% longer initial development time <span class="citation">(Nagappan et al., 2008)</span>.</p>
<p>Recent research on AI agents shows that code health metrics predict agent reliability, making test infrastructure a critical factor for AI-assisted development <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>1.0+:</strong> Comprehensive testing, excellent for agents</li>
<li><strong>0.5-0.99:</strong> Good test coverage</li>
<li><strong>0.2-0.49:</strong> Moderate, critical paths covered</li>
<li><strong>0-0.19:</strong> Minimal testing, high regression risk</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Add tests for all new functionality</li>
<li>Write tests when fixing bugs to prevent regression</li>
<li>Focus on testing public APIs and edge cases</li>
<li>Use test coverage reports to identify gaps</li>
</ul>`,
	},

	"coverage_percent": {
		Brief:     "Percentage of code covered by tests. Coverage correlates with fewer field defects <span class=\"citation\">(Mockus et al., 2009)</span>, though effect size is debated <span class=\"citation\">(Inozemtseva & Holmes, 2014)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>The percentage of code statements executed during test runs. Measures how much of the codebase has test verification. Can include line coverage, branch coverage, or combined metrics.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Coverage indicates which code has verified behavior. Agents can modify covered code with confidence that tests will catch mistakes. Uncovered code is a blind spot where agent changes may cause undetected problems.</p>

<h4>Research Evidence</h4>
<p>The relationship between coverage and defect detection is nuanced. Mockus et al. found that increases in coverage associate with fewer post-release field defects <span class="citation">(Mockus et al., 2009)</span>. However, Inozemtseva and Holmes demonstrated that when controlling for test suite size, coverage shows only "low to moderate correlation" with fault detection effectiveness <span class="citation">(Inozemtseva & Holmes, 2014)</span>.</p>
<p>The practical interpretation: coverage is necessary but not sufficient. Low coverage reliably indicates undertested code, but high coverage alone does not guarantee effective testing. Assertion quality and test design matter alongside coverage metrics. Recent research confirms that comprehensive test infrastructure is critical for AI agent reliability <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>80-100%:</strong> High coverage, strong safety net</li>
<li><strong>60-79%:</strong> Good coverage, most paths tested</li>
<li><strong>40-59%:</strong> Moderate, critical paths should be covered</li>
<li><strong>0-39%:</strong> Low coverage, high regression risk</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Use coverage reports to identify untested code</li>
<li>Prioritize coverage for critical business logic</li>
<li>Add tests for edge cases and error conditions</li>
<li>Set minimum coverage requirements in CI</li>
</ul>`,
	},

	"test_isolation": {
		Brief:     "Independence of tests from external state. Test doubles isolate the system under test from dependencies <span class=\"citation\">(Meszaros, 2007)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Measures how well tests are isolated from external dependencies: databases, file systems, network services, and global state. Isolated tests use mocks, stubs, or in-memory implementations.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Isolated tests run quickly and reliably, providing fast feedback for agent changes. Tests dependent on external services are flaky and slow, reducing agent iteration speed. Agents can more confidently modify code with reliable test suites.</p>

<h4>Research Evidence</h4>
<p>Meszaros established systematic patterns for test doubles (mocks, stubs, fakes) that isolate the System Under Test (SUT) from dependencies <span class="citation">(Meszaros, 2007)</span>. Beck emphasized that isolated tests run reliably, fast, and in any order—critical properties for rapid feedback <span class="citation">(Beck, 2002)</span>.</p>
<p>Luo et al. analyzed 201 flaky tests and found that shared state and external dependencies are primary causes of test flakiness <span class="citation">(Luo et al., 2014)</span>. Flaky tests erode confidence: if developers ignore failures "because it's flaky," real bugs slip through. AI agent performance depends on reliable test feedback for iterative code modification <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>90-100%:</strong> Highly isolated, fast and reliable</li>
<li><strong>70-89%:</strong> Good isolation, some integration tests</li>
<li><strong>50-69%:</strong> Mixed, review external dependencies</li>
<li><strong>0-49%:</strong> Poor isolation, likely flaky tests</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Use dependency injection to enable mocking</li>
<li>Replace database calls with in-memory implementations</li>
<li>Mock external service calls</li>
<li>Separate integration tests from unit tests</li>
</ul>`,
	},

	"assertion_density_avg": {
		Brief:     "Assertions per test. Assertion density negatively correlates with fault density in production code <span class=\"citation\">(Kudrjavets et al., 2006)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>The average number of assertions per test function. Measures how thoroughly tests verify expected behavior versus simply executing code paths.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Tests without sufficient assertions may pass while actual behavior is incorrect. When agents modify code, assertion-dense tests catch subtle bugs that path coverage alone would miss. Each assertion is a specification that agents must preserve.</p>

<h4>Research Evidence</h4>
<p>Kudrjavets et al. studied Windows components and found that assertion density in production code negatively correlates with fault density—components with more assertions had fewer bugs <span class="citation">(Kudrjavets et al., 2006)</span>. While this study focused on production assertions, the principle applies to test assertions: explicit verification catches errors that mere execution would miss.</p>
<p>Beck emphasizes that tests should verify behavior, not just execute code paths <span class="citation">(Beck, 2002)</span>. A test that runs without assertions is not a test—it's documentation at best. AI agents benefit from assertion-dense tests because each assertion acts as a specification that must be preserved during code modification <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>3+:</strong> Thorough verification per test</li>
<li><strong>2:</strong> Good, basic verification</li>
<li><strong>1:</strong> Minimal, may miss issues</li>
<li><strong>0:</strong> No assertions, tests provide no verification</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Add assertions for return values, state changes, and side effects</li>
<li>Verify error conditions, not just happy paths</li>
<li>Check boundary conditions and edge cases</li>
<li>Assert on specific values, not just truthiness</li>
</ul>`,
	},

	"test_file_ratio": {
		Brief:     "Ratio of test files to source files. Systematic test organization follows TDD structure <span class=\"citation\">(Meszaros, 2007)</span>.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>The ratio of test files to production source files. Measures whether tests are systematically organized to cover the codebase. A ratio of 1.0 means one test file per source file.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Systematic test organization helps agents locate tests for code they're modifying. When each source file has a corresponding test file, agents can easily find and extend relevant tests. Random test organization makes test discovery difficult.</p>

<h4>Research Evidence</h4>
<p>Meszaros documents systematic test organization patterns that enhance maintainability and navigation <span class="citation">(Meszaros, 2007)</span>. Beck's TDD methodology naturally produces organized test structure by requiring tests before implementation <span class="citation">(Beck, 2002)</span>.</p>
<p>Well-organized code structures significantly improve AI agent comprehension and reliability. Agents benefit from predictable file layouts where test files mirror source files, enabling automated test discovery and modification <span class="citation">(Borg et al., 2026)</span>.</p>

<h4>Recommended Thresholds</h4>
<ul>
<li><strong>0.8-1.0:</strong> Systematic coverage, one test file per module</li>
<li><strong>0.5-0.79:</strong> Good coverage, most modules tested</li>
<li><strong>0.3-0.49:</strong> Moderate, some modules lack tests</li>
<li><strong>0-0.29:</strong> Low, many modules untested</li>
</ul>

<h4>How to Improve</h4>
<ul>
<li>Create test files mirroring source file structure</li>
<li>Follow naming conventions (foo.go -> foo_test.go)</li>
<li>Add test stubs for untested modules</li>
<li>Review test coverage by module</li>
</ul>`,
	},
}

// getMetricDescription returns the description for a metric, with a default fallback.
func getMetricDescription(metricName string) MetricDescription {
	if desc, ok := metricDescriptions[metricName]; ok {
		return desc
	}
	return MetricDescription{Threshold: 6.0} // Default fallback
}
