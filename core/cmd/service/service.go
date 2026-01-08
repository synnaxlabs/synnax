// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package service provides service management functionality for Synnax. This package
// encapsulates all service-related operations including installation, uninstallation,
// starting, and stopping. Currently, only Windows service is supported.
package service

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config holds the service-specific configuration for installing the Synnax service.
// Core configuration is read from viper and written to the YAML config file.
type Config struct {
	// AutoStart enables automatic service start on machine bootup.
	AutoStart bool
	// DelayedStart delays service start until after OS startup completes.
	DelayedStart bool
}

// ConfigDir returns the directory where the service config file is stored.
func ConfigDir() string {
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = "C:\\ProgramData"
	}
	return filepath.Join(programData, "Synnax")
}

// ConfigPath returns the full path to the service config file.
func ConfigPath() string { return filepath.Join(ConfigDir(), "config.yaml") }

// configKeysToExclude contains keys that should not be written to the config file.
// These are either service-specific (not needed at runtime) or internal.
var configKeysToExclude = map[string]bool{
	"auto-start":    true,
	"delayed-start": true,
}

// WriteConfig writes the current viper configuration to the config file.
// This captures all the core configuration flags set during service installation,
// excluding service-specific flags like auto-start and delayed-start.
func WriteConfig() error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Debug: print viper state
	fmt.Printf("DEBUG WriteConfig: viper.AllKeys() = %v\n", viper.AllKeys())
	fmt.Printf("DEBUG WriteConfig: viper.Get('insecure') = %v\n", viper.Get("insecure"))
	fmt.Printf("DEBUG WriteConfig: viper.GetBool('insecure') = %v\n", viper.GetBool("insecure"))
	fmt.Printf("DEBUG WriteConfig: viper.AllSettings() = %v\n", viper.AllSettings())

	// Build settings map by explicitly reading each key from viper.
	// This is necessary because viper.AllSettings() doesn't properly evaluate
	// bound flag values - it returns defaults instead of actual flag values.
	settings := make(map[string]any)
	for _, key := range viper.AllKeys() {
		if configKeysToExclude[key] {
			continue
		}
		settings[key] = viper.Get(key)
	}

	fmt.Printf("DEBUG WriteConfig: final settings = %v\n", settings)

	data, err := yaml.Marshal(settings)
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), data, 0644)
}
