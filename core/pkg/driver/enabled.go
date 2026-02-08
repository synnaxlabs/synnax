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
	"strings"
	"sync"
	"time"

	"github.com/synnaxlabs/synnax/pkg/driver/internal/log"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/breaker"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/signal"
)

const (
	startCmdName        = "start"
	startStandaloneFlag = "--standalone"
	blockSigStopFlag    = "--disable-sig-stop"
	noColorFlag         = "--no-color"
	configFlag          = "--config"
	debugFlag           = "--debug"
)

var (
	errStartTimeout = errors.New(
		`timed out waiting for embedded Driver to start. This occurs either because
the Driver could not reach the Core or a task took an unusual amount of time to
start. Check logs above categorized 'driver' for more information.
`,
	)
	errForceKillFailed = errors.New(
		`embedded Driver shutdown timed out after being force killed. This may indicate
a hardware deadlock or other issue preventing the Driver from exiting gracefully.
`,
	)
)

const (
	// embeddedDriverPath is the path at which the driver lives inside our
	// embedded fs. Unix style paths are used in embedded directories regardless
	// of the OS.
	embeddedDriverPath = "assets/" + driverName
	configFileName     = "config.json"
	extractedDriverDir = "driver"
)

var configCodec = &binary.JSONCodec{}

// Driver manages the lifecycle of an embedded C++ driver subprocess. The driver
// binary is either extracted from an embedded filesystem or loaded from a configured
// path, then executed as a child process that communicates with the Synnax cluster.
//
// On startup, Open launches the subprocess and two goroutines that pipe its stdout and
// stderr through PipeToLogger. A third goroutine waits for the process to exit. All
// three run under an isolated signal context. Open blocks until the subprocess prints
// "started successfully" or the StartTimeout expires. If startup fails, Open cleans
// up the process and returns (nil, err).
//
// On shutdown, Close writes "STOP\n" to the subprocess's stdin, giving it a chance to
// exit gracefully. If the process doesn't exit within StopTimeout, Close escalates to
// Process.Kill. A secondary StopTimeout after the kill guarantees Close never blocks
// indefinitely. Close is idempotent — concurrent and repeated calls return the result
// of the first invocation.
type Driver struct {
	// cfg holds the validated configuration for the driver.
	cfg Config
	// cmd is the running driver subprocess. Nil before Start or after a failed launch.
	cmd *exec.Cmd
	// stdInPipe is the write end of the subprocess's stdin, used to send the STOP
	// command during shutdown.
	stdInPipe io.WriteCloser
	// started is closed once the subprocess prints "started successfully". Open
	// blocks on this channel to know when startup is complete.
	started chan struct{}
	// shutdown wraps the signal context's cancel and wait, allowing Close to tear
	// down the goroutines that manage the subprocess's I/O and lifetime.
	shutdown io.Closer
	// mu guards the cmd and stdInPipe fields during subprocess setup in start().
	mu sync.Mutex
	// closeOnce ensures close() executes exactly once, making Close idempotent.
	closeOnce sync.Once
	// closeErr stores the result of the single close() invocation for subsequent
	// Close calls to return.
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
	d.shutdown = signal.NewGracefulShutdown(sCtx, cancel)
	bre, err := breaker.NewBreaker(sCtx, breaker.Config{
		BaseInterval: 1 * time.Second,
		Scale:        1.1,
		MaxRetries:   100,
	})
	if err != nil {
		return err
	}
	var mf func(ctx context.Context) error
	mf = func(ctx context.Context) error {
		cfgFile, extractedBinary, err := d.setupCmd(ctx)
		if cfgFile != "" {
			defer func() { _ = os.Remove(cfgFile) }()
		}
		if extractedBinary != "" {
			defer func() { _ = os.Remove(extractedBinary) }()
		}
		if err != nil {
			return err
		}
		stdoutPipe, err := d.cmd.StdoutPipe()
		if err != nil {
			return err
		}
		stderrPipe, err := d.cmd.StderrPipe()
		if err != nil {
			return err
		}
		d.stdInPipe, err = d.cmd.StdinPipe()
		if err != nil {
			return err
		}

		if err := d.cmd.Start(); err != nil {
			return err
		}

		internalSCtx, cancel := signal.Isolated(signal.WithInstrumentation(d.cfg.Instrumentation))
		defer cancel()

		startedOnce := &sync.Once{}
		internalSCtx.Go(func(ctx context.Context) error {
			log.PipeToLogger(stdoutPipe, d.cfg.L, d.started, startedOnce)
			return nil
		},
			signal.WithKey("stdout_pipe"),
			signal.RecoverWithErrOnPanic(),
			signal.WithRetryOnPanic(),
		)
		internalSCtx.Go(func(ctx context.Context) error {
			log.PipeToLogger(stderrPipe, d.cfg.L, d.started, startedOnce)
			return nil
		},
			signal.WithKey("stderr_pipe"),
			signal.RecoverWithErrOnPanic(),
			signal.WithRetryOnPanic(),
		)
		internalSCtx.Go(func(ctx context.Context) error {
			err := d.cmd.Wait()
			return err
		},
			signal.WithKey("wait"),
			signal.RecoverWithErrOnPanic())
		err = internalSCtx.Wait()
		isSignal := false
		if err != nil {
			isSignal = strings.Contains(err.Error(), "signal") || strings.Contains(err.Error(), "exit status")
			if bre.Wait() && !isSignal {
				return mf(ctx)
			}
		}
		if isSignal {
			return nil
		}
		return err
	}
	sCtx.Go(mf)
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

const stopKeyword = "STOP\n"

// Close stops the driver process and waits up to StopTimeout for
// it to exit. If the process doesn't exit within the grace period, it escalates
// to Process.Kill. This prevents Close from blocking indefinitely when the
// driver is hung (hardware deadlock, stuck initialization, etc.). Close is
// idempotent — subsequent calls return the result of the first.
func (d *Driver) Close() error {
	d.closeOnce.Do(func() { d.closeErr = d.close() })
	return d.closeErr
}

func (d *Driver) close() error {
	if d.shutdown == nil {
		return nil
	}
	if d.cmd != nil && d.cmd.Process != nil {
		// Best-effort: ask the process to exit gracefully. If the process
		// already exited (e.g. crash or timeout cleanup race), the pipe is
		// closed and the write fails harmlessly.
		_, _ = d.stdInPipe.Write([]byte(stopKeyword))
	}
	done := make(chan error, 1)
	go func() { done <- d.shutdown.Close() }()
	select {
	case err := <-done:
		return err
	case <-time.After(d.cfg.StopTimeout):
		if d.cmd != nil && d.cmd.Process != nil {
			_ = d.cmd.Process.Kill()
		}
		select {
		case err := <-done:
			return err
		case <-time.After(d.cfg.StopTimeout):
			return errForceKillFailed
		}
	}
}

// setupCmd prepares the driver subprocess command under d.mu, writing the config
// file, extracting the binary (if needed), and constructing the exec.Cmd. It returns
// the paths of any temp files created so the caller can defer their cleanup.
func (d *Driver) setupCmd(ctx context.Context) (cfgFile, extractedBinary string, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	b, err := configCodec.Encode(ctx, d.cfg.format())
	if err != nil {
		return "", "", err
	}
	workDir := filepath.Join(d.cfg.ParentDirname, extractedDriverDir)
	if err = os.MkdirAll(workDir, xfs.UserRWX); err != nil {
		return "", "", err
	}
	cfgFile = filepath.Join(workDir, configFileName)
	if err = os.WriteFile(cfgFile, b, xfs.UserRW); err != nil {
		return "", "", err
	}
	var driverPath string
	if d.cfg.BinaryPath != "" {
		driverPath = d.cfg.BinaryPath
	} else {
		var data []byte
		data, err = executable.ReadFile(embeddedDriverPath)
		if err != nil {
			return cfgFile, "", err
		}
		extractedBinary = filepath.Join(workDir, driverName)
		if err = os.WriteFile(extractedBinary, data, xfs.UserRWX); err != nil {
			return cfgFile, "", err
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
	d.cmd = exec.Command(driverPath, flags...)
	configureSysProcAttr(d.cmd)
	return cfgFile, extractedBinary, nil
}
