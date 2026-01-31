package user

import "github.com/ingo/agent-readyness/testdata/deadcode/lib"

// Use calls lib.ExportedUsed, leaving ExportedUnused as dead code.
func Use() string {
	return lib.ExportedUsed()
}
