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
	"os"
	"path/filepath"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/synnax/cmd/cert"
	cmdinst "github.com/synnaxlabs/synnax/cmd/instrumentation"
	cmdstart "github.com/synnaxlabs/synnax/cmd/start"
	"github.com/synnaxlabs/x/errors"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	shutdownTimeout = 30 * time.Second
	name            = "SynnaxCore"
)

// Is returns true if the current process is running as a Windows Service.
func Is() (bool, error) { return svc.IsWindowsService() }

func install() error {
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
			return errors.Wrap(err, "failed to get absolute data path")
		}
		viper.Set(cmdstart.FlagData, filepath.Join(ConfigDir(), dataPath))
	}
	logPath := viper.GetString(cmdinst.FlagLogFilePath)
	if !filepath.IsAbs(logPath) {
		logPath, err = filepath.Abs(filepath.Join(workDir, logPath))
		if err != nil {
			return errors.Wrap(err, "failed to get absolute log path")
		}
		viper.Set(cmdinst.FlagLogFilePath, filepath.Join(ConfigDir(), logPath))
	}
	certsDir := viper.GetString(cert.FlagCertsDir)
	if !filepath.IsAbs(certsDir) {
		certsDir, err = filepath.Abs(filepath.Join(workDir, certsDir))
		if err != nil {
			return errors.Wrap(err, "failed to get absolute certs path")
		}
		viper.Set(cert.FlagCertsDir, filepath.Join(ConfigDir(), certsDir))
	}
	if err = WriteConfig(); err != nil {
		return errors.Wrap(err, "failed to write config")
	}

	m, err := mgr.Connect()
	if err != nil {
		return errors.Wrap(err, "failed to connect to service manager (are you running as administrator?)")
	}
	defer func() {
		err = errors.Combine(err, errors.Wrap(m.Disconnect(), "failed to disconnect from service manager"))
	}()

	autoStart := viper.GetBool(flagAutoStart)
	delayedStart := viper.GetBool(flagDelayedStart)

	startType := lo.Ternary(
		autoStart || delayedStart,
		uint32(mgr.StartAutomatic),
		uint32(mgr.StartManual),
	)

	s, err := m.CreateService(name, exePath, mgr.Config{
		StartType:        startType,
		ErrorControl:     mgr.ErrorNormal,
		DisplayName:      "Synnax Core",
		Description:      "Synnax telemetry engine for hardware systems",
		DelayedAutoStart: delayedStart,
	})
	if err != nil {
		return errors.Wrap(err, "failed to create service")
	}
	defer func() {
		err = errors.Combine(err, errors.Wrap(s.Close(), "failed to close service handle"))
	}()

	return errors.Wrap(s.SetRecoveryActions([]mgr.RecoveryAction{
		{Type: mgr.ServiceRestart, Delay: 5 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 30 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 60 * time.Second},
	}, 86400), "failed to set recovery actions")
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
		err = errors.Combine(err, errors.Wrapf(s.Close(), "failed to close %s handle", name))
	}()
	return errors.Wrapf(s.Delete(), "failed to delete %s", name)

}

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
		err = errors.Combine(err, errors.Wrapf(s.Close(), "failed to close %s handle", name))
	}()
	return errors.Wrapf(s.Start(), "failed to start %s", name)
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
		err = errors.Combine(err, errors.Wrapf(s.Close(), "failed to close %s handle", name))
	}()

	if _, err = s.Control(svc.Stop); err != nil {
		return errors.Wrap(err, "failed to stop service")
	}
	return nil
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

	changes <- svc.Status{State: svc.StartPending}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.startServer(ctx)
	}()

	// Give the server a moment to fail during startup before declaring Running.
	// This allows early startup errors to be caught before we tell SCM we're ready.
	select {
	case err := <-errCh:
		if err != nil {
			changes <- svc.Status{State: svc.Stopped}
			return false, 1
		}
		// Server exited cleanly during startup (unusual but possible)
		changes <- svc.Status{State: svc.Stopped}
		return false, 0
	case <-time.After(2 * time.Second):
		// Server didn't fail immediately, assume it's starting up
	}

	changes <- svc.Status{State: svc.Running, Accepts: acceptedCmds}

	for {
		select {
		case err := <-errCh:
			if err != nil && !errors.Is(err, context.Canceled) {
				changes <- svc.Status{State: svc.Stopped}
				return false, 1
			}
			changes <- svc.Status{State: svc.Stopped}
			return false, 0

		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				cancel()

				select {
				case <-errCh:
				case <-time.After(shutdownTimeout):
				}

				changes <- svc.Status{State: svc.Stopped}
				return false, 0
			}
		}
	}
}

// Run runs Synnax as a Windows Service.
func Run() error {
	viper.SetConfigFile(ConfigPath())
	if err := viper.ReadInConfig(); err != nil {
		return errors.Wrap(err, "failed to read service configuration")
	}
	return svc.Run(name, &synnaxService{startServer: func(ctx context.Context) error {
		ins := cmdinst.Configure()
		defer cmdinst.Cleanup(ctx, ins)
		cfg := cmdstart.GetCoreConfigFromViper(ins)
		return cmdstart.BootupCore(ctx, cfg)
	}})
}
