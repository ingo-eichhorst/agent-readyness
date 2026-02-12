// Package version provides the ARS tool version.
package version

// Version is the ARS tool version.
// Can be overridden at build time with:
//   go build -ldflags "-X github.com/ingo-eichhorst/agent-readyness/pkg/version.Version=2.0.1"
var Version = "dev"
