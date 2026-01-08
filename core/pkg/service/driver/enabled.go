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
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/breaker"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
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
	startedMessage      = "started successfully"
)

var errStartTimeout = errors.New(
	`timed out waiting for embedded Driver to start. This occurs either because
the Driver could not reach the Core or a task took an unusual amount of time to
start. Check logs above categorized 'driver' for more information.
`,
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

func OpenDriver(ctx context.Context, cfgs ...Config) (*Driver, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	d := &Driver{cfg: cfg, started: make(chan struct{})}
	ctx, cancel := context.WithTimeout(ctx, cfg.StartTimeout)
	defer cancel()
	return d, d.start(ctx)
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
		d.mu.Lock()
		b, err := configCodec.Encode(ctx, d.cfg.format())
		if err != nil {
			return err
		}
		workDir := filepath.Join(d.cfg.ParentDirname, extractedDriverDir)
		if err = os.MkdirAll(workDir, xfs.OwnerReadWriteExecute); err != nil {
			return err
		}
		cfgFileName := filepath.Join(workDir, configFileName)
		if err = os.WriteFile(cfgFileName, b, xfs.OwnerReadWrite); err != nil {
			return err
		}
		data, err := executable.ReadFile(embeddedDriverPath)
		if err != nil {
			return err
		}
		driverFileName := filepath.Join(workDir, driverName)
		if err = os.WriteFile(driverFileName, data, xfs.OwnerReadWriteExecute); err != nil {
			return err
		}
		defer func() {
			err = errors.Combine(err, os.Remove(cfgFileName))
			err = errors.Combine(err, os.Remove(driverFileName))
		}()
		flags := []string{
			startCmdName,
			startStandaloneFlag,
			blockSigStopFlag,
			noColorFlag,
		}
		if *d.cfg.Debug {
			flags = append(flags, debugFlag)
		}
		flags = append(flags, configFlag, cfgFileName)
		d.cmd = exec.Command(driverFileName, flags...)
		configureSysProcAttr(d.cmd)
		d.mu.Unlock()
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

		internalSCtx.Go(func(ctx context.Context) error {
			pipeOutputToLogger(stdoutPipe, d.cfg.L, d.started)
			return nil
		},
			signal.WithKey("stdoutPipe"),
			signal.RecoverWithErrOnPanic(),
			signal.WithRetryOnPanic(),
		)
		internalSCtx.Go(func(ctx context.Context) error {
			pipeOutputToLogger(stderrPipe, d.cfg.L, d.started)
			return nil
		},
			signal.WithKey("stderrPipe"),
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
		if errors.Is(err, context.DeadlineExceeded) {
			return errStartTimeout
		}
		return errors.Wrap(err, "failed to start Embedded Driver")
	}
	return nil
}

const stopKeyword = "STOP\n"

func (d *Driver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.shutdown != nil && d.cmd != nil && d.cmd.Process != nil {
		if _, err := d.stdInPipe.Write([]byte(stopKeyword)); err != nil {
			return err
		}
		err := d.shutdown.Close()
		return err
	}
	return nil
}

func pipeOutputToLogger(
	reader io.ReadCloser,
	logger *alamos.Logger,
	started chan<- struct{},
) {
	var caller string
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		// Find the first "]" and remove everything before it
		// This is to remove the timestamp from the log output
		b := scanner.Bytes()
		namedLogger := logger
		if len(b) == 0 {
			namedLogger.Warn("received empty log line from driver")
			continue
		}
		level := string(b[0])
		original := string(b)
		split := strings.Split(original, "]")
		message := original
		if len(split) >= 3 {
			callerSplit := strings.Split(split[0], " ")
			caller = callerSplit[len(callerSplit)-1]
			first := strings.TrimSpace(split[1])
			namedLogger = logger.Named(first)
			message = split[2]
			if len(message) > 1 {
				message = message[1:]
			}
		} else if len(split) == 2 {
			callerSplit := strings.Split(split[0], " ")
			caller = callerSplit[len(callerSplit)-1]
			message = split[1]
		}
		message = strings.TrimSpace(message)
		if started != nil && message == startedMessage {
			close(started)
		}
		callerField := zap.String("caller", caller)
		switch level {
		case "D":
			namedLogger.Debug(message, callerField)
		case "E", "F":
			namedLogger.Error(message, callerField)
		case "W":
			namedLogger.Warn(message, callerField)
		default:
			namedLogger.Info(message, callerField)
		}
	}
	if err := scanner.Err(); err != nil {
		logger.Error("Error reading from std pipe", zap.Error(err))
	}
}
