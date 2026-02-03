package output

// Citation represents a research reference for ARS metrics.
type Citation struct {
	Category    string
	Title       string
	Authors     string
	Year        int
	URL         string
	Description string
}

// researchCitations contains research backing for each metric category.
var researchCitations = []Citation{
	{
		Category:    "C1",
		Title:       "Refactoring: Improving the Design of Existing Code",
		Authors:     "Fowler et al.",
		Year:        1999,
		URL:         "https://martinfowler.com/books/refactoring.html",
		Description: "Cyclomatic complexity as a maintainability indicator",
	},
	{
		Category:    "C1",
		Title:       "A Complexity Measure",
		Authors:     "McCabe",
		Year:        1976,
		URL:         "https://ieeexplore.ieee.org/document/1702388",
		Description: "Original cyclomatic complexity metric definition",
	},
	{
		Category:    "C2",
		Title:       "To Type or Not to Type",
		Authors:     "Gao et al.",
		Year:        2017,
		URL:         "https://dl.acm.org/doi/10.1145/3133850",
		Description: "Type annotations improve code understanding and reduce bugs",
	},
	{
		Category:    "C2",
		Title:       "How Developers Use Types",
		Authors:     "Ore et al.",
		Year:        2018,
		URL:         "https://dl.acm.org/doi/10.1145/3276954.3276963",
		Description: "Type annotations aid navigation and comprehension",
	},
	{
		Category:    "C3",
		Title:       "Design Patterns: Elements of Reusable Object-Oriented Software",
		Authors:     "Gamma et al.",
		Year:        1994,
		URL:         "https://en.wikipedia.org/wiki/Design_Patterns",
		Description: "Foundational patterns for managing coupling and dependencies",
	},
	{
		Category:    "C3",
		Title:       "On the Criteria To Be Used in Decomposing Systems into Modules",
		Authors:     "Parnas",
		Year:        1972,
		URL:         "https://dl.acm.org/doi/10.1145/361598.361623",
		Description: "Module decomposition principles for maintainability",
	},
	{
		Category:    "C4",
		Title:       "How Developers Search for Code",
		Authors:     "Sadowski et al.",
		Year:        2015,
		URL:         "https://dl.acm.org/doi/10.1145/2786805.2786855",
		Description: "Documentation quality impacts developer productivity",
	},
	{
		Category:    "C4",
		Title:       "API Documentation Quality",
		Authors:     "Robillard",
		Year:        2009,
		URL:         "https://ieeexplore.ieee.org/document/5070510",
		Description: "API documentation as a critical success factor",
	},
	{
		Category:    "C5",
		Title:       "Your Code as a Crime Scene",
		Authors:     "Tornhill",
		Year:        2015,
		URL:         "https://pragprog.com/titles/atcrime/your-code-as-a-crime-scene/",
		Description: "Temporal coupling and code churn analysis",
	},
	{
		Category:    "C5",
		Title:       "Predicting Faults from Cached History",
		Authors:     "Kim et al.",
		Year:        2007,
		URL:         "https://dl.acm.org/doi/10.1145/1287624.1287633",
		Description: "Change history predicts future defects",
	},
	{
		Category:    "C6",
		Title:       "Test-Driven Development: By Example",
		Authors:     "Beck",
		Year:        2002,
		URL:         "https://www.pearson.com/en-us/subject-catalog/p/test-driven-development-by-example/P200000009480",
		Description: "Test coverage as quality indicator",
	},
	{
		Category:    "C6",
		Title:       "Code Coverage and Fault Detection",
		Authors:     "Mockus et al.",
		Year:        2009,
		URL:         "https://dl.acm.org/doi/10.1145/1595696.1595713",
		Description: "Relationship between test coverage and defect rates",
	},
}
