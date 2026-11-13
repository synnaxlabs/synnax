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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

const (
	// testConnectionCommand tests a connection to the cluster that its arguments
	// describe, one stage at a time.
	testConnectionCommand = "test_connection"
	// browseCommand lists the topics of a cluster device.
	browseCommand = "browse"
	// sampleCommand reads records from topics of a cluster device and infers their
	// shapes.
	sampleCommand = "sample"
)

// testConnectionArgs are the arguments of a test_connection command.
type testConnectionArgs struct {
	// Connection holds the properties of the cluster to test. It need not be a stored
	// device yet.
	Connection msgpack.EncodedJSON `json:"connection"`
}

// browseArgs are the arguments of a browse command.
type browseArgs struct {
	// Device is the key of the cluster device to browse.
	Device device.Key `json:"device"`
}

// scanner pings Kafka cluster devices and answers the commands of the scan task.
type scanner struct {
	// timeout bounds one ping.
	timeout telem.TimeSpan
	db      *gorp.DB
	device  *device.Service
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
	switch cmd.Type {
	case testConnectionCommand:
		var args testConnectionArgs
		if err := unmarshalArgs(cmd, &args); err != nil {
			return nil, err
		}
		return nil, testConnection(ctx, args.Connection)
	case browseCommand:
		var args browseArgs
		if err := unmarshalArgs(cmd, &args); err != nil {
			return nil, err
		}
		props, err := resolveProperties(ctx, s.db, s.device, args.Device)
		if err != nil {
			return nil, err
		}
		topics, err := browseTopics(ctx, props)
		if err != nil {
			return nil, err
		}
		kept, truncated, err := driver.Fit(browseResult{}, [][]browsedTopic{topics})
		if err != nil {
			return nil, err
		}
		return msgpack.NewEncodedJSON(
			browseResult{Topics: kept[0], Truncated: truncated},
		)
	case sampleCommand:
		var args sampleArgs
		if err := unmarshalArgs(cmd, &args); err != nil {
			return nil, err
		}
		props, err := resolveProperties(ctx, s.db, s.device, args.Device)
		if err != nil {
			return nil, err
		}
		res, err := sampleTopics(ctx, props, args)
		if err != nil {
			return nil, err
		}
		return msgpack.NewEncodedJSON(res)
	}
	return nil, driver.ErrUnsupportedCommand
}

// unmarshalArgs decodes the arguments of cmd into args.
func unmarshalArgs(cmd task.Command, args any) error {
	if err := cmd.Args.Unmarshal(args); err != nil {
		return errors.Wrapf(validate.ErrValidation, "invalid arguments: %s", err)
	}
	return nil
}

func (s scanner) ping(ctx context.Context, props Properties) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout.Duration())
	defer cancel()
	return ping(ctx, props)
}

func validateScanConfig(cfg *ScanConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if cfg.Rate.Period() <= 0 {
		return errors.Wrapf(
			validate.ErrValidation, "rate: %s is too high", cfg.Rate,
		)
	}
	return nil
}

// configureScan returns the scan task of t, and whether the factory starts it.
func (f *factory) configureScan(t task.Task) (*driver.ScanTask, bool, error) {
	var cfg ScanConfig
	if err := t.Config.Unmarshal(&cfg); err != nil {
		return nil, false, err
	}
	if err := validateScanConfig(&cfg); err != nil {
		return nil, false, err
	}
	st, err := driver.NewScanTask(driver.ScanTaskConfig{
		Instrumentation: f.cfg.Child("scan"),
		Scanner: scanner{
			timeout: f.cfg.PingTimeout,
			db:      f.cfg.DB,
			device:  f.cfg.Device,
		},
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
