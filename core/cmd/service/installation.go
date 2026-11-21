// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package service provides the commands that run the Core as a Linux or Windows
// service, and the controls that `synnax update` uses on it.
package service

import (
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/synnaxlabs/synnax/cmd/start"
)

// Installation is the Core service on this host.
type Installation struct {
	// Binary is the path of the Core binary that the service runs.
	Binary string
	// Data is the data directory of the Core that the service runs.
	Data string
}

// DataDir returns the data directory that the data flag names, or the data directory of
// the Core service when the flag is not set and the service is installed.
func DataDir() (string, error) {
	if !viper.IsSet(start.FlagData) {
		inst, installed, err := Lookup()
		if err != nil {
			return "", err
		}
		if installed {
			return filepath.Clean(inst.Data), nil
		}
	}
	return filepath.Abs(viper.GetString(start.FlagData))
}
