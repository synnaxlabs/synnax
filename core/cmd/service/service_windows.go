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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/cmd/cert"
	cmdinst "github.com/synnaxlabs/synnax/cmd/instrumentation"
	cmdstart "github.com/synnaxlabs/synnax/cmd/start"
	"github.com/synnaxlabs/x/errors"
	signal "github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	startupTimeout  = 60 * time.Second
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
		viper.Set(cmdstart.FlagData, dataPath)
	}
	logPath := viper.GetString(cmdinst.FlagLogFilePath)
	if !filepath.IsAbs(logPath) {
		logPath, err = filepath.Abs(filepath.Join(workDir, logPath))
		if err != nil {
			return errors.Wrap(err, "failed to get absolute log path")
		}
		viper.Set(cmdinst.FlagLogFilePath, logPath)
	}
	certsDir := viper.GetString(cert.FlagCertsDir)
	if !filepath.IsAbs(certsDir) {
		certsDir, err = filepath.Abs(filepath.Join(workDir, certsDir))
		if err != nil {
			return errors.Wrap(err, "failed to get absolute certs path")
		}
		viper.Set(cert.FlagCertsDir, certsDir)
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
type synnaxService struct {
	ins alamos.Instrumentation
}

// Execute is the main service control handler called by the Windows SCM.
func (s *synnaxService) Execute(
	_ []string,
	r <-chan svc.ChangeRequest,
	changes chan<- svc.Status,
) (bool, uint32) {
	const accepts = svc.AcceptStop | svc.AcceptShutdown
	defer func() { changes <- svc.Status{State: svc.Stopped} }()
	status := svc.Status{State: svc.StartPending}
	changes <- status

	sCtx, cancel := signal.WithCancel(
		context.Background(),
		signal.WithInstrumentation(s.ins),
	)
	defer cancel()

	onServerStarted := make(chan struct{}, 1)
	sCtx.Go(func(ctx context.Context) error {
		cfg := cmdstart.GetCoreConfigFromViper(s.ins)
		return cmdstart.BootupCore(ctx, onServerStarted, cfg)
	}, signal.CancelOnFail())

	startupTimer := time.After(startupTimeout)
o:
	for {
		select {
		case <-onServerStarted:
			status = svc.Status{State: svc.Running, Accepts: accepts}
			startupTimer = nil
			changes <- status
		case <-startupTimer:
			s.ins.L.Error("service startup timed out")
			cancel()
			break o
		case c := <-r:
			switch c.Cmd {
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				cancel()
				break o
			case svc.Interrogate:
				changes <- status
			default:
				s.ins.L.Warn("unhandled service control command", zap.Uint32("cmd", uint32(c.Cmd)))
			}
		case <-sCtx.Stopped():
			s.ins.L.Error("service shutdown unexpectedly")
			break o
		}
	}
	select {
	case <-sCtx.Stopped():
	case <-time.After(shutdownTimeout):
		s.ins.L.Error("service shutdown timed out")
		return false, 1
	}
	if err := sCtx.Wait(); errors.Skip(err, context.Canceled) != nil {
		s.ins.L.Error("service failed", zap.Error(err))
		return false, 1
	}
	return false, 0
}

// Run runs Synnax as a Windows Service.
func Run() error {
	viper.SetConfigFile(ConfigPath())
	if err := viper.ReadInConfig(); err != nil {
		return errors.Wrap(err, "failed to read service configuration")
	}
	ins := cmdinst.Configure()
	defer cmdinst.Cleanup(context.Background(), ins)
	return svc.Run(name, &synnaxService{ins: ins})
}
