package lib

// ExportedUsed is called by the user package.
func ExportedUsed() string {
	return "used"
}

// ExportedUnused is never referenced by any other package.
func ExportedUnused() string {
	return "unused"
}

// UnusedType is an exported type never referenced externally.
type UnusedType struct{}
