//go:build azure
// +build azure

package main

import (
	_ "github.com/comfortablynumb/pmp-cloud-inspector/pkg/provider/azure" // Register Azure provider
)
