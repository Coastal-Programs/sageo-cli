package main

import (
	"os"

	"github.com/jakeschepis/sageo-cli/internal/cli"
	versionpkg "github.com/jakeschepis/sageo-cli/internal/version"
)

// Injected at build time via -ldflags:
//
//	-X main.version=v0.1.0
var version = "dev"

func main() {
	versionpkg.Set(version)
	if err := cli.Execute(version); err != nil {
		os.Exit(1)
	}
}
