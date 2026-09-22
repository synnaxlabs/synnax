// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Scanner is the integration half of a ScanTask.
type Scanner interface {
	// Check reports whether dev is reachable. It must return when ctx is cancelled.
	Check(ctx context.Context, dev device.Device) error
	// Exec runs a command other than start and stop, such as a connection test, and
	// returns the data that answers it. It returns ErrUnsupportedCommand for a
	// command it does not know.
	Exec(ctx context.Context, cmd task.Command) (msgpack.EncodedJSON, error)
}

// ScanTaskConfig is the configuration for a ScanTask.
type ScanTaskConfig struct {
	// Scanner checks devices and runs commands.
	//
	// [REQUIRED]
	Scanner Scanner
	// Status writes the statuses of the task and of its devices.
	//
	// [REQUIRED]
	Status *status.Service
	// Device retrieves and observes the devices to check.
	//
	// [REQUIRED]
	Device *device.Service
	alamos.Instrumentation
	// Make selects the devices to check: those of this make on the rack of the task.
	//
	// [REQUIRED]
	Make string
	// ReachableMessage is the status message of a device that passes its check.
	//
	// [REQUIRED]
	ReachableMessage string
	// UnreachableMessage is the status message of a device that fails its check. The
	// error of the check is the status description.
	//
	// [REQUIRED]
	UnreachableMessage string
	// Task is the stored task this instance runs.
	//
	// [REQUIRED]
	Task task.Task
	// Interval is the time between checks, and the time limit of one check.
	//
	// [OPTIONAL] - Defaults to 5s.
	Interval time.Duration
}

var (
	_ config.Config[ScanTaskConfig] = ScanTaskConfig{}
	// DefaultScanTaskConfig is the default configuration for a ScanTask.
	DefaultScanTaskConfig = ScanTaskConfig{Interval: 5 * time.Second}
)

// Override implements config.Config.
func (c ScanTaskConfig) Override(other ScanTaskConfig) ScanTaskConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Scanner = override.Nil(c.Scanner, other.Scanner)
	c.Status = override.Nil(c.Status, other.Status)
	c.Device = override.Nil(c.Device, other.Device)
	c.Make = override.String(c.Make, other.Make)
	c.ReachableMessage = override.String(c.ReachableMessage, other.ReachableMessage)
	c.UnreachableMessage = override.String(
		c.UnreachableMessage, other.UnreachableMessage,
	)
	if other.Task.Key != uuid.Nil {
		c.Task = other.Task
	}
	c.Interval = override.Numeric(c.Interval, other.Interval)
	return c
}

// Validate implements config.Config.
func (c ScanTaskConfig) Validate() error {
	v := validate.New("driver.scan_task")
	v.NotNil("scanner", c.Scanner)
	v.NotNil("status", c.Status)
	v.NotNil("device", c.Device)
	v.NotEmptyString("make", c.Make)
	v.NotEmptyString("reachable_message", c.ReachableMessage)
	v.NotEmptyString("unreachable_message", c.UnreachableMessage)
	v.Ternary("task", c.Task.Key == uuid.Nil, "must have a key")
	v.Positive("interval", c.Interval)
	return v.Error()
}

// ScanTask keeps the statuses of the devices of one integration current. While it
// runs, it checks every device of its make on its rack at a fixed interval and when
// such a device changes. It also answers the commands of its Scanner, whether or not
// it runs. Safe for concurrent use.
type ScanTask struct {
	*Runner
	cfg ScanTaskConfig
}

var _ Task = (*ScanTask)(nil)

// NewScanTask validates the configuration and returns a stopped ScanTask.
func NewScanTask(cfgs ...ScanTaskConfig) (*ScanTask, error) {
	cfg, err := config.New(DefaultScanTaskConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	t := &ScanTask{cfg: cfg}
	t.Runner, err = NewRunner(RunnerConfig{
		Status:          cfg.Status,
		Instrumentation: cfg.Instrumentation,
		Task:            cfg.Task,
		Open:            func(context.Context) error { return nil },
		Run:             t.run,
	})
	return t, err
}

// Exec implements Task. A command other than start and stop goes to the Scanner, and
// its result answers the command.
func (t *ScanTask) Exec(ctx context.Context, cmd task.Command) error {
	if cmd.Type == startCommandType || cmd.Type == stopCommandType {
		return t.Runner.Exec(ctx, cmd)
	}
	data, err := t.cfg.Scanner.Exec(ctx, cmd)
	if errors.Is(err, ErrUnsupportedCommand) {
		return err
	}
	variant, message := status.VariantSuccess, "Command executed successfully"
	if err != nil {
		variant, message = status.VariantError, err.Error()
	}
	return errors.Combine(
		err, t.status.Reply(ctx, cmd.Key, variant, message, data),
	)
}

// health is the part of a device status that a check decides.
type health struct {
	variant     status.Variant
	message     string
	description string
}

func (t *ScanTask) run(ctx context.Context) error {
	changed := make(chan struct{}, 1)
	disconnect := t.cfg.Device.Observe().OnChange(func(
		_ context.Context,
		r gorp.TxReader[device.Key, device.Device],
	) {
		for ch := range r {
			if ch.Variant != change.VariantSet || !t.owns(ch.Value) {
				continue
			}
			select {
			case changed <- struct{}{}:
			default:
			}
			return
		}
	})
	defer disconnect()
	ticker := time.NewTicker(t.cfg.Interval)
	defer ticker.Stop()
	reported := make(map[device.Key]health)
	for {
		if err := t.scan(ctx, reported); err != nil && ctx.Err() == nil {
			t.Report(ctx, errors.Wrap(ErrTemporary, err.Error()))
		} else {
			t.Report(ctx, nil)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		case <-changed:
		}
	}
}

func (t *ScanTask) owns(dev device.Device) bool {
	return dev.Make == t.cfg.Make && dev.Rack == t.cfg.Task.Rack
}

// scan checks every owned device and writes the statuses that differ from reported,
// which it updates. It forgets devices that no longer exist, so a device created
// again gets a status.
func (t *ScanTask) scan(ctx context.Context, reported map[device.Key]health) error {
	var devices []device.Device
	if err := t.cfg.Device.NewRetrieve().
		Where(device.And(
			device.MatchMakes(t.cfg.Make),
			device.MatchRacks(t.cfg.Task.Rack),
		)).
		Entries(&devices).
		Exec(ctx, nil); err != nil {
		return err
	}
	results := make([]health, len(devices))
	checkCtx, cancel := context.WithTimeout(ctx, t.cfg.Interval)
	defer cancel()
	var wg sync.WaitGroup
	for i, dev := range devices {
		wg.Go(func() {
			if err := t.cfg.Scanner.Check(checkCtx, dev); err != nil {
				results[i] = health{
					variant:     status.VariantWarning,
					message:     t.cfg.UnreachableMessage,
					description: err.Error(),
				}
				return
			}
			results[i] = health{
				variant: status.VariantSuccess,
				message: t.cfg.ReachableMessage,
			}
		})
	}
	wg.Wait()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	var (
		statuses = make([]device.Status, 0, len(devices))
		present  = make(set.Set[device.Key], len(devices))
		now      = telem.Now()
	)
	for i, dev := range devices {
		present.Add(dev.Key)
		if reported[dev.Key] == results[i] {
			continue
		}
		statuses = append(statuses, device.Status{
			Key:         dev.OntologyID().String(),
			Name:        dev.Name,
			Time:        now,
			Variant:     results[i].variant,
			Message:     results[i].message,
			Description: results[i].description,
			Details:     device.StatusDetails{Rack: dev.Rack, Device: dev.Key},
		})
	}
	for key := range reported {
		if _, ok := present[key]; !ok {
			delete(reported, key)
		}
	}
	if len(statuses) == 0 {
		return nil
	}
	if err := t.cfg.Status.NewWriter(nil).SetMany(ctx, &statuses); err != nil {
		return err
	}
	for i, dev := range devices {
		reported[dev.Key] = results[i]
	}
	t.cfg.L.Debug("updated device statuses", zap.Int("count", len(statuses)))
	return nil
}
