// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build driver

package driver

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/synnaxlabs/synnax/pkg/driver/internal/log"
	"github.com/synnaxlabs/synnax/pkg/driver/internal/restart"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/errors"
	fs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
)

const (
	startCmdName        = "start"
	startStandaloneFlag = "--standalone"
	blockSigStopFlag    = "--disable-sig-stop"
	noColorFlag         = "--no-color"
	configFlag          = "--config"
	debugFlag           = "--debug"
)

// restartScale is the exponential growth factor applied to the restart backoff after
// each consecutive unexpected driver exit.
const restartScale = 1.1

var errStartTimeout = errors.New(
	`timed out waiting for embedded Driver to start. This occurs either because
the Driver could not reach the Core or a task took an unusual amount of time to
start. Check logs above categorized 'driver' for more information.
`,
)

const (
	// embeddedDriverPath is the path at which the driver lives inside our embedded fs.
	// Unix style paths are used in embedded directories regardless
	// of the OS.
	embeddedDriverPath = "assets/" + driverName
	configFileName     = "config.json"
	extractedDriverDir = "driver"
)

var configCodec = json.Codec

// Driver manages the lifecycle of an embedded C++ driver subprocess. The driver binary
// is either extracted from an embedded filesystem or loaded from a configured path,
// then executed as a child process that communicates with the Synnax cluster.
//
// On startup, Open launches the subprocess and two goroutines that pipe its stdout and
// stderr through PipeToLogger. A third goroutine waits for the process to exit. All
// three run under an isolated signal context. Open blocks until the subprocess prints
// "started successfully" or the StartTimeout expires. If startup fails, Open cleans up
// the process and returns (nil, err).
//
// On shutdown, Close cancels the supervisor context. The subprocess is launched via
// exec.CommandContext, so the cancellation asks it to stop gracefully (a STOP write via
// cmd.Cancel) and escalates to a kill after StopTimeout (cmd.WaitDelay); Close then waits
// for the supervisor goroutines to exit. Close is idempotent — concurrent and repeated
// calls return the result of the first invocation.
type Driver struct {
	// cfg holds the validated configuration for the driver.
	cfg Config
	// started is closed once the subprocess prints "started successfully". Open blocks
	// on this channel to know when startup is complete.
	started chan struct{}
	// shutdown cancels the supervisor context and waits for its goroutines to exit.
	// Canceling the context stops the running subprocess (see setupCmd), so Close needs
	// nothing more than this. Nil when the driver is disabled.
	shutdown io.Closer
	// closeOnce ensures close() executes exactly once, making Close idempotent.
	closeOnce sync.Once
	// closeErr stores the result of the single close() invocation for subsequent Close
	// calls to return.
	closeErr error
}

func Open(ctx context.Context, cfgs ...Config) (*Driver, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	d := &Driver{cfg: cfg, started: make(chan struct{})}
	ctx, cancel := context.WithTimeout(ctx, cfg.StartTimeout)
	defer cancel()
	if err := d.start(ctx); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *Driver) start(ctx context.Context) error {
	if !*d.cfg.Enabled {
		d.cfg.L.Info("embedded driver disabled")
		return nil
	}
	d.cfg.L.Info("starting embedded driver")
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(d.cfg.Instrumentation))
	// HardShutdown cancels the context before waiting. Cancellation both stops the
	// running subprocess (exec.CommandContext) and wakes the restart backoff
	// (breaker.Wait); a graceful wait-then-cancel would instead deadlock against a
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
			// A canceled context means Close initiated shutdown. Stop quietly regardless
			// of the action, so a restart decision that raced the cancel never relaunches.
			if ctx.Err() != nil {
				return nil
			}
			switch action {
			case restart.Restart:
				continue
			case restart.GiveUp:
				d.cfg.L.Error(
					"embedded driver exceeded restart limit; giving up",
					zap.Error(err),
				)
				return err
			default: // restart.Stop: a launch failure (an expected exit is caught above).
				return err
			}
		}
	})
	if _, err = signal.RecvUnderContext(ctx, d.started); err != nil {
		closeErr := d.Close()
		if errors.Is(err, context.DeadlineExceeded) {
			return errors.Combine(errStartTimeout, closeErr)
		}
		return errors.Combine(
			errors.Wrap(err, "failed to start Embedded Driver"),
			closeErr,
		)
	}
	return nil
}

// runOnce launches the driver subprocess, pipes its output, and blocks until it exits.
// Its deferred temp-file and context cleanup run before it returns, so a supervisor that
// calls runOnce in a loop neither accumulates cleanups nor grows its stack per restart.
// It returns the restart Action for the exit and the exit (or launch) error.
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

	internalSCtx, cancel := signal.Isolated(signal.WithInstrumentation(d.cfg.Instrumentation))
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
		d.cfg.L.Warn("embedded driver process exited unexpectedly", zap.Error(err))
	}
	return policy.Decide(expected, time.Since(startedAt)), err
}

const stopKeyword = "STOP\n"

// Close stops the driver and waits for it to exit. It cancels the supervisor context,
// which asks the subprocess to stop gracefully and escalates to a kill after StopTimeout
// (see setupCmd), then waits for the supervisor goroutines to exit. Close is idempotent —
// subsequent calls return the result of the first.
func (d *Driver) Close() error {
	d.closeOnce.Do(func() { d.closeErr = d.close() })
	return d.closeErr
}

func (d *Driver) close() error {
	if d.shutdown == nil {
		return nil
	}
	d.cfg.L.Info("stopping embedded driver")
	return d.shutdown.Close()
}

// setupCmd writes the config file, extracts the binary (if needed), and constructs the
// subprocess command bound to ctx. Canceling ctx asks the driver to stop gracefully (a
// STOP write via cmd.Cancel) and escalates to a kill after StopTimeout (cmd.WaitDelay).
// It returns the command and the paths of any temp files created so the caller can defer
// their cleanup.
func (d *Driver) setupCmd(
	ctx context.Context,
) (cmd *exec.Cmd, cfgFile, extractedBinary string, _ error) {
	b, err := configCodec.Encode(ctx, d.cfg.format())
	if err != nil {
		return nil, "", "", err
	}
	workDir := filepath.Join(d.cfg.ParentDirname, extractedDriverDir)
	if err = os.MkdirAll(workDir, fs.UserRWX); err != nil {
		return nil, "", "", err
	}
	cfgFile = filepath.Join(workDir, configFileName)
	if err = os.WriteFile(cfgFile, b, fs.UserRW); err != nil {
		return nil, "", "", err
	}
	var driverPath string
	if d.cfg.BinaryPath != "" {
		driverPath = d.cfg.BinaryPath
	} else {
		var data []byte
		data, err = executable.ReadFile(embeddedDriverPath)
		if err != nil {
			return nil, cfgFile, "", err
		}
		extractedBinary = filepath.Join(workDir, driverName)
		if err = os.WriteFile(extractedBinary, data, fs.UserRWX); err != nil {
			return nil, cfgFile, "", err
		}
		driverPath = extractedBinary
	}
	flags := []string{
		startCmdName,
		startStandaloneFlag,
		blockSigStopFlag,
		noColorFlag,
	}
	if *d.cfg.Debug {
		flags = append(flags, debugFlag)
	}
	flags = append(flags, configFlag, cfgFile)
	cmd = exec.CommandContext(ctx, driverPath, flags...)
	configureSysProcAttr(cmd)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, cfgFile, extractedBinary, err
	}
	// On ctx cancellation, ask the driver to stop gracefully via STOP instead of the
	// default SIGKILL; WaitDelay then escalates to a kill if it does not exit in time and
	// guarantees the stdin pipe is closed so cmd.Wait cannot block indefinitely.
	cmd.Cancel = func() error { _, err := io.WriteString(stdin, stopKeyword); return err }
	cmd.WaitDelay = d.cfg.StopTimeout
	return cmd, cfgFile, extractedBinary, nil
}
