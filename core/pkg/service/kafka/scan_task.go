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
	"io"
	"time"

	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// testConnectionCommandType is the scan task command that pings a proposed cluster.
const testConnectionCommandType = "test_connection"

// testConnectionArgs are the arguments of a test_connection command.
type testConnectionArgs struct {
	// Connection is the cluster to ping.
	Connection Properties `json:"connection"`
}

// scanTask pings the Kafka cluster devices on its rack and answers connection tests.
type scanTask struct {
	factoryCfg FactoryConfig
	task       task.Task
	cfg        ScanConfig
	// status is the authority on this instance's current status.
	status *driver.StatusHandler
	// closer stops the health check loop, nil while it is not running.
	closer io.Closer
	// lastWritten holds the last status written per device, so an unchanged health
	// result is not rewritten.
	lastWritten map[device.Key]string
}

var _ driver.Task = (*scanTask)(nil)

func (f *factory) configureScan(_ context.Context, t task.Task) (driver.Task, error) {
	var cfg ScanConfig
	if err := t.Config.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	cfg.ApplyDefaults()
	st := &scanTask{
		factoryCfg:  f.cfg,
		task:        t,
		cfg:         cfg,
		status:      driver.NewStatusHandler(f.cfg.DB, f.cfg.Status, t),
		lastWritten: make(map[device.Key]string),
	}
	if !cfg.Disabled {
		st.startLoop()
	}
	return st, nil
}

// Exec implements driver.Task.
func (t *scanTask) Exec(ctx context.Context, cmd task.Command) error {
	switch cmd.Type {
	case "start":
		t.startLoop()
		return t.status.Send(ctx, cmd.Key, status.VariantSuccess, true, "Scan started")
	case "stop":
		return t.stop(ctx, cmd.Key, true)
	case testConnectionCommandType:
		return t.testConnection(ctx, cmd)
	default:
		return driver.ErrUnsupportedCommand
	}
}

// Stop implements driver.Task.
func (t *scanTask) Stop(sendStatus bool) error {
	return t.stop(context.TODO(), driver.NoCommand, sendStatus)
}

func (t *scanTask) stop(ctx context.Context, cmdKey string, sendStatus bool) error {
	if t.closer != nil {
		err := t.closer.Close()
		t.closer = nil
		if err != nil {
			return err
		}
	}
	if !sendStatus {
		return nil
	}
	return t.status.Send(ctx, cmdKey, status.VariantSuccess, false, "Scan stopped")
}

func (t *scanTask) startLoop() {
	if t.closer != nil {
		return
	}
	sCtx, cancel := signal.Isolated(
		signal.WithInstrumentation(t.factoryCfg.Instrumentation),
	)
	sCtx.Go(t.loop, signal.RecoverWithErrOnPanic())
	t.closer = signal.NewHardShutdown(sCtx, cancel)
}

func (t *scanTask) loop(ctx context.Context) error {
	ticker := time.NewTicker(t.cfg.Rate.Period().Duration())
	defer ticker.Stop()
	for {
		t.scan(ctx)
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// scan pings every Kafka cluster device on the rack and records the outcome as the
// device's status.
func (t *scanTask) scan(ctx context.Context) {
	var devices []device.Device
	if err := t.factoryCfg.Device.NewRetrieve().
		Where(device.And(
			device.MatchRacks(t.task.Rack),
			device.MatchMakes(Make),
		)).
		Entries(&devices).
		Exec(ctx, nil); err != nil {
		t.factoryCfg.L.Error("failed to list Kafka devices", zap.Error(err))
		return
	}
	for _, dev := range devices {
		if ctx.Err() != nil {
			return
		}
		variant, message := t.check(ctx, dev)
		t.setDeviceStatus(ctx, dev, variant, message)
	}
}

// check pings dev and returns the status variant and message describing the result.
func (t *scanTask) check(
	ctx context.Context,
	dev device.Device,
) (status.Variant, string) {
	props, err := decodeProperties(dev)
	if err != nil {
		return status.VariantError, "Invalid connection properties: " + err.Error()
	}
	if err := t.ping(ctx, props); err != nil {
		return status.VariantError, "Connection failed: " + err.Error()
	}
	return status.VariantSuccess, "Device connected"
}

func (t *scanTask) ping(ctx context.Context, props Properties) error {
	ctx, cancel := context.WithTimeout(ctx, t.factoryCfg.PingTimeout.Duration())
	defer cancel()
	return ping(ctx, props)
}

func (t *scanTask) setDeviceStatus(
	ctx context.Context,
	dev device.Device,
	variant status.Variant,
	message string,
) {
	written := string(variant) + ":" + message
	if t.lastWritten[dev.Key] == written {
		return
	}
	stat := device.Status{
		Key:     dev.OntologyID().String(),
		Name:    dev.Name,
		Variant: variant,
		Message: message,
		Time:    telem.Now(),
		Details: device.StatusDetails{Rack: dev.Rack, Device: dev.Key},
	}
	if err := t.factoryCfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return t.factoryCfg.Status.NewWriter(tx).Set(ctx, &stat)
	}); err != nil {
		t.factoryCfg.L.Error("failed to set device status",
			zap.String("device", dev.Key),
			zap.Error(err),
		)
		return
	}
	t.lastWritten[dev.Key] = written
}

// testConnection answers cmd with the outcome of pinging the cluster in its args.
func (t *scanTask) testConnection(ctx context.Context, cmd task.Command) error {
	running := t.closer != nil
	var args testConnectionArgs
	if err := cmd.Args.Unmarshal(&args); err != nil {
		return t.status.Send(ctx, cmd.Key, status.VariantError, running,
			"Invalid connection: "+errors.Wrap(err, "parsing arguments").Error())
	}
	args.Connection.ApplyDefaults()
	if err := t.ping(ctx, args.Connection); err != nil {
		return t.status.Send(ctx, cmd.Key, status.VariantError, running,
			"Connection failed: "+err.Error())
	}
	return t.status.Send(ctx, cmd.Key, status.VariantSuccess, running,
		"Connection successful")
}
