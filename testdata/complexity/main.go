package main

// SimpleFunc has cyclomatic complexity 1 (no branches).
func SimpleFunc() int {
	return 42
}

// OneBranch has cyclomatic complexity 2 (1 if).
func OneBranch(x int) int {
	if x > 0 {
		return x
	}
	return -x
}

// MultiBranch has cyclomatic complexity 6:
// base 1 + if(1) + for(1) + case(3 cases, each +1 except default) = 1+1+1+3 = 6
func MultiBranch(x int, items []string) string {
	if x > 0 {
		for _, item := range items {
			switch item {
			case "a":
				return "alpha"
			case "b":
				return "beta"
			case "c":
				return "gamma"
			}
		}
	}
	return "none"
}

func main() {}
