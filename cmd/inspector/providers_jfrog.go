//go:build jfrog
// +build jfrog

package main

import (
	_ "github.com/comfortablynumb/pmp-cloud-inspector/pkg/provider/jfrog" // Register JFrog provider
)
