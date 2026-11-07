// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// testConnectionCommandType is the scan task command that pings a proposed cluster.
const testConnectionCommandType = "test_connection"

// testConnectionArgs are the arguments of a test_connection command.
type testConnectionArgs struct {
	// Connection is the cluster to ping.
	Connection Properties `json:"connection"`
}

// scanner pings Kafka cluster devices and answers connection tests.
type scanner struct {
	// timeout bounds one ping.
	timeout telem.TimeSpan
}

var _ driver.Scanner = scanner{}

// Check implements driver.Scanner.
func (s scanner) Check(ctx context.Context, dev device.Device) error {
	props, err := decodeProperties(dev)
	if err != nil {
		return err
	}
	return s.ping(ctx, props)
}

// Exec implements driver.Scanner.
func (s scanner) Exec(
	ctx context.Context,
	cmd task.Command,
) (msgpack.EncodedJSON, error) {
	if cmd.Type != testConnectionCommandType {
		return nil, driver.ErrUnsupportedCommand
	}
	var args testConnectionArgs
	if err := cmd.Args.Unmarshal(&args); err != nil {
		return nil, errors.Wrap(err, "invalid connection")
	}
	if err := s.ping(ctx, args.Connection); err != nil {
		return nil, errors.Wrap(err, "failed to connect")
	}
	return nil, nil
}

func (s scanner) ping(ctx context.Context, props Properties) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout.Duration())
	defer cancel()
	return ping(ctx, props)
}

// configureScan returns the scan task of t, and whether the factory starts it.
func (f *factory) configureScan(t task.Task) (*driver.ScanTask, bool, error) {
	var cfg ScanConfig
	if err := t.Config.Unmarshal(&cfg); err != nil {
		return nil, false, err
	}
	if cfg.Rate <= 0 || cfg.Rate.Period() <= 0 {
		return nil, false, errors.Wrapf(
			validate.ErrValidation, "rate: must be positive, got %s", cfg.Rate,
		)
	}
	st, err := driver.NewScanTask(driver.ScanTaskConfig{
		Instrumentation:    f.cfg.Child("scan"),
		Scanner:            scanner{timeout: f.cfg.PingTimeout},
		DB:                 f.cfg.DB,
		Status:             f.cfg.Status,
		Device:             f.cfg.Device,
		Make:               Make,
		ReachableMessage:   "Device connected",
		UnreachableMessage: "Failed to connect",
		Task:               t,
		Interval:           cfg.Rate.Period().Duration(),
	})
	if err != nil {
		return nil, false, err
	}
	return st, !cfg.Disabled, nil
}
