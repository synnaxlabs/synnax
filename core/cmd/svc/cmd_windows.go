// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build windows

package svc

import (
	"github.com/spf13/cobra"
	"github.com/synnaxlabs/synnax/cmd/flags"
)

// Service-specific flag constants.
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

var serviceInstallCmd = &cobra.Command{
	Use:   "install [flags]",
	Short: "Install Synnax as a Windows Service",
	Long: `Install Synnax as a Windows Service.

Core configuration flags (--listen, --data, --insecure, etc.) will be stored
in the service configuration and used when the service starts.

Example:
  synnax service install --listen 0.0.0.0:9090 --data C:\ProgramData\Synnax\data --insecure`,
	RunE: serviceInstall,
}

var serviceUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall Synnax Windows Service",
	Long: `Uninstall the Synnax Windows Service.

This will stop the service if it is running and remove it from the system.`,
	Args: cobra.NoArgs,
	RunE: serviceUninstall,
}

var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Synnax Windows Service",
	Args:  cobra.NoArgs,
	RunE:  serviceStart,
}

var serviceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Synnax Windows Service",
	Args:  cobra.NoArgs,
	RunE:  serviceStop,
}

// RegisterCommands registers the service command and all subcommands with the given
// root command.
func RegisterCommands(root *cobra.Command) error {
	root.AddCommand(serviceCmd)
	serviceCmd.AddCommand(serviceInstallCmd)
	serviceCmd.AddCommand(serviceUninstallCmd)
	serviceCmd.AddCommand(serviceStartCmd)
	serviceCmd.AddCommand(serviceStopCmd)
	return configureServiceInstallFlags()
}

func configureServiceInstallFlags() error {
	// Service-specific flags
	serviceInstallCmd.Flags().Bool(
		autoStartFlag,
		true,
		"Start the service automatically when Windows starts",
	)
	serviceInstallCmd.Flags().Bool(
		delayedStartFlag,
		false,
		"Delay service start until after Windows startup completes",
	)

	// Add the common server configuration flags (shared with start command)
	flags.ConfigureServerFlags(serviceInstallCmd)

	return nil
}

func buildConfigFromFlags(c *cobra.Command) (Config, error) {
	cmdFlags := c.Flags()
	listen, err := cmdFlags.GetString(flags.Listen)
	if err != nil {
		return Config{}, err
	}
	data, err := cmdFlags.GetString(flags.Data)
	if err != nil {
		return Config{}, err
	}
	insecure, err := cmdFlags.GetBool(flags.Insecure)
	if err != nil {
		return Config{}, err
	}
	username, err := cmdFlags.GetString(flags.Username)
	if err != nil {
		return Config{}, err
	}
	password, err := cmdFlags.GetString(flags.Password)
	if err != nil {
		return Config{}, err
	}
	autoCert, err := cmdFlags.GetBool(flags.AutoCert)
	if err != nil {
		return Config{}, err
	}
	noDriver, err := cmdFlags.GetBool(flags.NoDriver)
	if err != nil {
		return Config{}, err
	}
	peers, err := cmdFlags.GetStringSlice(flags.Peers)
	if err != nil {
		return Config{}, err
	}
	enableInt, err := cmdFlags.GetStringSlice(flags.EnableIntegrations)
	if err != nil {
		return Config{}, err
	}
	disableInt, err := cmdFlags.GetStringSlice(flags.DisableIntegrations)
	if err != nil {
		return Config{}, err
	}
	autoStart, err := cmdFlags.GetBool(autoStartFlag)
	if err != nil {
		return Config{}, err
	}
	delayedStart, err := cmdFlags.GetBool(delayedStartFlag)
	if err != nil {
		return Config{}, err
	}

	return Config{
		ListenAddress:       listen,
		DataDir:             data,
		Insecure:            insecure,
		Username:            username,
		Password:            password,
		AutoCert:            autoCert,
		NoDriver:            noDriver,
		Peers:               peers,
		EnableIntegrations:  enableInt,
		DisableIntegrations: disableInt,
		AutoStart:           autoStart,
		DelayedStart:        delayedStart,
	}, nil
}

func serviceInstall(c *cobra.Command, _ []string) error {
	cfg, err := buildConfigFromFlags(c)
	if err != nil {
		return err
	}
	if err := Install(cfg); err != nil {
		return err
	}
	c.Printf("Windows Service %s installed successfully.\n", name)
	c.Printf("Use 'synnax service start' or 'net start %s' to start the service.\n", name)
	return nil
}

func serviceUninstall(c *cobra.Command, _ []string) error {
	if err := Uninstall(); err != nil {
		return err
	}
	c.Printf("Windows Service %s uninstalled successfully.\n", name)
	return nil
}

func serviceStart(c *cobra.Command, _ []string) error {
	if err := Start(); err != nil {
		return err
	}
	c.Printf("%s started.\n", name)
	return nil
}

func serviceStop(c *cobra.Command, _ []string) error {
	if err := Stop(); err != nil {
		return err
	}
	c.Printf("%s stopped.\n", name)
	return nil
}
