// Package version exposes the running sageo-cli binary's version string.
//
// main() injects the ldflags-provided version at startup via Set. Packages
// that need to stamp the version into persistent artifacts (snapshots,
// reports) read it through Current() to avoid import cycles with main.
package version

// current is set by main.Set at startup and read by Current.
var current = "dev"

// Set records the build-time version string. Called from main.
func Set(v string) {
	if v != "" {
		current = v
	}
}

// Current returns the active version string. Defaults to "dev" if unset.
func Current() string {
	return current
}
