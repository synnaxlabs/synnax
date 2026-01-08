// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build windows

package service

import (
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/synnax/cmd/cert"
	"github.com/synnaxlabs/synnax/cmd/instrumentation"
	cmdstart "github.com/synnaxlabs/synnax/cmd/start"
)

const (
	autoStartFlag    = "auto-start"
	delayedStartFlag = "delayed-start"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage Synnax as a Windows Service",
	Long: `Manage Synnax as a Windows Service.

The service subcommands allow you to install, uninstall, start, and stop Synnax as a
Windows Service. When running as a service, Synnax will receive proper shutdown signals,
enabling graceful shutdown of both the Core and embedded Driver.`,
	Args: cobra.NoArgs,
}

var installCmd = &cobra.Command{
	Use:   "install [flags]",
	Short: "Install Synnax as a Windows Service",
	Long: `Install Synnax as a Windows Service.

Core configuration flags (--listen, --data, --insecure, etc.) will be stored
in a YAML config file at C:\ProgramData\Synnax\config.yaml and used when the
service starts. You can edit this file to change the configuration without
reinstalling the service.

Example:
  synnax service install --listen 0.0.0.0:9090 --insecure`,
	// PreRunE syncs changed cobra flags to viper. This is necessary because
	// viper.BindPFlags doesn't properly pick up flag values after cobra parses them.
	PreRun: syncFlagsToViper,
	RunE:   runInstall,
	Args:   cobra.NoArgs,
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall Synnax Windows Service",
	Long: `Uninstall the Synnax Windows Service.

This will stop the service if it is running and remove it from the system.`,
	Args: cobra.NoArgs,
	RunE: runUninstall,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Synnax Windows Service",
	Args:  cobra.NoArgs,
	RunE:  runStart,
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Synnax Windows Service",
	Args:  cobra.NoArgs,
	RunE:  runStop,
}

// syncFlagsToViper syncs changed cobra flags to viper. This is necessary because
// viper.BindPFlags doesn't properly pick up flag values after cobra parses them.
func syncFlagsToViper(cmd *cobra.Command, _ []string) {
	viper.BindPFlags(cmd.Flags())
	viper.SetDefault(cmdstart.FlagData, filepath.Join(ConfigDir(), "data"))
	viper.SetDefault(instrumentation.FlagLogFilePath, filepath.Join(ConfigDir(), "logs", "synnax.log"))
	viper.SetDefault(cert.FlagCertsDir, filepath.Join(ConfigDir(), "certs"))
}

// AddCommand adds the service subcommand to the given parent command.
func AddCommand(cmd *cobra.Command) error {
	cmd.AddCommand(serviceCmd)
	serviceCmd.AddCommand(installCmd)
	installCmd.Flags().Bool(
		autoStartFlag,
		true,
		"Start the service automatically when Windows starts",
	)
	installCmd.Flags().Bool(
		delayedStartFlag,
		false,
		"Delay service start until after Windows startup completes",
	)
	cmdstart.BindFlags(installCmd)
	serviceCmd.AddCommand(uninstallCmd)
	serviceCmd.AddCommand(startCmd)
	serviceCmd.AddCommand(stopCmd)
	return viper.BindPFlags(cmd.Flags())
}

func runInstall(c *cobra.Command, _ []string) error {
	cfg := Config{
		AutoStart:    viper.GetBool(autoStartFlag),
		DelayedStart: viper.GetBool(delayedStartFlag),
	}
	if err := install(cfg); err != nil {
		return err
	}
	c.Printf("Windows Service %s installed successfully.\n", name)
	c.Printf("Configuration saved to: %s\n", ConfigPath())
	c.Printf("Use 'synnax service start' or 'net start %s' to start %s.\n", name, name)
	return nil
}

func runUninstall(c *cobra.Command, _ []string) error {
	if err := uninstall(); err != nil {
		return err
	}
	c.Printf("Windows Service %s uninstalled successfully.\n", name)
	return nil
}

func runStart(c *cobra.Command, _ []string) error {
	if err := start(); err != nil {
		return err
	}
	c.Printf("%s started.\n", name)
	return nil
}

func runStop(c *cobra.Command, _ []string) error {
	if err := stop(); err != nil {
		return err
	}
	c.Printf("%s stopped.\n", name)
	return nil
}
