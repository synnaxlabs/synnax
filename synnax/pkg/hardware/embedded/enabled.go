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
	"context"
	"embed"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
	"io"
	"log"
	"os"
	"os/exec"
)

//go:embed assets/driver
var executable embed.FS

func OpenDriver(cfgs ...Config) (*Driver, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
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

	d := &Driver{cmd: exec.Command(tmpFile.Name())}
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
		}()
		return d.cmd.Wait()
	}, signal.WithKey("driver-wait"))

	return d, err
}

func pipeOutputToLogger(reader io.ReadCloser, logger *alamos.Logger) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		logger.Info(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatal("Error reading data from process", zap.Error(err))
	}
}

func (d *Driver) Stop() error {
	if d.shutdown != nil {
		d.cmd.Process.Signal(os.Interrupt)
		return d.shutdown.Close()
	}
	return nil
}
