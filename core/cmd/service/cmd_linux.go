// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build linux

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

const flagAutoStart = "auto-start"

// NewCmd returns the command group that manages the Core as a systemd service.
func NewCmd() *cobra.Command {
	group := &cobra.Command{
		Use:   "service",
		Short: "Manage the Core as a systemd service",
		Long: "Install, uninstall, start, stop, and show the Core as a systemd " +
			"service named synnax. The service runs as the synnax system user, " +
			"starts when the computer boots, and starts again after a failure.",
		Args: cobra.NoArgs,
	}
	installCmd := &cobra.Command{
		Use:   "install [flags]",
		Short: "Install the Core as a systemd service",
		Long: "Installs the Core as a systemd service. Run it as root. The command " +
			"creates the synnax system user and writes the start flags to " +
			"/etc/synnax/config.yaml, where you can change them later. It keeps " +
			"a config file that exists.",
		Example: "sudo synnax service install --listen synnax.example.com:9090 " +
			"--auto-cert --license-file /etc/synnax/license.lic",
		Args:    cobra.NoArgs,
		PreRunE: syncFlagsToViper,
		RunE: func(c *cobra.Command, _ []string) error {
			c.SilenceUsage = true
			if err := install(viper.GetBool(flagAutoStart)); err != nil {
				return err
			}
			c.Printf("Installed service %s.\n", name)
			c.Printf("Configuration saved to: %s\n", ConfigPath())
			c.Printf("Use 'synnax service start' to start %s.\n", name)
			return nil
		},
	}
	installCmd.Flags().Bool(
		flagAutoStart,
		true,
		"Start the service when the computer boots",
	)
	cmdstart.AddFlags(installCmd)
	group.AddCommand(
		installCmd,
		&cobra.Command{
			Use:   "uninstall",
			Short: "Uninstall the systemd service",
			Long:  "Stops the service and removes it. The config file stays.",
			Args:  cobra.NoArgs,
			RunE: func(c *cobra.Command, _ []string) error {
				c.SilenceUsage = true
				if err := uninstall(); err != nil {
					return err
				}
				c.Printf("Uninstalled service %s.\n", name)
				return nil
			},
		},
		&cobra.Command{
			Use:   "start",
			Short: "Start the systemd service",
			Args:  cobra.NoArgs,
			RunE: func(c *cobra.Command, _ []string) error {
				c.SilenceUsage = true
				if err := Start(); err != nil {
					return err
				}
				c.Printf("%s started.\n", name)
				return nil
			},
		},
		&cobra.Command{
			Use:   "stop",
			Short: "Stop the systemd service",
			Long:  "Stops the systemd service and waits until the Core exits.",
			Args:  cobra.NoArgs,
			RunE: func(c *cobra.Command, _ []string) error {
				c.SilenceUsage = true
				if err := Stop(); err != nil {
					return err
				}
				c.Printf("%s stopped.\n", name)
				return nil
			},
		},
		&cobra.Command{
			Use:   "status",
			Short: "Show the status of the systemd service",
			Long: "Shows whether the service is installed and running, its " +
				"configuration, and the last warnings and errors when it is not " +
				"running.",
			Args: cobra.NoArgs,
			RunE: func(c *cobra.Command, _ []string) error {
				c.SilenceUsage = true
				info, err := status()
				if err != nil {
					return err
				}
				FormatStatus(c, info)
				return nil
			},
		},
	)
	return group
}

func syncFlagsToViper(cmd *cobra.Command, _ []string) error {
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return err
	}
	viper.SetDefault(cmdstart.FlagData, filepath.Join(home, "data"))
	viper.SetDefault(
		instrumentation.FlagLogFilePath,
		filepath.Join(home, "logs", "synnax.log"),
	)
	viper.SetDefault(cert.FlagCertsDir, filepath.Join(home, "certs"))
	return nil
}

// FormatStatus prints the status of the service.
func FormatStatus(c *cobra.Command, info StatusInfo) {
	c.Printf("Service: %s\n", name)
	c.Printf("Version: %s\n", version.Get())
	switch {
	case !info.Installed:
		c.Println("Status:  Not installed")
	case info.ProcessID > 0:
		c.Printf("Status:  %s (PID: %d)\n", info.State, info.ProcessID)
	case info.ExitStatus != 0:
		c.Printf("Status:  %s\n", info.State)
		c.Printf("Exit:    Status %d\n", info.ExitStatus)
	default:
		c.Printf("Status:  %s\n", info.State)
	}
	c.Println()
	c.Println("Configuration:")
	c.Printf("  Config file:  %s\n", info.ConfigPath)
	switch {
	case info.ConfigError != nil:
		c.Printf("  (Error reading config: %v)\n", info.ConfigError)
	case info.DataDir == "" && info.LogFile == "":
		c.Println("  (No config file found)")
	default:
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
	}
	if !info.Installed || info.ProcessID > 0 || info.LogFile == "" {
		return
	}
	if lines, err := readRecentLogs(info.LogFile, 10); err == nil && len(lines) > 0 {
		c.Println()
		c.Println("Recent warnings/errors:")
		for _, line := range lines {
			c.Printf("  %s\n", line)
		}
	}
}
