//go:build gitlab
// +build gitlab

package main

import (
	_ "github.com/comfortablynumb/pmp-cloud-inspector/pkg/provider/gitlab" // Register GitLab provider
)
