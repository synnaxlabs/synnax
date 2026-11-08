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
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/synnax/cmd/cert"
	"github.com/synnaxlabs/synnax/cmd/exit"
	cmdinst "github.com/synnaxlabs/synnax/cmd/instrumentation"
	"github.com/synnaxlabs/synnax/cmd/listener"
	cmdstart "github.com/synnaxlabs/synnax/cmd/start"
	"github.com/synnaxlabs/x/errors"
)

const (
	name     = "synnax"
	unitPath = "/etc/systemd/system/synnax.service"
	// home is the home directory of the service user, which systemd creates as the
	// state
	// directory of the service.
	home = "/var/lib/synnax"
)

// ConfigDir returns the directory where the service config file is stored.
func ConfigDir() string { return "/etc/synnax" }

// ConfigPath returns the full path to the service config file.
func ConfigPath() string { return filepath.Join(ConfigDir(), "config.yaml") }

// unit is the systemd unit of the service. systemd does not start the Core again after
// an error that only a person can fix. LimitRTPRIO lets Arc run on a real-time thread.
const unit = `[Unit]
Description=Synnax Core
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
User=synnax
Group=synnax
StateDirectory=synnax
WorkingDirectory=%s
ExecStart="%s" start --config "%s"
KillSignal=SIGINT
Restart=always
RestartSec=5s
RestartPreventExitStatus=%d %d
LimitNOFILE=65536
LimitRTPRIO=99

[Install]
WantedBy=multi-user.target
`

// run runs the program with args and returns its output, or an error with its standard
// error.
func run(program string, args ...string) (string, error) {
	var stderr bytes.Buffer
	cmd := exec.Command(program, args...)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", errors.Wrapf(
			err,
			"%s %s failed: %s",
			program,
			strings.Join(args, " "),
			strings.TrimSpace(stderr.String()),
		)
	}
	return string(out), nil
}

// show returns the given properties of the service.
func show(properties ...string) (map[string]string, error) {
	out, err := run(
		"systemctl",
		"show",
		name,
		"--property="+strings.Join(properties, ","),
	)
	if err != nil {
		return nil, err
	}
	values := make(map[string]string, len(properties))
	for line := range strings.Lines(out) {
		if key, value, ok := strings.Cut(strings.TrimSpace(line), "="); ok {
			values[key] = value
		}
	}
	return values, nil
}

func install(autoStart bool) error {
	if os.Geteuid() != 0 {
		return errors.New("run synnax service install as root")
	}
	exePath, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "failed to get executable path")
	}
	if exePath, err = filepath.EvalSymlinks(exePath); err != nil {
		return errors.Wrap(err, "failed to resolve executable path")
	}
	if err = resolvePaths(); err != nil {
		return errors.Wrap(err, "failed to resolve the service paths")
	}
	group, err := createUser()
	if err != nil {
		return err
	}
	if err = WriteConfig(); err != nil {
		return errors.Wrap(err, "failed to write config")
	}
	if err = os.Chown(ConfigPath(), 0, group); err != nil {
		return errors.Wrap(err, "failed to give the service user the config")
	}
	if err = os.WriteFile(
		unitPath,
		fmt.Appendf(
			nil,
			unit,
			home,
			exePath,
			ConfigPath(),
			exit.DataRefused,
			exit.Config,
		),
		0o644,
	); err != nil {
		return errors.Wrap(err, "failed to write the systemd unit")
	}
	if _, err = run("systemctl", "daemon-reload"); err != nil || !autoStart {
		return err
	}
	_, err = run("systemctl", "enable", name)
	return err
}

// createUser creates the system user that runs the service when it does not exist, and
// returns the ID of its group.
func createUser() (int, error) {
	u, err := user.Lookup(name)
	if errors.As(err, new(user.UnknownUserError)) {
		if out, err := exec.Command(
			"useradd",
			"--system",
			"--user-group",
			"--home-dir", home,
			"--shell", "/usr/sbin/nologin",
			name,
		).CombinedOutput(); err != nil {
			return 0, errors.Wrapf(
				err,
				"failed to create the %s user: %s",
				name,
				strings.TrimSpace(string(out)),
			)
		}
		u, err = user.Lookup(name)
	}
	if err != nil {
		return 0, errors.Wrapf(err, "failed to look up the %s user", name)
	}
	return strconv.Atoi(u.Gid)
}

func uninstall() error {
	if _, err := os.Stat(unitPath); errors.Is(err, os.ErrNotExist) {
		return errors.Newf("service %s is not installed", name)
	} else if err != nil {
		return err
	}
	if _, err := run("systemctl", "disable", "--now", name); err != nil {
		return err
	}
	if err := os.Remove(unitPath); err != nil {
		return errors.Wrap(err, "failed to remove the systemd unit")
	}
	_, err := run("systemctl", "daemon-reload")
	return err
}

// Start starts the Core service.
func Start() error {
	_, err := run("systemctl", "start", name)
	return err
}

// Stop stops the Core service and waits until the Core exits. It does nothing when the
// service is not running.
func Stop() error {
	_, err := run("systemctl", "stop", name)
	return err
}

// Lookup returns the Core service on this host, or false when none is installed or
// systemd does not run. It reads the data directory from the flags of the service
// command, or from the config file that the command names.
func Lookup() (Installation, bool, error) {
	offline, err := systemdOffline()
	if errors.Is(err, exec.ErrNotFound) || (err == nil && offline) {
		return Installation{}, false, nil
	}
	if err != nil {
		return Installation{}, false, err
	}
	props, err := show("LoadState", "WorkingDirectory")
	if err != nil {
		return Installation{}, false, err
	}
	if props["LoadState"] != "loaded" {
		return Installation{}, false, nil
	}
	binary, args, err := execStart()
	if err != nil {
		return Installation{}, false, err
	}
	data, err := dataDir(args, props["WorkingDirectory"])
	if err != nil {
		return Installation{}, false, err
	}
	return Installation{Binary: binary, Data: data}, true, nil
}

// systemdOffline reports whether systemd does not manage this host, such as in a
// container.
func systemdOffline() (bool, error) {
	out, err := exec.Command("systemctl", "is-system-running").Output()
	// The command exits with an error in every state but running.
	if err != nil && !errors.As(err, new(*exec.ExitError)) {
		return false, err
	}
	return strings.TrimSpace(string(out)) == "offline", nil
}

// execStart returns the binary and the arguments of the service command. It reads them
// with busctl, because systemctl show joins the arguments with spaces.
func execStart() (string, []string, error) {
	out, err := run(
		"busctl",
		"get-property",
		"--json=short",
		"org.freedesktop.systemd1",
		"/org/freedesktop/systemd1/unit/synnax_2eservice",
		"org.freedesktop.systemd1.Service",
		"ExecStart",
	)
	if err != nil {
		return "", nil, err
	}
	// Each command is a D-Bus structure that starts with the binary and the argv.
	var property struct {
		Data [][]jsontext.Value `json:"data"`
	}
	if err = json.Unmarshal([]byte(out), &property); err != nil {
		return "", nil, errors.Wrapf(err, "failed to read the command of %s", name)
	}
	if len(property.Data) == 0 || len(property.Data[0]) < 2 {
		return "", nil, errors.Newf("service %s has no command", name)
	}
	var (
		binary string
		argv   []string
	)
	if err = json.Unmarshal(property.Data[0][0], &binary); err != nil {
		return "", nil, errors.Wrapf(err, "failed to read the binary of %s", name)
	}
	if err = json.Unmarshal(property.Data[0][1], &argv); err != nil {
		return "", nil, errors.Wrapf(err, "failed to read the arguments of %s", name)
	}
	if len(argv) == 0 {
		return "", nil, errors.Newf("service %s has no command", name)
	}
	return binary, argv[1:], nil
}

// dataDir returns the data directory of a `synnax start` with the given arguments in
// the working directory workDir.
func dataDir(args []string, workDir string) (string, error) {
	cmd := &cobra.Command{
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	}
	cmdstart.AddDataFlag(cmd)
	config := cmd.Flags().String("config", "", "")
	if err := cmd.ParseFlags(args); err != nil {
		return "", errors.Wrapf(err, "failed to read the flags of service %s", name)
	}
	data := cmd.Flags().Lookup(cmdstart.FlagData)
	dir := data.Value.String()
	if !data.Changed && *config != "" {
		v := viper.New()
		v.SetConfigFile(*config)
		if err := v.ReadInConfig(); err != nil {
			return "", errors.Wrap(err, "failed to read service configuration")
		}
		if v.IsSet(cmdstart.FlagData) {
			dir = v.GetString(cmdstart.FlagData)
		}
	}
	if filepath.IsAbs(dir) {
		return dir, nil
	}
	if workDir == "" {
		workDir = "/"
	}
	return filepath.Join(workDir, dir), nil
}

// StatusInfo contains information about the service status and configuration.
type StatusInfo struct {
	// Installed is true when systemd has loaded the unit of the service.
	Installed bool
	// State is the active state and the sub-state, such as "active (running)".
	State string
	// ProcessID is the ID of the Core process, or 0 when it does not run.
	ProcessID int
	// ExitStatus is the exit status of the last Core process.
	ExitStatus int
	// ConfigPath is the path of the config file of the service.
	ConfigPath string
	// DataDir is the data directory in the config file.
	DataDir string
	// LogFile is the log file in the config file.
	LogFile string
	// CertsDir is the certificate directory in the config file.
	CertsDir string
	// Listen is the listen address in the config file.
	Listen string
	// Insecure is true when the config file turns off TLS.
	Insecure bool
	// ConfigError is the error from reading the config file, if any.
	ConfigError error
}

func status() (StatusInfo, error) {
	info := StatusInfo{ConfigPath: ConfigPath()}
	if _, err := os.Stat(info.ConfigPath); err == nil {
		v := viper.New()
		v.SetConfigFile(info.ConfigPath)
		if err = v.ReadInConfig(); err != nil {
			info.ConfigError = err
		} else {
			info.DataDir = v.GetString(cmdstart.FlagData)
			info.LogFile = v.GetString(cmdinst.FlagLogFilePath)
			info.CertsDir = v.GetString(cert.FlagCertsDir)
			info.Listen = v.GetString(listener.FlagListen)
			info.Insecure = v.GetBool(cmdstart.FlagInsecure)
		}
	}
	props, err := show(
		"LoadState",
		"ActiveState",
		"SubState",
		"MainPID",
		"ExecMainStatus",
	)
	if err != nil || props["LoadState"] != "loaded" {
		return info, err
	}
	info.Installed = true
	info.State = props["ActiveState"] + " (" + props["SubState"] + ")"
	if info.ProcessID, err = strconv.Atoi(props["MainPID"]); err != nil {
		return info, errors.Wrap(err, "failed to read the process ID")
	}
	if info.ExitStatus, err = strconv.Atoi(props["ExecMainStatus"]); err != nil {
		return info, errors.Wrap(err, "failed to read the exit status")
	}
	return info, nil
}
