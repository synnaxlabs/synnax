// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build windows

package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	serviceShutdownTimeout = 30 * time.Second
)

// synnaxService implements svc.Handler for running Synnax as a Windows Service.
type synnaxService struct{}

// Execute is the main service control handler called by the Windows Service Control Manager.
func (s *synnaxService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const acceptedCmds = svc.AcceptStop | svc.AcceptShutdown

	// Open event log for logging service events
	elog, err := eventlog.Open(serviceName)
	if err != nil {
		return false, 1
	}
	defer elog.Close()

	// Tell SCM we're starting
	changes <- svc.Status{State: svc.StartPending}

	// Parse service arguments and apply to viper
	if err := parseServiceArgs(args); err != nil {
		elog.Error(1, fmt.Sprintf("Failed to parse service arguments: %v", err))
		changes <- svc.Status{State: svc.Stopped}
		return false, 1
	}

	// Create a cancellable context for the server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the server in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- startServer(ctx)
	}()

	// Tell SCM we're running
	changes <- svc.Status{State: svc.Running, Accepts: acceptedCmds}
	elog.Info(1, "Synnax service started successfully")

	// Main service control loop
	for {
		select {
		case err := <-errCh:
			if err != nil && !errors.Is(err, context.Canceled) {
				elog.Error(1, fmt.Sprintf("Server error: %v", err))
				changes <- svc.Status{State: svc.Stopped}
				return false, 1
			}
			// Server stopped normally
			changes <- svc.Status{State: svc.Stopped}
			return false, 0

		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus

			case svc.Stop, svc.Shutdown:
				elog.Info(1, "Synnax service stop requested, initiating graceful shutdown")
				changes <- svc.Status{State: svc.StopPending}

				// Cancel context to trigger graceful shutdown
				cancel()

				// Wait for server to stop with timeout
				select {
				case <-errCh:
					elog.Info(1, "Synnax service stopped gracefully")
				case <-time.After(serviceShutdownTimeout):
					elog.Warning(1, "Graceful shutdown timed out, forcing stop")
				}

				changes <- svc.Status{State: svc.Stopped}
				return false, 0
			}
		}
	}
}

// parseServiceArgs parses command-line arguments passed to the service and applies them to viper.
func parseServiceArgs(args []string) error {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			continue
		}

		key := strings.TrimPrefix(arg, "--")
		var value string

		// Check if value is in the same argument (--key=value)
		if idx := strings.Index(key, "="); idx != -1 {
			value = key[idx+1:]
			key = key[:idx]
		} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			// Value is in the next argument
			i++
			value = args[i]
		} else {
			// Boolean flag
			value = "true"
		}

		viper.Set(key, value)
	}
	return nil
}

// runAsWindowsService runs Synnax as a Windows Service.
func runAsWindowsService() error {
	// Install event log source if not already installed
	_ = eventlog.InstallAsEventCreate(serviceName, eventlog.Error|eventlog.Warning|eventlog.Info)

	return svc.Run(serviceName, &synnaxService{})
}

// serviceInstall installs Synnax as a Windows Service.
func serviceInstall(c *cobra.Command, _ []string) error {
	// Get the path to the current executable
	exePath, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "failed to get executable path")
	}
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return errors.Wrap(err, "failed to get absolute executable path")
	}

	// Open service manager
	m, err := mgr.Connect()
	if err != nil {
		return errors.Wrap(err, "failed to connect to service manager (are you running as administrator?)")
	}
	defer m.Disconnect()

	// Check if service already exists
	s, err := m.OpenService(serviceName)
	if err == nil {
		s.Close()
		return errors.Newf("service %s already exists; use 'synnax service uninstall' first", serviceName)
	}

	// Build service arguments from flags
	serviceArgs := buildServiceArgs(c)

	// Determine start type
	startType := uint32(mgr.StartAutomatic)
	if !viper.GetBool(serviceAutoStartFlag) {
		startType = uint32(mgr.StartManual)
	}

	// Create the service
	config := mgr.Config{
		StartType:    startType,
		ErrorControl: mgr.ErrorNormal,
		DisplayName:  serviceDisplayName,
		Description:  serviceDescription,
	}

	s, err = m.CreateService(serviceName, exePath, config, serviceArgs...)
	if err != nil {
		return errors.Wrap(err, "failed to create service")
	}
	defer s.Close()

	// Set delayed start if requested
	if viper.GetBool(serviceDelayedStartFlag) {
		err = s.UpdateConfig(mgr.Config{
			StartType:        uint32(mgr.StartAutomatic),
			ErrorControl:     mgr.ErrorNormal,
			DisplayName:      serviceDisplayName,
			Description:      serviceDescription,
			DelayedAutoStart: true,
		})
		if err != nil {
			c.PrintErrf("Warning: failed to set delayed start: %v\n", err)
		}
	}

	// Configure recovery actions (restart on failure)
	err = s.SetRecoveryActions(
		[]mgr.RecoveryAction{
			{Type: mgr.ServiceRestart, Delay: 5 * time.Second},
			{Type: mgr.ServiceRestart, Delay: 30 * time.Second},
			{Type: mgr.ServiceRestart, Delay: 60 * time.Second},
		},
		86400, // Reset failure count after 24 hours
	)
	if err != nil {
		c.PrintErrf("Warning: failed to set recovery actions: %v\n", err)
	}

	// Install event log source
	err = eventlog.InstallAsEventCreate(serviceName, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		c.PrintErrf("Warning: failed to install event log source: %v\n", err)
	}

	c.Printf("Service %s installed successfully.\n", serviceName)
	c.Println("Use 'synnax service start' or 'net start SynnaxServer' to start the service.")
	return nil
}

// buildServiceArgs builds the command-line arguments to pass to the service.
func buildServiceArgs(cmd *cobra.Command) []string {
	var args []string

	// Add server configuration flags
	if v := viper.GetString(listenFlag); v != "" && v != "localhost:9090" {
		args = append(args, "--"+listenFlag, v)
	}

	if v := viper.GetString(dataFlag); v != "" {
		args = append(args, "--"+dataFlag, v)
	} else {
		// Default to ProgramData location for service
		programData := os.Getenv("ProgramData")
		if programData == "" {
			programData = `C:\ProgramData`
		}
		defaultDataDir := filepath.Join(programData, "Synnax", "data")
		args = append(args, "--"+dataFlag, defaultDataDir)
	}

	if viper.GetBool(insecureFlag) {
		args = append(args, "--"+insecureFlag)
	}

	if v := viper.GetString(usernameFlag); v != "" && v != "synnax" {
		args = append(args, "--"+usernameFlag, v)
	}

	if v := viper.GetString(passwordFlag); v != "" && v != "seldon" {
		args = append(args, "--"+passwordFlag, v)
	}

	if viper.GetBool(autoCertFlag) {
		args = append(args, "--"+autoCertFlag)
	}

	if viper.GetBool(noDriverFlag) {
		args = append(args, "--"+noDriverFlag)
	}

	if peers := viper.GetStringSlice(peersFlag); len(peers) > 0 {
		args = append(args, "--"+peersFlag, strings.Join(peers, ","))
	}

	if integrations := viper.GetStringSlice(enableIntegrationsFlag); len(integrations) > 0 {
		args = append(args, "--"+enableIntegrationsFlag, strings.Join(integrations, ","))
	}

	if integrations := viper.GetStringSlice(disableIntegrationsFlag); len(integrations) > 0 {
		args = append(args, "--"+disableIntegrationsFlag, strings.Join(integrations, ","))
	}

	return args
}

// serviceUninstall removes the Synnax Windows Service.
func serviceUninstall(c *cobra.Command, _ []string) error {
	m, err := mgr.Connect()
	if err != nil {
		return errors.Wrap(err, "failed to connect to service manager (are you running as administrator?)")
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return errors.Wrapf(err, "service %s is not installed", serviceName)
	}
	defer s.Close()

	// Try to stop the service first
	status, err := s.Query()
	if err == nil && status.State != svc.Stopped {
		c.Println("Stopping service...")
		_, err = s.Control(svc.Stop)
		if err != nil {
			c.PrintErrf("Warning: failed to stop service: %v\n", err)
		} else {
			// Wait for service to stop
			for i := 0; i < 30; i++ {
				time.Sleep(time.Second)
				status, err = s.Query()
				if err != nil || status.State == svc.Stopped {
					break
				}
			}
		}
	}

	// Delete the service
	err = s.Delete()
	if err != nil {
		return errors.Wrap(err, "failed to delete service")
	}

	// Remove event log source
	_ = eventlog.Remove(serviceName)

	c.Printf("Service %s uninstalled successfully.\n", serviceName)
	return nil
}

// serviceStart starts the Synnax Windows Service.
func serviceStart(c *cobra.Command, _ []string) error {
	m, err := mgr.Connect()
	if err != nil {
		return errors.Wrap(err, "failed to connect to service manager (are you running as administrator?)")
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return errors.Wrapf(err, "service %s is not installed", serviceName)
	}
	defer s.Close()

	err = s.Start()
	if err != nil {
		return errors.Wrap(err, "failed to start service")
	}

	c.Printf("Service %s started.\n", serviceName)
	return nil
}

// serviceStop stops the Synnax Windows Service.
func serviceStop(c *cobra.Command, _ []string) error {
	m, err := mgr.Connect()
	if err != nil {
		return errors.Wrap(err, "failed to connect to service manager (are you running as administrator?)")
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return errors.Wrapf(err, "service %s is not installed", serviceName)
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		return errors.Wrap(err, "failed to query service status")
	}

	if status.State == svc.Stopped {
		c.Printf("Service %s is already stopped.\n", serviceName)
		return nil
	}

	_, err = s.Control(svc.Stop)
	if err != nil {
		return errors.Wrap(err, "failed to stop service")
	}

	// Wait for service to stop
	c.Print("Stopping service")
	for i := 0; i < 30; i++ {
		time.Sleep(time.Second)
		c.Print(".")
		status, err = s.Query()
		if err != nil {
			return errors.Wrap(err, "failed to query service status")
		}
		if status.State == svc.Stopped {
			c.Println(" stopped.")
			return nil
		}
	}

	return errors.New("timeout waiting for service to stop")
}
