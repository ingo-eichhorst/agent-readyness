package pkga

import "github.com/ingo-eichhorst/agent-readyness/testdata/coupling/pkgb"

// UseB calls into pkgb, creating an efferent dependency from pkga to pkgb.
func UseB() string {
	return pkgb.Hello()
}
