// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build linux || windows

package service

import (
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
	"github.com/synnaxlabs/synnax/cmd/cert"
	cmdinst "github.com/synnaxlabs/synnax/cmd/instrumentation"
	cmdstart "github.com/synnaxlabs/synnax/cmd/start"
	"github.com/synnaxlabs/x/errors"
	xos "github.com/synnaxlabs/x/os"
	"github.com/synnaxlabs/x/set"
	"gopkg.in/yaml.v3"
)

// configKeysToExclude contains keys that should not be written to the config file.
// These are either service-specific (not needed at runtime) or internal.
var configKeysToExclude = set.New("auto-start", "delayed-start")

// WriteConfig writes the current viper configuration to the config file.
// This captures all the core configuration flags set during service installation,
// excluding service-specific flags like auto-start and delayed-start.
// If a config file already exists, it is preserved to avoid overwriting
// user-customized settings during reinstallation.
func WriteConfig() error {
	exists, err := xos.FileExists(ConfigPath())
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	dir := ConfigDir()
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Build settings map by explicitly reading each key from viper.
	// This is necessary because viper.AllSettings() doesn't properly evaluate
	// bound flag values - it returns defaults instead of actual flag values.
	settings := make(map[string]any)
	for _, key := range viper.AllKeys() {
		if !configKeysToExclude.Contains(key) {
			settings[key] = viper.Get(key)
		}
	}
	data, err := yaml.Marshal(settings)
	if err != nil {
		return err
	}
	// The file can hold the root password and the license key.
	return os.WriteFile(ConfigPath(), data, 0o640)
}

// resolvePaths makes the data, log, certificate, and secrets key paths in viper
// absolute against the working directory, because a service starts in another
// directory. It leaves an empty path empty.
func resolvePaths() error {
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}
	for _, key := range []string{
		cmdstart.FlagData,
		cmdinst.FlagLogFilePath,
		cert.FlagCertsDir,
		cmdstart.FlagSecretsKeyFile,
	} {
		if p := viper.GetString(key); p != "" && !filepath.IsAbs(p) {
			viper.Set(key, filepath.Join(workDir, p))
		}
	}
	return nil
}

// stopMargin is how much longer than the Core's shutdown timeout a stop waits.
const stopMargin = 15 * time.Second

// stopTimeout returns how long a stop of the service waits for the Core to exit: the
// shutdown timeout in the service configuration plus stopMargin. The service reads no
// environment variables, so neither does stopTimeout.
func stopTimeout() (time.Duration, error) {
	v := viper.New()
	v.SetConfigFile(ConfigPath())
	flag := cmdstart.Cmd.Flags().Lookup(cmdstart.FlagShutdownTimeout)
	if err := v.BindPFlag(cmdstart.FlagShutdownTimeout, flag); err != nil {
		return 0, err
	}
	if err := v.ReadInConfig(); err != nil {
		return 0, errors.Wrap(err, "failed to read service configuration")
	}
	return v.GetDuration(cmdstart.FlagShutdownTimeout) + stopMargin, nil
}
