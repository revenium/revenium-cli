// Package build provides build-time version information.
// Variables are set via ldflags during compilation.
package build

// Version is the semantic version of the binary.
var Version = "dev"

// Commit is the git commit hash of the build.
var Commit = "none"

// Date is the build date in ISO 8601 format.
var Date = "unknown"
