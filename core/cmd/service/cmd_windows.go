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
	flagAutoStart    = "auto-start"
	flagDelayedStart = "delayed-start"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage Synnax as a Windows service",
	Long: `Manage Synnax as a Windows service.

The service subcommands allow you to install, uninstall, start, and stop Synnax as a Windows service. When running as a service, Synnax will receive proper shutdown signals, enabling graceful shutdown of both the Core and embedded Driver.`,
	Args: cobra.NoArgs,
}

var installCmd = &cobra.Command{
	Use:   "install [flags]",
	Short: "Install Synnax as a Windows service",
	Long: `Install Synnax as a Windows service.

Core configuration flags (--listen, --data, --insecure, etc.) will be stored in a YAML config file at C:\ProgramData\Synnax\config.yaml and used when the service starts. You can edit this file to change the configuration without reinstalling the service.`,
	PreRun:  syncFlagsToViper,
	RunE:    runInstall,
	Args:    cobra.NoArgs,
	Example: "synnax service install --listen localhost:9090 --insecure",
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall Synnax Windows service",
	Long: `Uninstall the Synnax Windows service.

This will stop the service if it is running and remove it from the system.`,
	Args: cobra.NoArgs,
	RunE: runUninstall,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Synnax Windows service",
	Long:  "Start the Synnax Windows service.",
	Args:  cobra.NoArgs,
	RunE:  runStart,
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Synnax Windows service",
	Long:  "Stop the Synnax Windows service.",
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
		flagAutoStart,
		true,
		"Start the service automatically when Windows starts",
	)
	installCmd.Flags().Bool(
		flagDelayedStart,
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
	if err := install(); err != nil {
		return err
	}
	c.Printf("Windows service %s installed successfully.\n", name)
	c.Printf("Configuration saved to: %s\n", ConfigPath())
	c.Printf("Use 'synnax service start' to start %s.\n", name)
	return nil
}

func runUninstall(c *cobra.Command, _ []string) error {
	if err := uninstall(); err != nil {
		return err
	}
	c.Printf("Windows service %s uninstalled successfully.\n", name)
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
