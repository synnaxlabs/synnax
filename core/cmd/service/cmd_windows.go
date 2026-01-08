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
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

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

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Synnax Windows service status",
	Long: `Show the current status of the Synnax Windows service.

Displays whether the service is installed and running, along with configuration details like data directory, log file location, and listen address.`,
	Args: cobra.NoArgs,
	RunE: runStatus,
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
	serviceCmd.AddCommand(statusCmd)
	return viper.BindPFlags(cmd.Flags())
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
	if err := start(); err != nil {
		return err
	}
	c.Printf("%s started.\n", name)
	return nil
}

func runStop(c *cobra.Command, _ []string) error {
	c.SilenceUsage = true
	if err := stop(); err != nil {
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

	c.Printf("Service: %s\n", name)
	if !info.Installed {
		c.Println("Status:  Not installed")
	} else if info.ProcessID > 0 {
		c.Printf("Status:  %s (PID: %d)\n", info.State, info.ProcessID)
	} else {
		c.Printf("Status:  %s\n", info.State)
		if info.State == "Stopped" && (info.Win32ExitCode != 0 || info.ServiceSpecificExitCode != 0) {
			if info.ServiceSpecificExitCode != 0 {
				c.Printf("Exit:    Service error code %d\n", info.ServiceSpecificExitCode)
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

	return nil
}

// logEntry represents a parsed JSON log entry from zap.
type logEntry struct {
	Level   string  `json:"level"`
	TS      float64 `json:"ts"`
	Logger  string  `json:"logger"`
	Caller  string  `json:"caller"`
	Msg     string  `json:"msg"`
	Error   string  `json:"error"`
}

// ANSI color codes for log levels.
const (
	colorReset  = "\033[0m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorBoldRed = "\033[1;31m"
)

// logLevelInfo contains display information for each log level.
type logLevelInfo struct {
	color string
	label string
}

// importantLogLevels maps log levels to their display info.
var importantLogLevels = map[string]logLevelInfo{
	"warn":   {color: colorYellow, label: "WARN"},
	"error":  {color: colorRed, label: "ERROR"},
	"fatal":  {color: colorBoldRed, label: "FATAL"},
	"panic":  {color: colorBoldRed, label: "PANIC"},
	"dpanic": {color: colorBoldRed, label: "DPANIC"},
}

// readRecentLogs reads the log file and returns formatted important log entries.
func readRecentLogs(filePath string, maxEntries int) ([]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []logEntry
	scanner := bufio.NewScanner(f)
	// Increase buffer size for long log lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		var entry logEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip non-JSON lines
		}
		if _, ok := importantLogLevels[entry.Level]; ok {
			entries = append(entries, entry)
			if len(entries) > maxEntries {
				entries = entries[1:]
			}
		}
	}

	// Format entries for display
	var lines []string
	for _, e := range entries {
		ts := time.Unix(int64(e.TS), int64((e.TS-float64(int64(e.TS)))*1e9))
		info := importantLogLevels[e.Level]
		msg := e.Msg
		if e.Error != "" {
			msg += ": " + e.Error
		}
		line := ts.Format("2006-01-02 15:04:05") + " " + info.color + info.label + colorReset + " " + msg
		lines = append(lines, line)
	}
	return lines, scanner.Err()
}
