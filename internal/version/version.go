// Package version provides the build-time version string for the logos binary.
// The Version variable is overridden at build time via -ldflags:
//
//	go build -ldflags "-X github.com/senna-lang/logosyncx/internal/version.Version=v0.1.0" -o logos .
package version

import (
	"fmt"
	"runtime"
)

// Version is set at build time via -ldflags.
// It defaults to "dev" for local builds without ldflags.
var Version = "dev"

// String returns a human-readable version string including OS and architecture.
// Example: "logos v0.1.0 (darwin/arm64)"
func String() string {
	return fmt.Sprintf("logos %s (%s/%s)", Version, runtime.GOOS, runtime.GOARCH)
}

// IsDev reports whether this is a development (non-release) build.
func IsDev() bool {
	return Version == "dev"
}
