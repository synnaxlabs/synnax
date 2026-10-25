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
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
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
	// command it does not know. The task runs one Exec at a time, apart from the
	// command loop of the driver, and cancels ctx when the task stops.
	Exec(ctx context.Context, cmd task.Command) (msgpack.EncodedJSON, error)
}

// ScanTaskConfig is the configuration for a ScanTask.
type ScanTaskConfig struct {
	// Scanner checks devices and runs commands.
	//
	// [REQUIRED]
	Scanner Scanner
	// DB opens the transactions that status writes run in.
	//
	// [REQUIRED]
	DB *gorp.DB
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
	c.DB = override.Nil(c.DB, other.DB)
	c.Status = override.Nil(c.Status, other.Status)
	c.Device = override.Nil(c.Device, other.Device)
	c.Make = override.String(c.Make, other.Make)
	c.ReachableMessage = override.String(c.ReachableMessage, other.ReachableMessage)
	c.UnreachableMessage = override.String(
		c.UnreachableMessage, other.UnreachableMessage,
	)
	if other.Task.Key != uuid.Nil() {
		c.Task = other.Task
	}
	c.Interval = override.Numeric(c.Interval, other.Interval)
	return c
}

// Validate implements config.Config.
func (c ScanTaskConfig) Validate() error {
	v := validate.New("driver.scan_task")
	v.NotNil("scanner", c.Scanner)
	v.NotNil("db", c.DB)
	v.NotNil("status", c.Status)
	v.NotNil("device", c.Device)
	v.NotEmptyString("make", c.Make)
	v.NotEmptyString("reachable_message", c.ReachableMessage)
	v.NotEmptyString("unreachable_message", c.UnreachableMessage)
	v.Ternary("task", c.Task.Key == uuid.Nil(), "must have a key")
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
	// turn holds a token while a command of the scanner runs.
	turn     chan struct{}
	commands struct {
		// sCtx runs the commands of the scanner. It is nil when no command ran since
		// the last stop.
		sCtx   signal.Context
		cancel context.CancelFunc
		sync.Mutex
	}
}

var _ Task = (*ScanTask)(nil)

// NewScanTask validates the configuration and returns a stopped ScanTask.
func NewScanTask(cfgs ...ScanTaskConfig) (*ScanTask, error) {
	cfg, err := config.New(DefaultScanTaskConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	t := &ScanTask{cfg: cfg, turn: make(chan struct{}, 1)}
	t.Runner, err = NewRunner(RunnerConfig{
		DB:              cfg.DB,
		Status:          cfg.Status,
		Instrumentation: cfg.Instrumentation,
		Task:            cfg.Task,
		Open: func(context.Context) (func(context.Context) error, error) {
			return t.run, nil
		},
	})
	if err != nil {
		return nil, err
	}
	return t, nil
}

// Exec implements Task. A command other than start and stop goes to the Scanner in the
// background, and Exec returns at once. The result of the Scanner answers the command,
// within the time limit of ctx, which starts when the command gets its turn. A command
// that the Scanner does not support gets no answer.
func (t *ScanTask) Exec(ctx context.Context, cmd task.Command) error {
	if cmd.Type == startCommandType || cmd.Type == stopCommandType {
		return t.Runner.Exec(ctx, cmd)
	}
	deadline, limited := ctx.Deadline()
	limit := time.Until(deadline)
	t.commands.Lock()
	defer t.commands.Unlock()
	if t.commands.sCtx == nil {
		t.commands.sCtx, t.commands.cancel = signal.Isolated(
			signal.WithInstrumentation(t.cfg.Instrumentation),
		)
	}
	t.commands.sCtx.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return nil
		case t.turn <- struct{}{}:
		}
		defer func() { <-t.turn }()
		if limited {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, limit)
			defer cancel()
		}
		t.execScanner(ctx, cmd)
		return nil
	}, signal.RecoverWithErrOnPanic())
	return nil
}

// execScanner runs cmd on the Scanner and answers it. A command cancelled by a stop
// gets no answer, since the successor of the task owns its status.
func (t *ScanTask) execScanner(ctx context.Context, cmd task.Command) {
	data, err := t.cfg.Scanner.Exec(ctx, cmd)
	if errors.Is(err, ErrUnsupportedCommand) {
		t.cfg.L.Warn(
			"unsupported command",
			zap.Stringer("command", cmd),
			zap.Stringer("task", cmd.Task),
		)
		return
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return
	}
	variant, message := status.VariantSuccess, "Command executed successfully"
	if err != nil {
		variant, message = status.VariantError, err.Error()
	}
	if rErr := t.status.Reply(
		context.WithoutCancel(ctx), cmd.Key, variant, message, data,
	); rErr != nil {
		t.cfg.L.Error("failed to answer command", zap.Error(rErr))
	}
}

// Stop implements Task. It also cancels the commands of the Scanner and waits for them
// to return.
func (t *ScanTask) Stop(sendStatus bool) error {
	err := t.Runner.Stop(sendStatus)
	t.commands.Lock()
	sCtx, cancel := t.commands.sCtx, t.commands.cancel
	t.commands.sCtx, t.commands.cancel = nil, nil
	t.commands.Unlock()
	if sCtx == nil {
		return err
	}
	cancel()
	return errors.Combine(err, errors.Skip(sCtx.Wait(), context.Canceled))
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
	for {
		err := t.scan(ctx)
		if ctx.Err() != nil {
			err = nil
		}
		t.Report(ctx, "Failed to scan for devices", err)
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

// scan checks every owned device and updates each stored status that differs from the
// result. A device write can store an older status after a scan, so the next scan
// compares against storage.
func (t *ScanTask) scan(ctx context.Context) error {
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
	var count int
	if err := t.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		stored, err := t.storedHealth(ctx, tx, devices)
		if err != nil {
			return err
		}
		var (
			statuses = make([]device.Status, 0, len(devices))
			now      = telem.Now()
		)
		for i, dev := range devices {
			// A device with no status was deleted after the retrieve, or lost its
			// status, which its next Create restores.
			h, ok := stored[dev.OntologyID().String()]
			if !ok || h == results[i] {
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
		count = len(statuses)
		if count == 0 {
			return nil
		}
		return t.cfg.Status.NewWriter(tx).SetMany(ctx, &statuses)
	}); err != nil {
		return err
	}
	if count > 0 {
		t.cfg.L.Debug("updated device statuses", zap.Int("count", count))
	}
	return nil
}

// storedHealth returns the health in the stored status of each device, by status key. A
// device with no stored status has no entry.
func (t *ScanTask) storedHealth(
	ctx context.Context,
	tx gorp.Tx,
	devices []device.Device,
) (map[status.Key]health, error) {
	keys := make([]status.Key, len(devices))
	for i, dev := range devices {
		keys[i] = dev.OntologyID().String()
	}
	var statuses []device.Status
	if err := errors.Skip(
		t.cfg.Status.NewRetrieve[device.StatusDetails]().
			Where(status.MatchKeys[device.StatusDetails](keys...)).
			Entries(&statuses).
			Exec(ctx, tx),
		query.ErrNotFound,
	); err != nil {
		return nil, err
	}
	stored := make(map[status.Key]health, len(statuses))
	for _, s := range statuses {
		stored[s.Key] = health{
			variant:     s.Variant,
			message:     s.Message,
			description: s.Description,
		}
	}
	return stored, nil
}
