// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build driver
// +build driver

package embedded

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"
)

//go:embed assets/driver
var executable embed.FS

func OpenDriver(ctx context.Context, cfgs ...Config) (*Driver, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)

	ecd := &binary.JSONEncoderDecoder{}
	cfgFile, err := os.CreateTemp("", "synnax-driver-config*.json")
	if err != nil {
		return nil, err
	}
	b, err := ecd.Encode(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if _, err := cfgFile.Write(b); err != nil {
		return nil, err
	}
	if err := cfgFile.Close(); err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	data, err := executable.ReadFile("assets/driver")
	if err != nil {
		return nil, err
	}
	tmpFile, err := os.CreateTemp("", "driver")
	if err != nil {
		return nil, err
	}
	if _, err := tmpFile.Write(data); err != nil {
		return nil, err
	}
	if err := tmpFile.Close(); err != nil {
		return nil, err
	}
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return nil, err
	}
	d := &Driver{cmd: exec.Command(tmpFile.Name(), "--config", cfgFile.Name())}
	d.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdoutPipe, err := d.cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderrPipe, err := d.cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	d.shutdown = signal.NewShutdown(sCtx, cancel)

	if err := d.cmd.Start(); err != nil {
		return nil, err
	}

	sCtx.Go(func(ctx context.Context) error {
		pipeOutputToLogger(stdoutPipe, cfg.Instrumentation.L)
		return nil
	}, signal.WithKey("driver-stdoutPipe"))

	sCtx.Go(func(ctx context.Context) error {
		pipeOutputToLogger(stderrPipe, cfg.Instrumentation.L)
		return nil
	}, signal.WithKey("driver-stderrPipe"))

	sCtx.Go(func(ctx context.Context) error {
		defer func() {
			err = errors.CombineErrors(err, os.Remove(tmpFile.Name()))
			err = errors.CombineErrors(err, os.Remove(cfgFile.Name()))
		}()
		return d.cmd.Wait()
	}, signal.WithKey("driver-wait"))

	return d, err
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
		switch level {
		case "D":
			logger.Debug(filtered)
		case "E", "F":
			logger.Error(filtered)
		case "W":
			logger.Warn(filtered)
		default:
			logger.Info(filtered)

		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal("Error reading data from process", zap.Error(err))
	}
}

func (d *Driver) Stop() error {
	if d.shutdown != nil {
		d.cmd.Process.Signal(syscall.SIGINT)
		err := d.shutdown.Close()
		return err
	}
	return nil
}
