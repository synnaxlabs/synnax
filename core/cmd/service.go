// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd

import "github.com/spf13/cobra"

const (
	serviceName        = "SynnaxServer"
	serviceDisplayName = "Synnax Server"
	serviceDescription = "Synnax telemetry engine for hardware systems"
)

const (
	serviceAutoStartFlag    = "auto-start"
	serviceDelayedStartFlag = "delayed-start"
	serviceUserFlag         = "user"
	servicePasswordFlag     = "service-password"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage Synnax as a Windows Service",
	Long: `Manage Synnax as a Windows Service. This command is only available on Windows.

The service subcommands allow you to install, uninstall, start, and stop Synnax
as a Windows Service. When running as a service, Synnax will receive proper
shutdown signals, enabling graceful shutdown of both the server and embedded driver.`,
	Args: cobra.NoArgs,
}

var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Synnax as a Windows Service",
	Long: `Install Synnax as a Windows Service.

Server configuration flags (--listen, --data, --insecure, etc.) will be stored
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

func init() {
	root.AddCommand(serviceCmd)

	serviceCmd.AddCommand(serviceInstallCmd)
	serviceCmd.AddCommand(serviceUninstallCmd)
	serviceCmd.AddCommand(serviceStartCmd)
	serviceCmd.AddCommand(serviceStopCmd)

	configureServiceInstallFlags()
}

func configureServiceInstallFlags() {
	// Service-specific flags
	serviceInstallCmd.Flags().Bool(
		serviceAutoStartFlag,
		true,
		"Start the service automatically when Windows starts",
	)
	serviceInstallCmd.Flags().Bool(
		serviceDelayedStartFlag,
		false,
		"Delay service start until after Windows startup completes",
	)
	serviceInstallCmd.Flags().String(
		serviceUserFlag,
		"",
		"User account for the service (default: LocalSystem)",
	)
	serviceInstallCmd.Flags().String(
		servicePasswordFlag,
		"",
		"Password for the service user account",
	)

	// Inherit start command flags for server configuration
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
		enableIntegrationsFlag,
		nil,
		"Device integrations to enable (labjack, modbus, ni, opc, sequence)",
	)
	serviceInstallCmd.Flags().StringSlice(
		disableIntegrationsFlag,
		nil,
		"Device integrations to disable (labjack, modbus, ni, opc, sequence)",
	)
	serviceInstallCmd.Flags().StringSliceP(
		peersFlag,
		"p",
		nil,
		"Addresses of additional peers in the cluster",
	)
}
