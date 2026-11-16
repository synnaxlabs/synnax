// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/version"
)

const (
	flagAutoStart    = "auto-start"
	flagDelayedStart = "delayed-start"
)

// NewCmd returns the command group that manages the Core as a Windows service.
func NewCmd() *cobra.Command {
	group := &cobra.Command{
		Use:   "service",
		Short: "Manage Synnax as a Windows service",
		Long: `Manage Synnax as a Windows service.

The service subcommands allow you to install, uninstall, start, and stop Synnax as a Windows service. When running as a service, Synnax will receive proper shutdown signals, enabling graceful shutdown of both the Core and embedded Driver.`,
		Args: cobra.NoArgs,
	}
	installCmd := &cobra.Command{
		Use:   "install [flags]",
		Short: "Install Synnax as a Windows service",
		Long: `Install Synnax as a Windows service.

Core configuration flags (--listen, --data, --insecure, etc.) will be stored in a YAML config file at C:\ProgramData\Synnax\config.yaml and used when the service starts. You can edit this file to change the configuration without reinstalling the service.`,
		PreRunE: syncFlagsToViper,
		RunE:    runInstall,
		Args:    cobra.NoArgs,
		Example: "synnax service install --listen localhost:9090 --insecure",
	}
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
	cmdstart.AddFlags(installCmd)
	group.AddCommand(
		installCmd,
		&cobra.Command{
			Use:   "uninstall",
			Short: "Uninstall Synnax Windows service",
			Long: `Uninstall the Synnax Windows service.

This will stop the service if it is running and remove it from the system.`,
			Args: cobra.NoArgs,
			RunE: runUninstall,
		},
		&cobra.Command{
			Use:   "start",
			Short: "Start the Synnax Windows service",
			Long:  "Start the Synnax Windows service.",
			Args:  cobra.NoArgs,
			RunE:  runStart,
		},
		&cobra.Command{
			Use:   "stop",
			Short: "Stop the Synnax Windows service",
			Long:  "Stop the Synnax Windows service and wait until it stops.",
			Args:  cobra.NoArgs,
			RunE:  runStop,
		},
		&cobra.Command{
			Use:   "status",
			Short: "Show Synnax Windows service status",
			Long: `Show the current status of the Synnax Windows service.

Displays whether the service is installed and running, along with configuration details like data directory, log file location, and listen address.`,
			Args: cobra.NoArgs,
			RunE: runStatus,
		},
	)
	return group
}

// syncFlagsToViper syncs changed cobra flags to viper. This is necessary because
// viper.BindPFlags doesn't properly pick up flag values after cobra parses them.
func syncFlagsToViper(cmd *cobra.Command, _ []string) error {
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return err
	}
	viper.SetDefault(cmdstart.FlagData, filepath.Join(ConfigDir(), "data"))
	viper.SetDefault(
		instrumentation.FlagLogFilePath,
		filepath.Join(ConfigDir(), "logs", "synnax.log"),
	)
	viper.SetDefault(cert.FlagCertsDir, filepath.Join(ConfigDir(), "certs"))
	return nil
}

func runInstall(c *cobra.Command, _ []string) error {
	c.SilenceUsage = true
	if err := install(); err != nil {
		return err
	}
	c.Printf("Windows service %s installed successfully.\n", name)
	c.Printf("Configuration saved to: %s\n", ConfigPath())
	c.Printf("Use 'synnax service start' to start %s.\n", name)
	return nil
}

func runUninstall(c *cobra.Command, _ []string) error {
	c.SilenceUsage = true
	if err := uninstall(); err != nil {
		return err
	}
	c.Printf("Windows service %s uninstalled successfully.\n", name)
	return nil
}

func runStart(c *cobra.Command, _ []string) error {
	c.SilenceUsage = true
	if err := Start(); err != nil {
		return err
	}
	c.Printf("%s started.\n", name)
	return nil
}

func runStop(c *cobra.Command, _ []string) error {
	c.SilenceUsage = true
	if err := Stop(); err != nil {
		return err
	}
	c.Printf("%s stopped.\n", name)
	return nil
}

func runStatus(c *cobra.Command, _ []string) error {
	c.SilenceUsage = true
	info, err := status()
	if err != nil {
		return err
	}
	FormatStatus(c, info)
	return nil
}

func FormatStatus(c *cobra.Command, info StatusInfo) {
	c.Printf("Service: %s\n", name)
	c.Printf("Version: %s\n", version.Get())
	if !info.Installed {
		c.Println("Status:  Not installed")
	} else if info.ProcessID > 0 {
		c.Printf("Status:  %s (PID: %d)\n", info.State, info.ProcessID)
	} else {
		c.Printf("Status:  %s\n", info.State)
		if info.State == "Stopped" &&
			(info.Win32ExitCode != 0 || info.ServiceSpecificExitCode != 0) {
			if info.ServiceSpecificExitCode != 0 {
				c.Printf(
					"Exit:    Service error code %d\n",
					info.ServiceSpecificExitCode,
				)
			} else {
				c.Printf("Exit:    Win32 error code %d\n", info.Win32ExitCode)
			}
		}
	}

	c.Println()
	c.Println("Configuration:")
	c.Printf("  Config file:  %s\n", info.ConfigPath)

	if info.ConfigError != nil {
		c.Printf("  (Error reading config: %v)\n", info.ConfigError)
	} else if info.DataDir != "" || info.LogFile != "" {
		if info.DataDir != "" {
			c.Printf("  Data dir:     %s\n", info.DataDir)
		}
		if info.LogFile != "" {
			c.Printf("  Log file:     %s\n", info.LogFile)
		}
		if info.CertsDir != "" {
			c.Printf("  Certs dir:    %s\n", info.CertsDir)
		}
		if info.Listen != "" {
			c.Printf("  Listen:       %s\n", info.Listen)
		}
		c.Printf("  Insecure:     %t\n", info.Insecure)
	} else {
		c.Println("  (No config file found)")
	}

	// Show recent warnings/errors when stopped (useful for debugging failures)
	if info.Installed && info.State == "Stopped" && info.LogFile != "" {
		lines, err := readRecentLogs(info.LogFile, 10)
		if err == nil && len(lines) > 0 {
			c.Println()
			c.Println("Recent warnings/errors:")
			for _, line := range lines {
				c.Printf("  %s\n", line)
			}
		}
	}
}
