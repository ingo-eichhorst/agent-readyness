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
		Brief:     "Average lines per function. Shorter functions (under 25 lines) are easier for agents to understand atomically.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Measures the average number of lines of code per function across the codebase. Includes all executable statements, comments within functions, and blank lines within function bodies.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Agents process code within context windows with limited capacity. Long functions consume more context, leaving less room for related code, documentation, and reasoning. Shorter functions allow agents to see complete units of behavior, understand purpose quickly, and make targeted modifications.</p>

<h4>Research Evidence</h4>
<p>Studies consistently show that smaller functions have fewer defects and are easier to understand <span class="citation">(Fowler et al., 1999)</span>. The "Single Responsibility Principle" suggests functions should do one thing, naturally leading to shorter implementations.</p>

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
		Brief:     "Average lines per file. Smaller files (under 300 lines) help agents navigate and understand module scope.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>The average number of lines per source file in the codebase, including code, comments, and blank lines. Measures overall file organization and module granularity.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Large files often indicate poor separation of concerns, making it harder for agents to locate relevant code and understand module boundaries. When agents need to modify code in large files, they risk unintended side effects due to hidden dependencies between distant sections.</p>

<h4>Research Evidence</h4>
<p>Module decomposition research shows that smaller, focused modules improve maintainability <span class="citation">(Parnas, 1972)</span>. Design patterns literature emphasizes cohesion: code that changes together should live together, but in manageable units <span class="citation">(Gamma et al., 1994)</span>.</p>

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
		Brief:     "Incoming dependencies per module. Lower coupling means modules can be modified more safely.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Afferent coupling (Ca) counts how many other modules depend on a given module. High afferent coupling means the module is heavily used throughout the codebase, making changes to it potentially far-reaching.</p>

<h4>Why It Matters for AI Agents</h4>
<p>When agents modify highly-coupled modules, changes ripple to all dependents. Agents must understand and account for all usages, which may exceed context window capacity. Lower coupling allows agents to make confident, isolated changes.</p>

<h4>Research Evidence</h4>
<p>Coupling is a key indicator of maintainability <span class="citation">(Parnas, 1972)</span>. The principle of loose coupling, central to good design, enables independent module evolution <span class="citation">(Gamma et al., 1994)</span>.</p>

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
		Brief:     "Outgoing dependencies per module. Modules depending on too many others become fragile.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Efferent coupling (Ce) counts how many other modules a given module depends on. High efferent coupling means the module relies on many external components, making it vulnerable to changes elsewhere.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Modules with high efferent coupling require agents to understand many dependencies before making changes. This increases cognitive load and the risk of missing interactions. Agents work best with self-contained modules having minimal external dependencies.</p>

<h4>Research Evidence</h4>
<p>The Stable Dependencies Principle suggests depending only on stable abstractions <span class="citation">(Martin, 2003)</span>. High efferent coupling often indicates a module is orchestrating too much, violating separation of concerns.</p>

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
		Brief:     "Percentage of duplicated code. Less duplication means fewer places to update when agents make changes.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>The percentage of code that appears multiple times in the codebase, typically measured as duplicate sequences of 6+ lines or tokens. Includes exact duplicates and near-duplicates with minor variations.</p>

<h4>Why It Matters for AI Agents</h4>
<p>When agents identify a bug or make an improvement, duplicated code requires the same change in multiple locations. Agents may miss some instances, leading to inconsistent behavior. Additionally, duplicates consume context window space without adding new information.</p>

<h4>Research Evidence</h4>
<p>Duplication is identified as a key code smell requiring refactoring <span class="citation">(Fowler et al., 1999)</span>. Studies show duplicated code has higher defect density and maintenance cost than unique code.</p>

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
		Brief:     "Percentage of values with explicit type annotations. Explicit types help agents understand data flow.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>The percentage of function parameters, return values, and variables that have explicit type annotations. In Go, this is inherent; in TypeScript and Python, it measures type hint usage.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Type annotations serve as machine-readable documentation of programmer intent. Agents use types to understand what data flows through the system, validate their changes are type-safe, and navigate codebases efficiently. Without types, agents must infer intent from usage patterns, which is error-prone.</p>

<h4>Research Evidence</h4>
<p>Research shows type annotations significantly reduce bugs and improve code comprehension <span class="citation">(Gao et al., 2017)</span>. Studies on TypeScript adoption found that typed code has 15% fewer bugs than untyped JavaScript <span class="citation">(Ore et al., 2018)</span>.</p>

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
		Brief:     "Adherence to naming conventions. Consistent naming helps agents predict and generate correct identifiers.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Measures how consistently the codebase follows naming conventions: camelCase for functions and variables, PascalCase for types and classes, UPPER_SNAKE_CASE for constants. Also checks for descriptive names over abbreviations.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Agents learn patterns from training data that follow common conventions. Inconsistent naming breaks these patterns, causing agents to generate code that clashes with local style. Consistent naming also helps agents infer purpose from names and maintain coherent code generation.</p>

<h4>Research Evidence</h4>
<p>Studies show identifier names are primary comprehension aids in code reading <span class="citation">(Sadowski et al., 2015)</span>. Consistent naming reduces cognitive load and helps both humans and AI systems understand code structure.</p>

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
		Brief:     "Unexplained numeric literals per 1,000 lines. Named constants help agents understand value significance.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Counts numeric literals in code (excluding 0, 1, and common values) that are not defined as named constants. Reported as occurrences per 1,000 lines of code. Magic numbers are unexplained values embedded directly in logic.</p>

<h4>Why It Matters for AI Agents</h4>
<p>When agents encounter magic numbers, they cannot determine the value's purpose or whether it can be safely changed. Named constants like MAX_RETRIES = 3 communicate intent; the literal 3 does not. Agents may incorrectly reuse or modify magic numbers without understanding their significance.</p>

<h4>Research Evidence</h4>
<p>Magic numbers are a classic code smell that reduces maintainability <span class="citation">(Fowler et al., 1999)</span>. Meaningful names for constants improve code understanding and reduce errors during modification.</p>

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
		Brief:     "Use of strict type checking features. Stricter typing catches more errors at compile time.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Measures whether the codebase uses strict type checking features: strict mode in TypeScript, strict mypy settings in Python, or equivalent. A binary metric (enabled or not) that significantly impacts type safety.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Strict type checking catches type errors at compile time rather than runtime. When agents generate code, strict checking provides immediate feedback on type mismatches, allowing agents to self-correct. Without strict mode, type errors may only surface during execution.</p>

<h4>Research Evidence</h4>
<p>Strict typing reduces runtime errors by catching type mismatches early <span class="citation">(Gao et al., 2017)</span>. TypeScript's strict mode prevents common JavaScript pitfalls that would otherwise require extensive testing to catch.</p>

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
		Brief:     "Handling of null/undefined values. Explicit null handling prevents agent-generated null pointer bugs.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Measures usage of null-safe patterns: optional chaining (?.), nullish coalescing (??), null assertions, and proper Optional/Maybe types. Also detects unsafe patterns like unchecked null dereferences.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Null reference errors are among the most common runtime failures. When agents generate code, they may not anticipate all cases where values could be null. Explicit null handling in the codebase establishes patterns that agents can follow, reducing null-related bugs in generated code.</p>

<h4>Research Evidence</h4>
<p>Tony Hoare called null references his "billion-dollar mistake." Modern language features for null safety (Optional types, null-aware operators) significantly reduce null pointer exceptions <span class="citation">(Gao et al., 2017)</span>.</p>

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
		Brief:     "Deepest directory nesting level. Shallower hierarchies are easier to navigate and understand.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>The maximum depth of directory nesting in the source tree, counting from the project root. Measures how deeply files are organized into subdirectories (e.g., src/api/v2/handlers/auth/utils.go = depth 6).</p>

<h4>Why It Matters for AI Agents</h4>
<p>Deep directory hierarchies make it harder for agents to locate related code and understand project organization. Long import paths consume context space and are prone to errors. Shallower structures provide clearer boundaries and easier navigation.</p>

<h4>Research Evidence</h4>
<p>Research on module decomposition emphasizes clarity of organization <span class="citation">(Parnas, 1972)</span>. Flat hierarchies reduce cognitive load when navigating codebases.</p>

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
		Brief:     "Average imports per module. Fewer imports per module means more focused, understandable code.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>The average number of imports per source file. Counts both internal and external dependencies, measuring how widely each module reaches into the rest of the codebase or ecosystem.</p>

<h4>Why It Matters for AI Agents</h4>
<p>High fanout means agents must understand many dependencies to reason about a single file. Each import brings potential side effects and API contracts into scope. Lower fanout creates more self-contained modules that agents can modify with confidence.</p>

<h4>Research Evidence</h4>
<p>Module coupling research shows that high fanout increases maintenance cost and error rates <span class="citation">(Gamma et al., 1994)</span>. The Interface Segregation Principle suggests depending only on interfaces you actually use.</p>

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
		Brief:     "Number of circular dependencies. Circular dependencies complicate reasoning and safe modifications.",
		Threshold: 7.0,
		Detailed: `<h4>Definition</h4>
<p>Counts the number of circular dependency chains where module A imports B which imports A (directly or transitively). Circular dependencies create ordering problems and make it impossible to understand modules in isolation.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Circular dependencies mean agents cannot understand one module without understanding all modules in the cycle. This creates reasoning complexity that scales with cycle size. Breaking cycles allows agents to analyze and modify modules independently.</p>

<h4>Research Evidence</h4>
<p>Circular dependencies violate the Acyclic Dependencies Principle <span class="citation">(Martin, 2003)</span>. They indicate architectural problems that impede testing, building, and understanding code <span class="citation">(Parnas, 1972)</span>.</p>

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
		Brief:     "Average complexity of import statements. Simpler imports are easier to understand and maintain.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Measures the complexity of import patterns: deep submodule imports, aliased imports, re-exports, and barrel files. Higher scores indicate more complex import structures that are harder to trace and understand.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Complex import patterns obscure where code actually lives. Agents may struggle to locate the true source of an import, especially with re-exports and barrel files. Simpler imports create clearer dependency graphs that agents can navigate.</p>

<h4>Research Evidence</h4>
<p>Clear module boundaries improve code comprehension <span class="citation">(Parnas, 1972)</span>. Import complexity is a form of accidental complexity that increases maintenance burden without adding value.</p>

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
		Brief:     "Exported symbols not used elsewhere. Dead exports clutter APIs and confuse agents about intended interfaces.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Counts exported/public symbols (functions, types, constants) that are never imported or used outside their defining module. These create noise in the public API without providing value.</p>

<h4>Why It Matters for AI Agents</h4>
<p>When agents explore a module's API, dead exports appear as valid options but lead to confusion when used. Agents may incorrectly incorporate unused functionality or spend context window space understanding code that serves no purpose.</p>

<h4>Research Evidence</h4>
<p>Clean APIs expose only necessary functionality <span class="citation">(Robillard, 2009)</span>. Dead code is a form of technical debt that increases cognitive load without benefit <span class="citation">(Fowler et al., 1999)</span>.</p>

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
		Brief:     "README length in words. Substantial READMEs help agents understand project purpose and setup.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>The word count of the project's README file. Measures documentation completeness for the primary entry point that developers (and agents) encounter when exploring a project.</p>

<h4>Why It Matters for AI Agents</h4>
<p>The README is the first documentation agents read when given a task. A comprehensive README helps agents understand project purpose, architecture, conventions, and how to contribute. Without this context, agents make incorrect assumptions about project structure and practices.</p>

<h4>Research Evidence</h4>
<p>Documentation quality directly impacts developer productivity <span class="citation">(Sadowski et al., 2015)</span>. Studies show that well-documented projects receive more contributions and have fewer repeated questions <span class="citation">(Robillard, 2009)</span>.</p>

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
		Brief:     "Percentage of lines that are comments. Balanced comments explain why, not what, helping agents understand intent.",
		Threshold: 5.0,
		Detailed: `<h4>Definition</h4>
<p>The percentage of source lines that are comments. Measures how much inline documentation exists to explain code purpose, assumptions, and non-obvious behavior.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Comments explain intent that cannot be derived from code alone. Agents use comments to understand why code exists, what edge cases it handles, and what assumptions it makes. Well-commented code helps agents generate appropriate modifications that preserve intent.</p>

<h4>Research Evidence</h4>
<p>Comments that explain "why" rather than "what" significantly aid code comprehension <span class="citation">(Sadowski et al., 2015)</span>. However, excessive comments can indicate code that is too complex and should be refactored <span class="citation">(Fowler et al., 1999)</span>.</p>

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
		Brief:     "Percentage of public APIs with documentation. Documented APIs help agents use and extend code correctly.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>The percentage of public functions, methods, classes, and types that have documentation comments (doc strings, JSDoc, GoDoc). Measures formal API documentation coverage.</p>

<h4>Why It Matters for AI Agents</h4>
<p>API documentation is the contract between modules. Agents rely on doc comments to understand function purposes, parameter meanings, return values, and error conditions. Without API docs, agents must infer behavior from implementation, which is error-prone.</p>

<h4>Research Evidence</h4>
<p>API documentation is the primary resource developers consult when using code <span class="citation">(Robillard, 2009)</span>. Incomplete API documentation is a major obstacle to effective code reuse <span class="citation">(Sadowski et al., 2015)</span>.</p>

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
		Brief:     "Whether a CHANGELOG exists. Changelogs help agents understand project evolution and version differences.",
		Threshold: 5.0,
		Detailed: `<h4>Definition</h4>
<p>Binary metric indicating whether the project has a CHANGELOG file documenting version history, notable changes, and migration guides. Common formats include CHANGELOG.md, HISTORY.md, or NEWS.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Changelogs help agents understand how the project has evolved and what changes exist between versions. When agents work on upgrades or migrations, changelogs provide crucial context about breaking changes and deprecated features.</p>

<h4>Research Evidence</h4>
<p>Version history documentation is essential for project maintenance <span class="citation">(Sadowski et al., 2015)</span>. Keep-a-changelog.com establishes community standards for this documentation.</p>

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
		Brief:     "Whether example code exists. Examples demonstrate intended usage patterns for agents to follow.",
		Threshold: 5.0,
		Detailed: `<h4>Definition</h4>
<p>Binary metric indicating whether the project includes example code in an examples/, demo/, or similar directory, or inline examples in documentation. Also counts example functions in tests (ExampleXxx in Go).</p>

<h4>Why It Matters for AI Agents</h4>
<p>Examples are the most effective way to communicate intended usage. Agents can pattern-match against examples to generate code that follows project conventions. Without examples, agents may use APIs in unintended ways.</p>

<h4>Research Evidence</h4>
<p>Developers rely heavily on examples when learning APIs <span class="citation">(Robillard, 2009)</span>. Example-driven documentation significantly improves correct API usage <span class="citation">(Sadowski et al., 2015)</span>.</p>

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
		Brief:     "Whether CONTRIBUTING guide exists. Contribution guidelines help agents follow project conventions.",
		Threshold: 5.0,
		Detailed: `<h4>Definition</h4>
<p>Binary metric indicating whether the project has a CONTRIBUTING file explaining how to contribute: code style, testing requirements, pull request process, and development setup.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Contributing guidelines tell agents how to make changes that will be accepted. This includes code style, testing requirements, commit message formats, and review processes. Agents following these guidelines produce higher-quality contributions.</p>

<h4>Research Evidence</h4>
<p>Clear contribution guidelines significantly increase contribution quality and reduce rejected pull requests <span class="citation">(Sadowski et al., 2015)</span>.</p>

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
		Brief:     "Whether architecture diagrams exist. Visual documentation helps agents understand system structure quickly.",
		Threshold: 5.0,
		Detailed: `<h4>Definition</h4>
<p>Binary metric indicating whether the project includes architecture diagrams, flow charts, or other visual documentation. Detects common formats: .svg, .png, .mermaid in docs/, or diagram blocks in markdown.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Diagrams communicate system structure more effectively than text for certain relationships. While current agents primarily process text, diagram descriptions in alt-text or accompanying text help agents understand high-level architecture.</p>

<h4>Research Evidence</h4>
<p>Visual documentation aids comprehension of complex systems <span class="citation">(Gamma et al., 1994)</span>. Architecture diagrams are particularly valuable for understanding component relationships and data flow.</p>

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
		Brief:     "Average code changes per file over time. Lower churn indicates stable code that's safer to modify.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Measures how frequently code changes over time, calculated from git history. High churn indicates files that are modified often, potentially due to instability, evolving requirements, or maintenance burden.</p>

<h4>Why It Matters for AI Agents</h4>
<p>High-churn code is more likely to change again soon, increasing the risk that agent modifications will conflict with ongoing work. Stable code provides a reliable foundation for agent changes. Churn also correlates with defect density.</p>

<h4>Research Evidence</h4>
<p>Code churn is a strong predictor of defects <span class="citation">(Kim et al., 2007)</span>. High-churn files are often complexity hotspots requiring special attention <span class="citation">(Tornhill, 2015)</span>.</p>

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
		Brief:     "Files that change together. Lower temporal coupling means more independent components.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Percentage of file pairs that frequently change together in commits but have no direct import relationship. Indicates hidden coupling not visible in code structure but present in change patterns.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Temporal coupling reveals hidden dependencies that agents cannot see from code alone. When files are temporally coupled, changing one without the other often introduces bugs. Agents may miss these implicit relationships.</p>

<h4>Research Evidence</h4>
<p>Temporal coupling analysis reveals architectural issues not visible in static code <span class="citation">(Tornhill, 2015)</span>. Files that change together often should be co-located or have explicit dependencies.</p>

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
		Brief:     "Number of distinct authors per file. Fewer authors suggests clearer ownership and consistent style.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Average number of distinct commit authors per file, measured from git history. High fragmentation indicates code touched by many developers without clear ownership.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Files with many authors often have inconsistent styles and patterns from different contributors. This inconsistency makes it harder for agents to match existing conventions. Clear ownership typically correlates with consistent, maintainable code.</p>

<h4>Research Evidence</h4>
<p>Code ownership patterns affect defect rates <span class="citation">(Kim et al., 2007)</span>. Files without clear ownership tend to accumulate inconsistencies and technical debt.</p>

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
		Brief:     "Ratio of additions to modifications. Higher stability means less rework and more forward progress.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Measures the balance between new code additions and modifications to existing code. High stability indicates more additive development; low stability suggests significant rework or refactoring.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Stable code that mostly receives additions is safer for agent modifications. Frequently rewritten code may change again soon, potentially conflicting with agent work. Stability indicates mature, settled designs.</p>

<h4>Research Evidence</h4>
<p>Modification patterns predict future defects <span class="citation">(Kim et al., 2007)</span>. Code stability is associated with maturity and lower maintenance burden.</p>

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
		Brief:     "How concentrated changes are in a few files. Lower concentration means distributed, healthier codebase.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Measures how concentrated code changes are in a small number of "hotspot" files. High concentration means most changes happen in few files; low concentration indicates changes are distributed evenly.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Hotspot files are high-risk modification targets. When agents must modify hotspots, the risk of conflicts and regressions is higher. Distributed changes across many files indicate healthier architecture with single-purpose modules.</p>

<h4>Research Evidence</h4>
<p>Hotspots are primary targets for quality improvement <span class="citation">(Tornhill, 2015)</span>. The Pareto principle often applies: 20% of files account for 80% of bugs and changes.</p>

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
		Brief:     "Ratio of test code to production code. More tests provide safety nets for agent modifications.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>The ratio of test lines of code to production lines of code. A ratio of 1.0 means equal amounts of test and production code. Higher ratios indicate more comprehensive testing.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Tests are the safety net that catches agent mistakes. With good test coverage, agents can make changes and immediately verify they haven't broken existing functionality. Without tests, agent changes may introduce silent regressions.</p>

<h4>Research Evidence</h4>
<p>Test-driven development improves code quality and reduces defect rates <span class="citation">(Beck, 2002)</span>. Higher test ratios correlate with fewer production bugs <span class="citation">(Mockus et al., 2009)</span>.</p>

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
		Brief:     "Percentage of code covered by tests. Higher coverage means more verified behavior for agents to rely on.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>The percentage of code statements executed during test runs. Measures how much of the codebase has test verification. Can include line coverage, branch coverage, or combined metrics.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Coverage indicates which code has verified behavior. Agents can modify covered code with confidence that tests will catch mistakes. Uncovered code is a blind spot where agent changes may cause undetected problems.</p>

<h4>Research Evidence</h4>
<p>Coverage correlates with defect detection <span class="citation">(Mockus et al., 2009)</span>. While coverage alone doesn't guarantee quality, low coverage reliably indicates undertested code.</p>

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
		Brief:     "Independence of tests from external state. Isolated tests run reliably and help agents verify changes quickly.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>Measures how well tests are isolated from external dependencies: databases, file systems, network services, and global state. Isolated tests use mocks, stubs, or in-memory implementations.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Isolated tests run quickly and reliably, providing fast feedback for agent changes. Tests dependent on external services are flaky and slow, reducing agent iteration speed. Agents can more confidently modify code with reliable test suites.</p>

<h4>Research Evidence</h4>
<p>Test isolation is fundamental to effective testing <span class="citation">(Beck, 2002)</span>. Flaky tests that depend on external state erode confidence and slow development.</p>

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
		Brief:     "Assertions per test. More assertions per test provide stronger verification of expected behavior.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>The average number of assertions per test function. Measures how thoroughly tests verify expected behavior versus simply executing code paths.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Tests without sufficient assertions may pass while actual behavior is incorrect. When agents modify code, assertion-dense tests catch subtle bugs that path coverage alone would miss. Each assertion is a specification that agents must preserve.</p>

<h4>Research Evidence</h4>
<p>Assertion density correlates with defect detection effectiveness <span class="citation">(Mockus et al., 2009)</span>. Tests should verify behavior, not just execute code <span class="citation">(Beck, 2002)</span>.</p>

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
		Brief:     "Ratio of test files to source files. Higher ratios suggest systematic test organization.",
		Threshold: 6.0,
		Detailed: `<h4>Definition</h4>
<p>The ratio of test files to production source files. Measures whether tests are systematically organized to cover the codebase. A ratio of 1.0 means one test file per source file.</p>

<h4>Why It Matters for AI Agents</h4>
<p>Systematic test organization helps agents locate tests for code they're modifying. When each source file has a corresponding test file, agents can easily find and extend relevant tests. Random test organization makes test discovery difficult.</p>

<h4>Research Evidence</h4>
<p>Consistent project structure aids navigation and comprehension <span class="citation">(Sadowski et al., 2015)</span>. Test organization patterns like "test mirrors source" improve maintainability.</p>

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
