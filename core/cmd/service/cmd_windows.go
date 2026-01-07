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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/synnax/cmd/start"
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
in the service configuration and used when the service starts.

Example:
  synnax service install --listen 0.0.0.0:9090 --data C:\ProgramData\Synnax\data --insecure`,
	RunE: runInstall,
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

// AddCommand adds the service subcommand to the given parent command.
func AddCommand(cmd *cobra.Command) {
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
	start.BindFlags(installCmd)
	serviceCmd.AddCommand(uninstallCmd)
	serviceCmd.AddCommand(startCmd)
	serviceCmd.AddCommand(stopCmd)
}

func buildConfigFromFlags(c *cobra.Command) (Config, error) {
	listen := viper.GetString(start.FlagListen)
	data := viper.GetString(start.FlagData)
	insecure := viper.GetBool(start.FlagInsecure)
	username := viper.GetString(start.FlagUsername)
	password := viper.GetString(start.FlagPassword)
	autoCert := viper.GetBool(start.FlagAutoCert)
	noDriver := viper.GetBool(start.FlagNoDriver)
	peers := viper.GetStringSlice(start.FlagPeers)
	enableInt := viper.GetStringSlice(start.FlagEnableIntegrations)
	disableInt := viper.GetStringSlice(start.FlagDisableIntegrations)
	autoStart := viper.GetBool(autoStartFlag)
	delayedStart := viper.GetBool(delayedStartFlag)
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

func runInstall(c *cobra.Command, _ []string) error {
	cfg, err := buildConfigFromFlags(c)
	if err != nil {
		return err
	}
	if err := install(cfg); err != nil {
		return err
	}
	c.Printf("Windows Service %s installed successfully.\n", name)
	c.Printf("Use 'synnax service start' or 'net start %s' to start the service.\n", name)
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
	if err := Start(); err != nil {
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
