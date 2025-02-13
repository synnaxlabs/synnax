// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build driver

package embedded

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/breaker"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xos "github.com/synnaxlabs/x/os"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	startCmdName        = "start"
	startStandaloneFlag = "--standalone"
)

func OpenDriver(ctx context.Context, cfgs ...Config) (*Driver, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	d := &Driver{cfg: cfg}
	return d, d.start()
}

func (d *Driver) start() error {
	if !*d.cfg.Enabled {
		d.cfg.L.Info("embedded driver disabled")
		return nil
	}
	d.cfg.L.Info("starting embedded driver")
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(d.cfg.Instrumentation))
	d.shutdown = signal.NewShutdown(sCtx, cancel)
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
		codec := &binary.JSONCodec{}
		b, err := codec.Encode(ctx, d.cfg.format())
		if err != nil {
			return err
		}
		cfgFileName, err := xos.WriteTemp("", "synnax-driver-config*.json", b)
		if err != nil {
			return err
		}
		data, err := executable.ReadFile("assets/" + driverName)
		if err != nil {
			return err
		}
		driverFileName, err := xos.WriteTemp("", driverName, data)
		if err != nil {
			return err
		}
		defer func() {
			err = errors.Combine(err, os.Remove(cfgFileName))
			err = errors.Combine(err, os.Remove(driverFileName))
		}()
		if err := os.Chmod(driverFileName, 0755); err != nil {
			return err
		}
		d.cmd = exec.Command(driverFileName, "start", "-s", "--block-sig-stop", "--config", cfgFileName)
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
			pipeOutputToLogger(stdoutPipe, d.cfg.L)
			return nil
		},
			signal.WithKey("stdoutPipe"),
			signal.RecoverWithErrOnPanic(),
			signal.WithRetryOnPanic(),
		)
		internalSCtx.Go(func(ctx context.Context) error {
			pipeOutputToLogger(stderrPipe, d.cfg.L)
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
	return nil
}

const stopKeyword = "STOP\n"

func (d *Driver) Stop() error {
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

func pipeOutputToLogger(reader io.ReadCloser, logger *alamos.Logger) {
	existingCfg := logger.Config
	var caller string
	existingCfg.ZapConfig.EncoderConfig.EncodeCaller = func(_ zapcore.EntryCaller, encoder zapcore.PrimitiveArrayEncoder) {
		encoder.AppendString(caller)
	}
	logger, _ = logger.WithConfig(existingCfg)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		// Find the first "]" and remove everything before it
		// This is to remove the timestamp from the log output
		b := scanner.Bytes()
		dl := logger
		if len(b) == 0 {
			dl.Warn("received empty log line from driver")
			continue
		}
		level := string(b[0])
		original := string(b)
		split := strings.Split(original, "]")
		message := original
		if len(split) >= 3 {
			callerSplit := strings.Split(split[0], " ")
			caller = callerSplit[len(callerSplit)-1]
			first := split[1]
			if len(first) >= 2 {
				first = first[2:]
			}
			dl = logger.Named(first)
			message = split[2]
			if len(message) > 1 {
				message = message[1:]
			}
		}
		switch level {
		case "D":
			dl.Debug(message)
		case "E", "F":
			dl.Error(message)
		case "W":
			dl.Warn(message)
		default:
			dl.Info(message)
		}
	}
	if err := scanner.Err(); err != nil {
		logger.Error("Error reading from std pipe", zap.Error(err))
	}
}
