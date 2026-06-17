// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/driver/internal/log"
	"github.com/synnaxlabs/synnax/pkg/driver/internal/restart"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Config is the configuration for opening an embedded Driver.
type Config struct {
	// Insecure sets whether not to use TLS for communication. If insecure is set to
	// true, CACertPath, ClientCertFile, and ClientKeyFile are ignored.
	Insecure *bool `json:"insecure"`
	// Enabled is used to enable or disable the embedded Driver.
	Enabled *bool `json:"enabled"`
	// Debug sets whether to enable debug logging.
	Debug *bool `json:"debug"`
	// Instrumentation is used for logging, tracing, and metrics.
	alamos.Instrumentation
	// Credentials are the authentication credentials the Driver should use when
	// connecting to the Core.
	Credentials auth.Credentials
	// CACertPath sets the path to the CA certificate to use for authenticated/encrypted
	// communication. Not required if the CA is universally recognized or already
	// installed on the users' system.
	CACertPath string `json:"ca_cert_path"`
	// ClientCertFile sets the path to the client cert file to use for
	// authenticated/encrypted communication.
	ClientCertFile string `json:"client_cert_file"`
	// ClientKeyFile sets the secret key file used for authenticated/encrypted
	// communication between the Driver and the Core.
	ClientKeyFile string `json:"client_key_file"`
	// Address is the reachable address of the Core for the Driver to connect to.
	Address address.Address `json:"address"`
	// ParentDirname is the parent directory in which the Driver will create a 'driver'
	// directory to extract and execute the Driver binary and extract configuration
	// files into.
	ParentDirname string `json:"parent_dirname"`
	// FS is the filesystem containing the Driver binary. The binary is read from this
	// filesystem at the OS-specific name (driver / driver.exe), extracted to disk under
	// ParentDirname, and executed. When nil, the embedded Driver is unavailable and
	// Open returns a no-op Driver. Defaults to the embedded Driver binary when built
	// with -tags=driver, and is nil otherwise.
	FS fs.FS `json:"-"`
	// Integrations define which device integrations are enabled.
	Integrations []string `json:"integrations"`
	// StartTimeout sets the maximum acceptable time to wait for the Driver to bootup
	// successfully before timing out and returning a failed startup error.
	StartTimeout time.Duration `json:"start_timeout"`
	// StopTimeout is the time to wait for the Driver to exit gracefully after sending
	// STOP before escalating to a forceful kill.
	StopTimeout time.Duration `json:"stop_timeout"`
	// RestartBaseInterval is the initial backoff before restarting the Driver after an
	// unexpected exit; it grows exponentially with each consecutive failure. Core-side
	// supervision only, not forwarded to the Driver.
	RestartBaseInterval time.Duration `json:"-"`
	// RestartMaxRetries caps consecutive restarts after unexpected exits before the
	// supervisor gives up. The counter resets after a healthy run (see
	// RestartHealthyUptime).
	RestartMaxRetries int `json:"-"`
	// RestartHealthyUptime is the minimum run time after which an exit is treated as a
	// healthy run rather than a crash loop, resetting the restart counter.
	RestartHealthyUptime time.Duration `json:"-"`
	// TaskOpTimeout sets the duration before reporting stuck task operations.
	TaskOpTimeout time.Duration `json:"task_op_timeout"`
	// TaskPollInterval sets the interval between task timeout checks.
	TaskPollInterval time.Duration `json:"task_poll_interval"`
	// TaskShutdownTimeout sets the max time to wait for task workers during shutdown.
	TaskShutdownTimeout time.Duration `json:"task_shutdown_timeout"`
	// RackKey is the key of the rack that the Driver should assume the identity of.
	RackKey rack.Key `json:"rack_key"`
	// ClusterKey is the key of the current cluster.
	ClusterKey uuid.UUID `json:"cluster_key"`
	// TaskWorkerCount sets the number of worker threads for task operations.
	TaskWorkerCount uint8 `json:"task_worker_count"`
}

func (c Config) format() map[string]any {
	if *c.Insecure {
		c.CACertPath = ""
		c.ClientCertFile = ""
		c.ClientKeyFile = ""
	}
	return map[string]any{
		"connection": map[string]any{
			"host":             c.Address.Host(),
			"port":             c.Address.Port(),
			"credentials":      c.Credentials,
			"ca_cert_file":     c.CACertPath,
			"client_cert_file": c.ClientCertFile,
			"client_key_file":  c.ClientKeyFile,
		},
		"retry": map[string]any{
			"base_interval": 1,
			"max_retries":   40,
			"scale":         1.1,
		},
		"remote_info": map[string]any{
			"rack_key":    c.RackKey,
			"cluster_key": c.ClusterKey.String(),
		},
		"manager": map[string]any{
			"op_timeout":       c.TaskOpTimeout.Seconds(),
			"poll_interval":    c.TaskPollInterval.Seconds(),
			"shutdown_timeout": c.TaskShutdownTimeout.Seconds(),
			"worker_count":     c.TaskWorkerCount,
		},
		"integrations": c.Integrations,
		"debug":        *c.Debug,
	}
}

var (
	_               config.Config[Config] = Config{}
	AllIntegrations                       = []string{
		"arc",
		"ethercat",
		"http",
		"labjack",
		"modbus",
		"ni",
		"opc",
	}
	DefaultConfig = Config{
		Integrations:         []string{},
		Enabled:              new(true),
		Debug:                new(false),
		StartTimeout:         time.Second * 10,
		StopTimeout:          10 * time.Second,
		RestartBaseInterval:  2 * time.Second,
		RestartMaxRetries:    100,
		RestartHealthyUptime: time.Minute,
		TaskOpTimeout:        time.Second * 60,
		TaskPollInterval:     time.Second * 1,
		TaskShutdownTimeout:  time.Second * 30,
		TaskWorkerCount:      4,
	}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Enabled = override.Nil(c.Enabled, other.Enabled)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Address = override.String(c.Address, other.Address)
	c.RackKey = override.Numeric(c.RackKey, other.RackKey)
	c.ClusterKey = override.UUID(c.ClusterKey, other.ClusterKey)
	c.Integrations = override.Slice(c.Integrations, other.Integrations)
	c.Insecure = override.Nil(c.Insecure, other.Insecure)
	c.CACertPath = override.String(c.CACertPath, other.CACertPath)
	c.ClientCertFile = override.String(c.ClientCertFile, other.ClientCertFile)
	c.ClientKeyFile = override.String(c.ClientKeyFile, other.ClientKeyFile)
	c.Credentials = override.Zero(c.Credentials, other.Credentials)
	c.Debug = override.Nil(c.Debug, other.Debug)
	c.StartTimeout = override.Numeric(c.StartTimeout, other.StartTimeout)
	c.ParentDirname = override.String(c.ParentDirname, other.ParentDirname)
	c.FS = override.Nil(c.FS, other.FS)
	c.TaskOpTimeout = override.Numeric(c.TaskOpTimeout, other.TaskOpTimeout)
	c.TaskPollInterval = override.Numeric(c.TaskPollInterval, other.TaskPollInterval)
	c.TaskShutdownTimeout = override.Numeric(
		c.TaskShutdownTimeout, other.TaskShutdownTimeout,
	)
	c.TaskWorkerCount = override.Numeric(c.TaskWorkerCount, other.TaskWorkerCount)
	c.StopTimeout = override.Numeric(c.StopTimeout, other.StopTimeout)
	c.RestartBaseInterval = override.Numeric(
		c.RestartBaseInterval, other.RestartBaseInterval,
	)
	c.RestartMaxRetries = override.Numeric(c.RestartMaxRetries, other.RestartMaxRetries)
	c.RestartHealthyUptime = override.Numeric(
		c.RestartHealthyUptime, other.RestartHealthyUptime,
	)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("driver.embedded")
	validate.NotNil(v, "enabled", c.Enabled)
	validate.NotNil(v, "insecure", c.Insecure)
	if v.Error() != nil {
		return v.Error()
	}
	if !*c.Enabled {
		return nil
	}
	validate.NotEmptyString(v, "address", c.Address)
	validate.NotNil(v, "debug", c.Debug)
	validate.NotEmptyString(v, "parent_dirname", c.ParentDirname)
	validate.InBounds(v, "task_worker_count", c.TaskWorkerCount, 1, 64)
	return v.Error()
}

// restartScale is the exponential growth factor applied to the restart backoff after
// each consecutive unexpected Driver exit.
const restartScale = 1.1

// Driver manages the lifecycle of an embedded C++ Driver subprocess. The Driver binary
// is read from the configured filesystem (Config.FS), extracted to disk, and executed
// as a child process that communicates with the Core.
//
// On startup, Open launches the subprocess and two goroutines that pipe its stdout and
// stderr through PipeToLogger. A third goroutine waits for the process to exit. All
// three run under an isolated signal context. Open blocks until the subprocess prints
// "started successfully" or the StartTimeout expires. If startup fails, Open cleans up
// the process and returns (nil, err).
//
// On shutdown, Close cancels the supervisor context. The subprocess is launched via
// exec.CommandContext, so the cancellation asks it to stop gracefully (a STOP write via
// cmd.Cancel) and escalates to a kill after StopTimeout (cmd.WaitDelay); Close then
// waits for the supervisor goroutines to exit. Close is idempotent — concurrent and
// repeated calls return the result of the first invocation.
//
// When the binary is unavailable — either because the server was built without the
// "driver" build tag or because Enabled was set to false — Open returns a Driver whose
// methods are no-ops.
type Driver struct {
	// cfg holds the validated configuration for the Driver.
	cfg Config
	// started is closed once the subprocess prints "started successfully". Open blocks
	// on this channel to know when startup is complete.
	started chan struct{}
	// shutdown cancels the supervisor context and waits for its goroutines to exit.
	// Canceling the context stops the running subprocess (see setupCmd), so Close needs
	// nothing more than this. Nil when the Driver is disabled.
	shutdown io.Closer
	// closeOnce ensures close() executes exactly once, making Close idempotent.
	closeOnce sync.Once
	// closeErr stores the result of the single close() invocation for subsequent Close
	// calls to return.
	closeErr error
}

// Open creates and starts an embedded Driver. When the Core is built with -tags=driver,
// the Driver binary is extracted from the embedded filesystem and executed as a
// subprocess. Otherwise, Open returns a [Driver] whose methods are no-ops. Open also
// returns a no-op Driver when cfg.Enabled is false.
func Open(ctx context.Context, cfgs ...Config) (*Driver, error) {
	cfg, err := config.New(Config{FS: defaultFS}.Override(DefaultConfig), cfgs...)
	if err != nil {
		return nil, err
	}
	d := &Driver{cfg: cfg, started: make(chan struct{})}
	if !d.enabled() {
		if !*cfg.Enabled {
			cfg.L.Info("embedded Driver disabled")
		} else {
			cfg.L.Info("Core built without embedded Driver")
		}
		return d, nil
	}
	ctx, cancel := context.WithTimeout(ctx, cfg.StartTimeout)
	defer cancel()
	if err := d.start(ctx); err != nil {
		return nil, err
	}
	return d, nil
}

// enabled reports whether Open should launch the Driver subprocess. The subprocess is
// only launched when the configuration enables it AND a Driver binary is available via
// the configured filesystem.
func (d *Driver) enabled() bool { return *d.cfg.Enabled && d.cfg.FS != nil }

func (d *Driver) start(ctx context.Context) error {
	d.cfg.L.Info("starting embedded Driver")
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(d.cfg.Instrumentation))
	// HardShutdown cancels the context before waiting. Cancellation both stops the
	// running subprocess (exec.CommandContext) and wakes the restart backoff
	// (Policy.Decide); a graceful wait-then-cancel would instead deadlock against a
	// supervisor parked in that backoff.
	d.shutdown = signal.NewHardShutdown(sCtx, cancel)
	policy, err := restart.New(sCtx, restart.Config{
		BaseInterval:  d.cfg.RestartBaseInterval,
		Scale:         restartScale,
		MaxRetries:    d.cfg.RestartMaxRetries,
		HealthyUptime: d.cfg.RestartHealthyUptime,
	})
	if err != nil {
		return err
	}
	sCtx.Go(func(ctx context.Context) error {
		// startedOnce is shared across restarts so d.started is closed exactly once, by
		// whichever run first reports a successful start. A per-run Once would let a
		// restarted run close the already-closed channel and panic.
		startedOnce := &sync.Once{}
		for {
			action, err := d.runOnce(ctx, policy, startedOnce)
			// A canceled context means Close initiated shutdown. Stop quietly
			// regardless of the action, so a restart decision that raced the cancel
			// never relaunches.
			if ctx.Err() != nil {
				return nil
			}
			switch action {
			case restart.Restart:
				continue
			case restart.GiveUp:
				d.cfg.L.Error(
					"embedded Driver exceeded restart limit; giving up",
					zap.Error(err),
				)
				return err
			default:
				// restart.Stop: a launch failure (an expected exit was caught above).
				return err
			}
		}
	})
	if _, err = signal.RecvUnderContext(ctx, d.started); err != nil {
		closeErr := d.Close()
		if errors.Is(err, context.DeadlineExceeded) {
			return errors.Combine(
				errors.New(
					"timed out waiting for embedded Driver to start. This occurs either because the Driver could not reach the Core or a task took an unusual amount of time to start. Check logs above categorized 'driver' for more information.",
				),
				closeErr,
			)
		}
		return errors.Combine(
			errors.Wrap(err, "failed to start embedded Driver"),
			closeErr,
		)
	}
	return nil
}

// runOnce launches the Driver subprocess, pipes its output, and blocks until it exits.
// Its deferred temp-file and context cleanup run before it returns, so a supervisor
// that calls runOnce in a loop neither accumulates cleanups nor grows its stack per
// restart. It returns the restart Action for the exit and the exit (or launch) error.
func (d *Driver) runOnce(
	ctx context.Context,
	policy *restart.Policy,
	startedOnce *sync.Once,
) (restart.Action, error) {
	cmd, cfgFile, extractedBinary, err := d.setupCmd(ctx)
	if cfgFile != "" {
		defer func() {
			if rmErr := os.Remove(cfgFile); rmErr != nil {
				d.cfg.L.Error("failed to remove config file", zap.Error(rmErr))
			}
		}()
	}
	if extractedBinary != "" {
		defer func() {
			if rmErr := os.Remove(extractedBinary); rmErr != nil {
				d.cfg.L.Error("failed to remove extracted binary", zap.Error(rmErr))
			}
		}()
	}
	if err != nil {
		return restart.Stop, err
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return restart.Stop, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return restart.Stop, err
	}
	if err = cmd.Start(); err != nil {
		return restart.Stop, err
	}
	startedAt := time.Now()

	internalSCtx, cancel := signal.Isolated(
		signal.WithInstrumentation(d.cfg.Instrumentation),
	)
	defer cancel()

	internalSCtx.Go(func(context.Context) error {
		log.PipeToLogger(stdoutPipe, d.cfg.L, d.started, startedOnce)
		return nil
	},
		signal.WithKey("stdout_pipe"),
		signal.RecoverWithErrOnPanic(),
		signal.WithRetryOnPanic(),
	)
	internalSCtx.Go(func(context.Context) error {
		log.PipeToLogger(stderrPipe, d.cfg.L, d.started, startedOnce)
		return nil
	},
		signal.WithKey("stderr_pipe"),
		signal.RecoverWithErrOnPanic(),
		signal.WithRetryOnPanic(),
	)
	internalSCtx.Go(func(context.Context) error {
		return cmd.Wait()
	},
		signal.WithKey("wait"),
		signal.RecoverWithErrOnPanic())
	err = internalSCtx.Wait()
	// A canceled context means Close stopped the process: the exit is expected, not a
	// crash, so it must not count toward the restart policy.
	expected := ctx.Err() != nil
	if !expected {
		d.cfg.L.Warn("embedded Driver process exited unexpectedly", zap.Error(err))
	}
	return policy.Decide(expected, time.Since(startedAt)), err
}

// Close stops the Driver and waits for it to exit. It cancels the supervisor context,
// which asks the subprocess to stop gracefully and escalates to a kill after
// StopTimeout (see setupCmd), then waits for the supervisor goroutines to exit. Close
// is idempotent — subsequent calls return the result of the first.
func (d *Driver) Close() error {
	d.closeOnce.Do(func() { d.closeErr = d.close() })
	return d.closeErr
}

func (d *Driver) close() error {
	if d.shutdown == nil {
		return nil
	}
	d.cfg.L.Info("stopping embedded Driver")
	return d.shutdown.Close()
}

// setupCmd writes the config file, extracts the binary, and constructs the subprocess
// command bound to ctx. Canceling ctx asks the Driver to stop gracefully (a STOP write
// via cmd.Cancel) and escalates to a kill after StopTimeout (cmd.WaitDelay). It returns
// the command and the paths of any temp files created so the caller can defer their
// cleanup.
func (d *Driver) setupCmd(
	ctx context.Context,
) (_ *exec.Cmd, cfgFile, extractedBinary string, _ error) {
	b, err := json.Marshal(d.cfg.format())
	if err != nil {
		return nil, "", "", err
	}
	workDir := filepath.Join(d.cfg.ParentDirname, "driver")
	if err = os.MkdirAll(workDir, xfs.UserRWX); err != nil {
		return nil, "", "", err
	}
	cfgFile = filepath.Join(workDir, "config.json")
	if err = os.WriteFile(cfgFile, b, xfs.UserRW); err != nil {
		return nil, "", "", err
	}
	data, err := fs.ReadFile(d.cfg.FS, driverName)
	if err != nil {
		return nil, cfgFile, "", err
	}
	extractedBinary = filepath.Join(workDir, driverName)
	if err = os.WriteFile(extractedBinary, data, xfs.UserRWX); err != nil {
		return nil, cfgFile, "", err
	}
	flags := []string{"start", "--standalone", "--disable-sig-stop", "--no-color"}
	if *d.cfg.Debug {
		flags = append(flags, "--debug")
	}
	flags = append(flags, "--config", cfgFile)
	cmd := exec.CommandContext(ctx, extractedBinary, flags...)
	configureSysProcAttr(cmd)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, cfgFile, extractedBinary, err
	}
	// On ctx cancellation, ask the Driver to stop gracefully via STOP instead of the
	// default SIGKILL; WaitDelay then escalates to a kill if it does not exit in time
	// and guarantees the stdin pipe is closed so cmd.Wait cannot block indefinitely.
	cmd.Cancel = func() error { _, err := io.WriteString(stdin, "STOP\n"); return err }
	cmd.WaitDelay = d.cfg.StopTimeout
	return cmd, cfgFile, extractedBinary, nil
}
