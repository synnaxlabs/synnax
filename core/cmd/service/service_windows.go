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
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/cmd/cert"
	"github.com/synnaxlabs/synnax/cmd/exit"
	cmdinst "github.com/synnaxlabs/synnax/cmd/instrumentation"
	"github.com/synnaxlabs/synnax/cmd/listener"
	cmdstart "github.com/synnaxlabs/synnax/cmd/start"
	"github.com/synnaxlabs/x/errors"
	signal "github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	startupTimeout   = 60 * time.Second
	shutdownTimeout  = 30 * time.Second
	stopTimeout      = shutdownTimeout + 15*time.Second
	stopPollInterval = 250 * time.Millisecond
	name             = "SynnaxCore"
)

// ConfigDir returns the directory where the service config file is stored.
func ConfigDir() string {
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = "C:\\ProgramData"
	}
	return filepath.Join(programData, "Synnax")
}

// ConfigPath returns the full path to the service config file.
func ConfigPath() string { return filepath.Join(ConfigDir(), "config.yaml") }

// Is returns true if the current process is running as a Windows Service.
func Is() (bool, error) { return svc.IsWindowsService() }

func install() (err error) {
	exePath, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "failed to get executable path")
	}
	if exePath, err = filepath.Abs(exePath); err != nil {
		return errors.Wrap(err, "failed to get absolute executable path")
	}

	if err = resolvePaths(); err != nil {
		return errors.Wrap(err, "failed to resolve the service paths")
	}
	if err = WriteConfig(); err != nil {
		return errors.Wrap(err, "failed to write config")
	}

	m, err := mgr.Connect()
	if err != nil {
		return errors.Wrap(
			err,
			"failed to connect to service manager (are you running as administrator?)",
		)
	}
	defer func() {
		err = errors.Combine(
			err,
			errors.Wrap(m.Disconnect(), "failed to disconnect from service manager"),
		)
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
		err = errors.Combine(
			err,
			errors.Wrap(s.Close(), "failed to close service handle"),
		)
	}()

	return errors.Wrap(s.SetRecoveryActions([]mgr.RecoveryAction{
		{Type: mgr.ServiceRestart, Delay: 5 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 30 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 60 * time.Second},
	}, 86400), "failed to set recovery actions")
}

func uninstall() (err error) {
	m, err := mgr.Connect()
	if err != nil {
		return errors.Wrap(
			err,
			"failed to connect to service manager (are you running as administrator?)",
		)
	}
	defer func() {
		err = errors.Combine(
			err,
			errors.Wrap(m.Disconnect(), "failed to disconnect from service manager"),
		)
	}()

	s, err := m.OpenService(name)
	if err != nil {
		return errors.Wrapf(err, "service %s is not installed", name)
	}
	defer func() {
		err = errors.Combine(
			err,
			errors.Wrapf(s.Close(), "failed to close %s handle", name),
		)
	}()
	return errors.Wrapf(s.Delete(), "failed to delete %s", name)
}

// Start starts the Core service.
func Start() (err error) {
	m, err := mgr.Connect()
	if err != nil {
		return errors.Wrap(
			err,
			"failed to connect to service manager (are you running as administrator?)",
		)
	}
	defer func() {
		err = errors.Combine(
			err,
			errors.Wrap(m.Disconnect(), "failed to disconnect from service manager"),
		)
	}()

	s, err := m.OpenService(name)
	if err != nil {
		return errors.Wrapf(err, "service %s is not installed", name)
	}
	defer func() {
		err = errors.Combine(
			err,
			errors.Wrapf(s.Close(), "failed to close %s handle", name),
		)
	}()
	return errors.Wrapf(s.Start(), "failed to start %s", name)
}

// Stop stops the Core service and waits until the Core exits. It does nothing when the
// service is not running.
func Stop() (err error) {
	m, err := mgr.Connect()
	if err != nil {
		return errors.Wrap(
			err,
			"failed to connect to service manager (are you running as administrator?)",
		)
	}
	defer func() {
		err = errors.Combine(
			err,
			errors.Wrap(m.Disconnect(), "failed to disconnect from service manager"),
		)
	}()

	s, err := m.OpenService(name)
	if err != nil {
		return errors.Wrapf(err, "service %s is not installed", name)
	}
	defer func() {
		err = errors.Combine(
			err,
			errors.Wrapf(s.Close(), "failed to close %s handle", name),
		)
	}()

	st, err := s.Query()
	if err != nil {
		return errors.Wrap(err, "failed to query service status")
	}
	if st.State == svc.Stopped {
		return nil
	}
	if st.State != svc.StopPending {
		if st, err = s.Control(svc.Stop); err != nil {
			return errors.Wrap(err, "failed to stop service")
		}
	}
	deadline := time.Now().Add(stopTimeout)
	for st.State != svc.Stopped {
		if time.Now().After(deadline) {
			return errors.Newf("%s did not stop within %s", name, stopTimeout)
		}
		time.Sleep(stopPollInterval)
		if st, err = s.Query(); err != nil {
			return errors.Wrap(err, "failed to query service status")
		}
	}
	return nil
}

// Lookup returns the Core service on this host, or false when none is installed.
func Lookup() (_ Installation, _ bool, err error) {
	m, err := mgr.Connect()
	if err != nil {
		return Installation{}, false, errors.Wrap(
			err,
			"failed to connect to service manager (are you running as administrator?)",
		)
	}
	defer func() {
		err = errors.Combine(
			err,
			errors.Wrap(m.Disconnect(), "failed to disconnect from service manager"),
		)
	}()
	s, err := m.OpenService(name)
	if errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
		return Installation{}, false, nil
	}
	if err != nil {
		return Installation{}, false, errors.Wrapf(err, "failed to open %s", name)
	}
	defer func() {
		err = errors.Combine(
			err,
			errors.Wrapf(s.Close(), "failed to close %s handle", name),
		)
	}()
	cfg, err := s.Config()
	if err != nil {
		return Installation{}, false, errors.Wrapf(err, "failed to read %s", name)
	}
	v := viper.New()
	v.SetConfigFile(ConfigPath())
	if err = v.ReadInConfig(); err != nil {
		return Installation{}, false, errors.Wrap(
			err,
			"failed to read service configuration",
		)
	}
	return Installation{
		Binary: binaryPath(cfg.BinaryPathName),
		Data:   v.GetString(cmdstart.FlagData),
	}, true, nil
}

// binaryPath returns the executable of a service command line, which quotes a path with
// spaces.
func binaryPath(command string) string {
	if rest, ok := strings.CutPrefix(command, `"`); ok {
		path, _, _ := strings.Cut(rest, `"`)
		return path
	}
	path, _, _ := strings.Cut(command, " ")
	return path
}

// StatusInfo contains information about the service status and configuration.
type StatusInfo struct {
	Installed               bool
	State                   string
	ProcessID               uint32
	Win32ExitCode           uint32
	ServiceSpecificExitCode uint32
	ConfigPath              string
	DataDir                 string
	LogFile                 string
	CertsDir                string
	Listen                  string
	Insecure                bool
	ConfigError             error
}

func stateToString(state svc.State) string {
	switch state {
	case svc.Stopped:
		return "Stopped"
	case svc.StartPending:
		return "Starting"
	case svc.StopPending:
		return "Stopping"
	case svc.Running:
		return "Running"
	case svc.ContinuePending:
		return "Continuing"
	case svc.PausePending:
		return "Pausing"
	case svc.Paused:
		return "Paused"
	default:
		return "Unknown"
	}
}

func status() (_ StatusInfo, err error) {
	info := StatusInfo{
		ConfigPath: ConfigPath(),
	}

	// Try to read config file for paths
	if _, err = os.Stat(info.ConfigPath); err == nil {
		v := viper.New()
		v.SetConfigFile(info.ConfigPath)
		if err := v.ReadInConfig(); err != nil {
			info.ConfigError = err
		} else {
			info.DataDir = v.GetString(cmdstart.FlagData)
			info.LogFile = v.GetString(cmdinst.FlagLogFilePath)
			info.CertsDir = v.GetString(cert.FlagCertsDir)
			info.Listen = v.GetString(listener.FlagListen)
			info.Insecure = v.GetBool(cmdstart.FlagInsecure)
		}
	}

	m, err := mgr.Connect()
	if err != nil {
		return info, errors.Wrap(err, "failed to connect to service manager")
	}
	defer func() {
		err = errors.Combine(
			err,
			errors.Wrap(m.Disconnect(), "failed to disconnect from service manager"),
		)
	}()

	s, err := m.OpenService(name)
	if err != nil {
		// Service not installed
		return info, nil
	}
	defer func() {
		err = errors.Combine(
			err,
			errors.Wrapf(s.Close(), "failed to close %s handle", name),
		)
	}()

	info.Installed = true

	st, err := s.Query()
	if err != nil {
		return info, errors.Wrap(err, "failed to query service status")
	}

	info.State = stateToString(st.State)
	info.ProcessID = st.ProcessId
	info.Win32ExitCode = st.Win32ExitCode
	info.ServiceSpecificExitCode = st.ServiceSpecificExitCode

	return info, nil
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
		cfg, err := cmdstart.GetCoreConfigFromViper(s.ins)
		if err != nil {
			return err
		}
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
		return true, uint32(exit.Code(err))
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
