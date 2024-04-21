// Copyright 2024 Synnax Labs, Inc.
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
	"bytes"
	"context"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/breaker"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
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
		return nil
	}
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(d.cfg.Instrumentation))
	d.shutdown = signal.NewShutdown(sCtx, cancel)
	bre := breaker.Breaker{
		BaseInterval: 1 * time.Second,
		Scale:        1.1,
		MaxRetries:   100,
	}
	var mf func(ctx context.Context) error
	mf = func(ctx context.Context) error {
		d.mu.Lock()
		ecd := &binary.JSONEncoderDecoder{}
		cfgFile, err := os.CreateTemp("", "synnax-driver-config*.json")
		if err != nil {
			return err
		}
		b, err := ecd.Encode(ctx, d.cfg.format())
		if err != nil {
			return err
		}
		if _, err := cfgFile.Write(b); err != nil {
			return err
		}
		if err := cfgFile.Close(); err != nil {
			return err
		}

		data, err := executable.ReadFile("assets/" + driverName)
		if err != nil {
			return err
		}
		tmpFile, err := os.CreateTemp("", driverName)
		if err != nil {
			return err
		}
		if _, err := tmpFile.Write(data); err != nil {
			return err
		}
		if err := tmpFile.Close(); err != nil {
			return err
		}
		defer func() {
			err = errors.CombineErrors(err, os.Remove(tmpFile.Name()))
			err = errors.CombineErrors(err, os.Remove(cfgFile.Name()))
		}()
		if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
			return err
		}
		d.cmd = exec.Command(tmpFile.Name(), cfgFile.Name())
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

		if err := d.cmd.Start(); err != nil {
			return err
		}

		internalSCtx, cancel := signal.Isolated(signal.WithInstrumentation(d.cfg.Instrumentation))
		defer cancel()

		internalSCtx.Go(func(ctx context.Context) error {
			pipeOutputToLogger(stdoutPipe, d.cfg.L)
			return nil
		}, signal.WithKey("stdoutPipe"))
		internalSCtx.Go(func(ctx context.Context) error {
			pipeOutputToLogger(stderrPipe, d.cfg.L)
			return nil
		}, signal.WithKey("stderrPipe"))
		internalSCtx.Go(func(ctx context.Context) error {
			err := d.cmd.Wait()
			return err
		}, signal.WithKey("wait"))
		err = internalSCtx.Wait()
		isSignal := false
		if err != nil {
			isSignal = strings.Contains(err.Error(), "signal")
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

func pipeOutputToLogger(reader io.ReadCloser, logger *alamos.Logger) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		// Find the first "]" and remove everything before it
		// This is to remove the timestamp from the log output
		b := scanner.Bytes()
		level := string(b[0])
		idx := bytes.IndexByte(b, ']')
		filtered := string(b[idx+1:])
		split := strings.Split(filtered, "]")
		dl := logger
		if len(split) >= 2 {
			first := split[0][2:]
			dl = logger.Named(first)
			filtered = split[1][1:]
		}
		switch level {
		case "D":
			dl.Debug(filtered)
		case "E", "F":
			dl.Error(filtered)
		case "W":
			dl.Warn(filtered)
		default:
			dl.Info(filtered)
		}
	}
	if err := scanner.Err(); err != nil {
		logger.Error("Error reading from std pipe", zap.Error(err))
	}
}

func (d *Driver) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.shutdown != nil && d.cmd != nil && d.cmd.Process != nil {
		d.cmd.Process.Signal(syscall.SIGINT)
		err := d.shutdown.Close()
		return err
	}
	return nil
}
