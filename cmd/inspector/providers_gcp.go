//go:build gcp
// +build gcp

package main

import (
	_ "github.com/comfortablynumb/pmp-cloud-inspector/pkg/provider/gcp" // Register GCP provider
)
