package agent

// IntentClarityTask tests the agent's ability to understand code purpose.
var intentClarityTask = task{
	ID:          "intent_clarity",
	Name:        "Intent Clarity",
	Description: "Measures agent's ability to find and explain code purpose",
	Prompt: `Find the main entry point function in this project and explain what it does. Include:
1) The function name and file location
2) What the function accomplishes
3) Key dependencies or imports it uses

Be specific and reference actual code.`,
	ToolsAllowed:   "Read,Glob,Grep",
	TimeoutSeconds: 300,
}

// ModificationConfidenceTask tests the agent's ability to propose targeted changes.
var modificationConfidenceTask = task{
	ID:          "modification_confidence",
	Name:        "Modification Confidence",
	Description: "Measures agent's ability to propose safe, scoped changes",
	Prompt: `Find a function in this project that would benefit from input validation. Propose the specific validation code to add, showing:
1) The function to modify and its location
2) The exact code changes (diff-style)
3) Why this validation is appropriate

Do not actually modify the file - just propose the change.`,
	ToolsAllowed:   "Read,Glob,Grep",
	TimeoutSeconds: 300,
}

// CrossFileCoherenceTask tests the agent's multi-file navigation capabilities.
var crossFileCoherenceTask = task{
	ID:          "cross_file_coherence",
	Name:        "Cross-File Coherence",
	Description: "Measures agent's ability to trace code across files",
	Prompt: `Trace the data flow for the primary operation in this project. Start from an entry point (CLI command, API handler, or main function) and follow the data through to where it's processed or stored. Show:
1) The starting point
2) Each file/function the data passes through
3) The final destination

Reference actual file paths and function names.`,
	ToolsAllowed:   "Read,Glob,Grep",
	TimeoutSeconds: 300,
}

// SemanticCompletenessTask tests the agent's context-aware modification abilities.
var semanticCompletenessTask = task{
	ID:          "semantic_completeness",
	Name:        "Semantic Completeness",
	Description: "Measures agent's ability to follow existing patterns",
	Prompt: `Find an area of this codebase that could benefit from improved error handling. Propose error handling that matches existing patterns in the codebase. Show:
1) The code location needing improvement
2) Examples of existing error handling patterns you found
3) Your proposed changes that follow those patterns

Do not actually modify files - just propose the change.`,
	ToolsAllowed:   "Read,Glob,Grep",
	TimeoutSeconds: 300,
}

// AllTasks returns all standardized C7 evaluation tasks.
func allTasks() []task {
	return []task{
		intentClarityTask,
		modificationConfidenceTask,
		crossFileCoherenceTask,
		semanticCompletenessTask,
	}
}
