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

import "github.com/spf13/cobra"

// Flag constants for service configuration.
const (
	autoStartFlag       = "auto-start"
	delayedStartFlag    = "delayed-start"
	listenFlag          = "listen"
	dataFlag            = "data"
	insecureFlag        = "insecure"
	usernameFlag        = "username"
	passwordFlag        = "password"
	autoCertFlag        = "auto-cert"
	noDriverFlag        = "no-driver"
	enableIntegrations  = "enable-integrations"
	disableIntegrations = "disable-integrations"
	peersFlag           = "peers"
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
	Use:   "install",
	Short: "Install Synnax as a Windows Service",
	Long: `Install Synnax as a Windows Service.

Core configuration flags (--listen, --data, --insecure, etc.) will be stored
in the service configuration and used when the service starts.

Example:
  synnax service install --listen 0.0.0.0:9090 --data C:\ProgramData\Synnax\data --insecure`,
	Args: cobra.NoArgs,
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

	// Server configuration flags (same as start command)
	serviceInstallCmd.Flags().StringP(
		listenFlag,
		"l",
		"localhost:9090",
		"The address to listen for client connections",
	)
	serviceInstallCmd.Flags().StringP(
		dataFlag,
		"d",
		"",
		"Directory where the Synnax node will store its data (default: C:\\ProgramData\\Synnax\\data)",
	)
	serviceInstallCmd.Flags().BoolP(
		insecureFlag,
		"i",
		false,
		"Disable encryption, authentication, and authorization",
	)
	serviceInstallCmd.Flags().String(
		usernameFlag,
		"synnax",
		"Username for the admin user",
	)
	serviceInstallCmd.Flags().String(
		passwordFlag,
		"seldon",
		"Password for the admin user",
	)
	serviceInstallCmd.Flags().Bool(
		autoCertFlag,
		false,
		"Automatically generate self-signed certificates",
	)
	serviceInstallCmd.Flags().Bool(
		noDriverFlag,
		false,
		"Disable the embedded Synnax driver",
	)
	serviceInstallCmd.Flags().StringSlice(
		enableIntegrations,
		nil,
		"Device integrations to enable (labjack, modbus, ni, opc, sequence)",
	)
	serviceInstallCmd.Flags().StringSlice(
		disableIntegrations,
		nil,
		"Device integrations to disable (labjack, modbus, ni, opc, sequence)",
	)
	serviceInstallCmd.Flags().StringSliceP(
		peersFlag,
		"p",
		nil,
		"Addresses of additional peers in the cluster",
	)

	return nil
}

func buildConfigFromFlags(c *cobra.Command) (Config, error) {
	flags := c.Flags()
	listen, err := flags.GetString(listenFlag)
	if err != nil {
		return Config{}, err
	}
	data, err := flags.GetString(dataFlag)
	if err != nil {
		return Config{}, err
	}
	insecure, err := flags.GetBool(insecureFlag)
	if err != nil {
		return Config{}, err
	}
	username, err := flags.GetString(usernameFlag)
	if err != nil {
		return Config{}, err
	}
	password, err := flags.GetString(passwordFlag)
	if err != nil {
		return Config{}, err
	}
	autoCert, err := flags.GetBool(autoCertFlag)
	if err != nil {
		return Config{}, err
	}
	noDriver, err := flags.GetBool(noDriverFlag)
	if err != nil {
		return Config{}, err
	}
	peers, err := flags.GetStringSlice(peersFlag)
	if err != nil {
		return Config{}, err
	}
	enableInt, err := flags.GetStringSlice(enableIntegrations)
	if err != nil {
		return Config{}, err
	}
	disableInt, err := flags.GetStringSlice(disableIntegrations)
	if err != nil {
		return Config{}, err
	}
	autoStart, err := flags.GetBool(autoStartFlag)
	if err != nil {
		return Config{}, err
	}
	delayedStart, err := flags.GetBool(delayedStartFlag)
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
	c.Printf("Windows Service %s installed successfully.\n", Name)
	c.Printf("Use 'synnax service start' or 'net start %s' to start the service.\n", Name)
	return nil
}

func serviceUninstall(c *cobra.Command, _ []string) error {
	if err := Uninstall(); err != nil {
		return err
	}
	c.Printf("Windows Service %s uninstalled successfully.\n", Name)
	return nil
}

func serviceStart(c *cobra.Command, _ []string) error {
	if err := Start(); err != nil {
		return err
	}
	c.Printf("%s started.\n", Name)
	return nil
}

func serviceStop(c *cobra.Command, _ []string) error {
	if err := Stop(); err != nil {
		return err
	}
	c.Printf("%s stopped.\n", Name)
	return nil
}
