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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/synnax/cmd/cert"
	cmdinst "github.com/synnaxlabs/synnax/cmd/instrumentation"
	cmdstart "github.com/synnaxlabs/synnax/cmd/start"
	"github.com/synnaxlabs/x/errors"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	shutdownTimeout = 30 * time.Second
	stopPollTimeout = 30 * time.Second
)

const (
	name        = "SynnaxCore"
	displayName = "Synnax Core"
	description = "Synnax telemetry engine for hardware systems"
)

// Is returns true if the current process is running as a Windows Service.
func Is() (bool, error) { return svc.IsWindowsService() }

func install(cfg Config) error {
	exePath, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "failed to get executable path")
	}
	if exePath, err = filepath.Abs(exePath); err != nil {
		return errors.Wrap(err, "failed to get absolute executable path")
	}

	// For services, always use absolute paths for directories. If the user specified a
	// relative path or didn't specify one (using the default), find the absolute path
	// by combining that path with the working directory.
	workDir, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "failed to get working directory")
	}
	dataPath := viper.GetString(cmdstart.FlagData)
	if !filepath.IsAbs(dataPath) {
		dataPath, err = filepath.Abs(filepath.Join(workDir, dataPath))
		if err != nil {
			return err
		}
		viper.Set(cmdstart.FlagData, filepath.Join(ConfigDir(), dataPath))
	}
	logPath := viper.GetString(cmdinst.FlagLogFilePath)
	if !filepath.IsAbs(logPath) {
		logPath, err = filepath.Abs(filepath.Join(workDir, logPath))
		if err != nil {
			return err
		}
		viper.Set(cmdinst.FlagLogFilePath, filepath.Join(ConfigDir(), logPath))
	}
	certsDir := viper.GetString(cert.FlagCertsDir)
	if !filepath.IsAbs(certsDir) {
		certsDir, err = filepath.Abs(filepath.Join(workDir, certsDir))
		if err != nil {
			return err
		}
		viper.Set(cert.FlagCertsDir, filepath.Join(ConfigDir(), certsDir))
	}
	if err = WriteConfig(); err != nil {
		return err
	}

	m, err := mgr.Connect()
	if err != nil {
		return errors.Wrap(err, "failed to connect to service manager (are you running as administrator?)")
	}
	defer func() {
		err = errors.Combine(err, errors.Wrap(m.Disconnect(), "failed to disconnect from service manager"))
	}()

	startType := lo.Ternary(
		cfg.AutoStart || cfg.DelayedStart,
		uint32(mgr.StartAutomatic),
		uint32(mgr.StartManual),
	)

	s, err := m.CreateService(name, exePath, mgr.Config{
		StartType:        startType,
		ErrorControl:     mgr.ErrorNormal,
		DisplayName:      displayName,
		Description:      description,
		DelayedAutoStart: cfg.DelayedStart,
	})
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, s.Close())
	}()

	if err = s.SetRecoveryActions([]mgr.RecoveryAction{
		{Type: mgr.ServiceRestart, Delay: 5 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 30 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 60 * time.Second},
	}, 86400); err != nil {
		return err
	}
	return eventlog.InstallAsEventCreate(name, eventlog.Error|eventlog.Warning|eventlog.Info)
}

func uninstall() error {
	m, err := mgr.Connect()
	if err != nil {
		return errors.Wrap(err, "failed to connect to service manager (are you running as administrator?)")
	}
	defer func() {
		err = errors.Combine(err, errors.Wrap(m.Disconnect(), "failed to disconnect from service manager"))
	}()

	s, err := m.OpenService(name)
	if err != nil {
		return errors.Wrapf(err, "service %s is not installed", name)
	}
	defer func() {
		err = errors.Combine(err, errors.Wrap(s.Close(), "failed to close service handle"))
	}()

	// Try to stop the service first if it's running.
	if status, qErr := s.Query(); qErr == nil && status.State != svc.Stopped {
		_, _ = s.Control(svc.Stop)
		for range 30 {
			time.Sleep(time.Second)
			if status, qErr = s.Query(); qErr != nil || status.State == svc.Stopped {
				break
			}
		}
	}

	if err = s.Delete(); err != nil {
		return errors.Wrap(err, "failed to delete service")
	}

	err = errors.Combine(err, errors.Wrap(eventlog.Remove(name), "failed to remove event log source"))
	return err
}

// start starts the Synnax Windows Service.
func start() error {
	m, err := mgr.Connect()
	if err != nil {
		return errors.Wrap(err, "failed to connect to service manager (are you running as administrator?)")
	}
	defer func() {
		err = errors.Combine(err, errors.Wrap(m.Disconnect(), "failed to disconnect from service manager"))
	}()

	s, err := m.OpenService(name)
	if err != nil {
		return errors.Wrapf(err, "service %s is not installed", name)
	}
	defer func() {
		err = errors.Combine(err, errors.Wrap(s.Close(), "failed to close service handle"))
	}()

	if err = s.Start(); err != nil {
		return errors.Wrap(err, "failed to start service")
	}
	return nil
}

func stop() error {
	m, err := mgr.Connect()
	if err != nil {
		return errors.Wrap(err, "failed to connect to service manager (are you running as administrator?)")
	}
	defer func() {
		err = errors.Combine(err, errors.Wrap(m.Disconnect(), "failed to disconnect from service manager"))
	}()

	s, err := m.OpenService(name)
	if err != nil {
		return errors.Wrapf(err, "service %s is not installed", name)
	}
	defer func() {
		err = errors.Combine(err, errors.Wrap(s.Close(), "failed to close service handle"))
	}()

	status, err := s.Query()
	if err != nil {
		return errors.Wrap(err, "failed to query service status")
	}
	if status.State == svc.Stopped {
		return nil
	}

	if _, err = s.Control(svc.Stop); err != nil {
		return errors.Wrap(err, "failed to stop service")
	}

	for range int(stopPollTimeout.Seconds()) {
		time.Sleep(time.Second)
		if status, err = s.Query(); err != nil {
			return errors.Wrap(err, "failed to query service status")
		}
		if status.State == svc.Stopped {
			return nil
		}
	}

	return errors.New("timeout waiting for service to stop")
}

// synnaxService implements svc.Handler for running Synnax as a Windows Service.
type synnaxService struct{ startServer func(context.Context) error }

// Execute is the main service control handler called by the Windows SCM.
func (s *synnaxService) Execute(
	_ []string,
	r <-chan svc.ChangeRequest,
	changes chan<- svc.Status,
) (bool, uint32) {
	const acceptedCmds = svc.AcceptStop | svc.AcceptShutdown

	elog, err := eventlog.Open(name)
	if err != nil {
		return false, 1
	}
	defer func() { _ = elog.Close() }()

	elog.Info(1, "Execute called, setting StartPending")
	changes <- svc.Status{State: svc.StartPending}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	elog.Info(1, "Starting server goroutine")
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.startServer(ctx)
	}()

	// Give the server a moment to fail during startup before declaring Running.
	// This allows early startup errors to be caught before we tell SCM we're ready.
	select {
	case err := <-errCh:
		if err != nil {
			elog.Error(1, fmt.Sprintf("server failed during startup: %v", err))
			changes <- svc.Status{State: svc.Stopped}
			return false, 1
		}
		// Server exited cleanly during startup (unusual but possible)
		elog.Info(1, "server exited during startup without error")
		changes <- svc.Status{State: svc.Stopped}
		return false, 0
	case <-time.After(2 * time.Second):
		// Server didn't fail immediately, assume it's starting up
	}

	changes <- svc.Status{State: svc.Running, Accepts: acceptedCmds}
	elog.Info(1, "Synnax service started, entering main loop")

	for {
		select {
		case err := <-errCh:
			if err != nil && !errors.Is(err, context.Canceled) {
				elog.Error(1, fmt.Sprintf("server returned error: %v", err))
				changes <- svc.Status{State: svc.Stopped}
				return false, 1
			}
			elog.Info(1, "server returned without error, stopping")
			changes <- svc.Status{State: svc.Stopped}
			return false, 0

		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				elog.Info(1, "stop requested, initiating graceful shutdown")
				changes <- svc.Status{State: svc.StopPending}
				cancel()

				select {
				case <-errCh:
					elog.Info(1, "stopped gracefully")
				case <-time.After(shutdownTimeout):
					elog.Warning(1, "graceful shutdown timed out")
				}

				changes <- svc.Status{State: svc.Stopped}
				return false, 0
			}
		}
	}
}

// Run runs Synnax as a Windows Service.
func Run() error {
	// Best-effort install of event log source. This often fails because the source was
	// already registered during service installation.
	if err := eventlog.InstallAsEventCreate(name, eventlog.Error|eventlog.Warning|eventlog.Info); err != nil {
		// Only warn if it's not the expected "already exists" error.
		if !strings.Contains(err.Error(), "already exists") {
			if elog, openErr := eventlog.Open(name); openErr == nil {
				_ = elog.Warning(1, fmt.Sprintf("failed to install event log source: %v", err))
				_ = elog.Close()
			}
		}
	}

	// Load configuration from the YAML config file.
	viper.SetConfigFile(ConfigPath())
	if err := viper.ReadInConfig(); err != nil {

		return errors.Wrap(err, "failed to read service configuration")
	}

	return svc.Run(name, &synnaxService{startServer: func(ctx context.Context) error {
		ins := cmdinst.Configure()
		defer cmdinst.Cleanup(ctx, ins)
		// TODO: connect elog and instrumentation somehow?
		cfg := cmdstart.GetCoreConfigFromViper(ins)
		return cmdstart.BootupCore(ctx, cfg)
	}})
}
