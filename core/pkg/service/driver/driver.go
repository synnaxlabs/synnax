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
	"encoding/json/v2"
	"fmt"
	"sync"
	"time"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/version"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// Driver is the Go task executor that handles task lifecycle and command processing.
type Driver struct {
	cfg    Config
	closer io.MultiCloser
	rack   rack.Rack
	mu     struct {
		instances map[task.Key]instance
		// failures holds the error of each task that this driver failed to deploy.
		failures map[task.Key]error
		// stopping holds a channel for each task whose deleted instance is stopping.
		// The channel closes when the stop returns. A deploy of the task waits for it.
		stopping map[task.Key]chan struct{}
		sync.RWMutex
	}
}

// instance is a live task and the config hash it was built from. Start commands
// compare the hash against the stored task to decide whether to rebuild.
type instance struct {
	task Task
	hash string
	// deleted is true once a commit deletes the task. A deleted instance takes no
	// commands, a start deploys a new one, and stopDeleted stops it.
	deleted bool
}

// commandSink is a confluence sink that processes incoming command frames.
type commandSink struct {
	confluence.UnarySink[framer.StreamerResponse]
	driver *Driver
	// open is closed on the streamer's open acknowledgment.
	open chan struct{}
}

// Open creates and starts a new Go driver. The driver is fully initialized and ready to
// receive task changes when this function returns. Background goroutines for command
// streaming are started automatically.
func Open(ctx context.Context, cfgs ...Config) (d *Driver, err error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	d = &Driver{cfg: cfg}
	d.mu.instances = make(map[task.Key]instance)
	d.mu.failures = make(map[task.Key]error)
	d.mu.stopping = make(map[task.Key]chan struct{})
	cleanup, ok := service.NewOpener(ctx, &d.closer)
	defer func() { err = cleanup(err) }()

	integrations := make([]string, len(cfg.Factories))
	for i, f := range cfg.Factories {
		integrations[i] = f.Name()
	}

	if d.rack, err = cfg.Rack.RetrieveOrCreateEmbedded(
		ctx,
		fmt.Sprintf("Node %d", cfg.HostKey),
		rack.NewSetAction(rack.SetPayload{
			Path:  []string{"integrations"},
			Value: integrations,
		}),
		rack.NewSetAction(rack.SetPayload{
			Path:  []string{"version"},
			Value: version.Get(),
		}),
	); !ok(err, nil) {
		return nil, err
	}
	cfg.L.Info("created Core driver rack", zap.Stringer("key", d.rack.Key))

	if err = d.createInitialTasks(ctx); !ok(err, nil) {
		return nil, err
	}
	d.startHeartbeat()
	d.configureExistingTasks(ctx)
	// A reader of a commit must not wait for a task to stop, so it only marks the
	// instances of deleted tasks, and a goroutine stops them.
	wake := make(chan struct{}, 1)
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	sCtx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-wake:
			}
			d.stopDeleted()
		}
	}, signal.WithKey("stop_deleted"))
	ok(nil, signal.NewHardShutdown(sCtx, cancel))
	disconnect := cfg.Task.Observe().OnChange(func(
		_ context.Context,
		r gorp.TxReader[task.Key, task.Task],
	) {
		if d.markDeleted(r) {
			select {
			case wake <- struct{}{}:
			default:
			}
		}
	})
	ok(nil, io.NoFailCloserFunc(disconnect))
	if err = d.startCommandStreaming(ctx); !ok(err, nil) {
		return nil, err
	}
	return d, nil
}

func (d *Driver) createInitialTasks(ctx context.Context) error {
	return d.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		w := d.cfg.Task.NewWriter(tx)
		for _, f := range d.cfg.Factories {
			for _, t := range f.InitialTasks() {
				exists, err := d.cfg.Task.NewRetrieve().
					Where(task.And(task.MatchRacks(d.rack.Key), task.MatchTypes(t.Type))).
					Exists(ctx, tx)
				if err != nil {
					return err
				}
				if exists {
					continue
				}
				t.Rack = d.rack.Key
				t.Internal = true
				if err = w.Create(ctx, &t); err != nil {
					return errors.Wrapf(err, "failed to create initial %s task", t.Type)
				}
			}
		}
		return nil
	})
}

func (d *Driver) startHeartbeat() {
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(d.cfg.Instrumentation))
	d.closer = append(d.closer, signal.NewHardShutdown(sCtx, cancel))
	signal.GoTick(
		sCtx,
		d.cfg.HeartbeatInterval,
		func(ctx context.Context, _ time.Time) error {
			if err := d.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
				// Reading the rack each beat means a rename that races a beat is
				// overwritten for at most that one beat.
				var r rack.Rack
				if err := d.cfg.Rack.NewRetrieve().
					Where(rack.MatchKeys(d.rack.Key)).
					Entry(&r).
					Exec(ctx, tx); err != nil {
					return err
				}
				return d.cfg.Status.NewWriter(tx).Set(ctx, &rack.Status{
					Owner:   new(d.rack.Key.ResourceID()),
					Name:    r.Name,
					Time:    telem.Now(),
					Variant: status.VariantSuccess,
					Message: "Driver is running",
					Details: rack.StatusDetails{Running: true},
				})
			}); err != nil {
				d.cfg.L.Error("failed to update rack status", zap.Error(err))
			}
			return nil
		},
	)
}

// startCommandStreaming initializes the command channel streamer. This is optional and
// will log warnings if the command channel doesn't exist or streaming fails.
func (d *Driver) startCommandStreaming(ctx context.Context) error {
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(d.cfg.Instrumentation))
	d.closer = append(d.closer, signal.NewGracefulShutdown(sCtx, cancel))
	streamer, err := d.cfg.Framer.NewStreamer(ctx, framer.StreamerConfig{
		Keys:        channel.Keys{d.cfg.Task.CommandChannelKey()},
		SendOpenAck: true,
	})
	if err != nil {
		return err
	}
	p := plumber.New()
	p.SetSegment[framer.StreamerRequest, framer.StreamerResponse]("streamer", streamer)
	open := make(chan struct{})
	sink := &commandSink{driver: d, open: open}
	sink.Sink = sink.process
	p.SetSink[framer.StreamerResponse]("driver", sink)
	p.MustConnect[framer.StreamerResponse]("streamer", "driver", 10)
	streamerRequests := confluence.NewStream[framer.StreamerRequest]()
	streamer.InFrom(streamerRequests)
	d.closer = append(d.closer, io.NoFailCloserFunc(streamerRequests.Close))
	p.Flow(
		sCtx,
		confluence.CloseOutputInletsOnExit(),
		confluence.RecoverWithErrOnPanic(),
	)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-open:
		return nil
	}
}

func (s *commandSink) process(ctx context.Context, res framer.StreamerResponse) error {
	if s.open != nil {
		close(s.open)
		s.open = nil
		return nil
	}
	s.driver.processCommand(ctx, res.Frame)
	return nil
}

func (d *Driver) processCommand(ctx context.Context, frame framer.Frame) {
	var cmd task.Command
	for series := range frame.Series() {
		for s := range series.Samples() {
			if err := json.Unmarshal(s, &cmd); err != nil {
				d.cfg.L.Error("failed to unmarshal command", zap.Error(err))
				continue
			}
			if cmd.Type == startCommandType {
				d.handleStart(ctx, cmd)
				continue
			}
			// Non-start commands go to whichever driver holds the live
			// instance, even when the stored row has moved to another rack.
			d.mu.RLock()
			inst, ok := d.mu.instances[cmd.Task]
			deployErr := d.mu.failures[cmd.Task]
			d.mu.RUnlock()
			if ok && !inst.deleted {
				d.exec(ctx, inst.task, cmd)
				continue
			}
			if cmd.Key != NoCommand {
				d.refuseCommand(ctx, cmd, deployErr)
			}
		}
	}
}

const (
	// startCommandType deploys the stored task row before running it.
	startCommandType = "start"
	stopCommandType  = "stop"
)

// handleStart deploys the latest stored config before running the task: when no live
// instance exists or the stored config differs from the one the instance was built
// from, the task is rebuilt first, then the start command executes. A pinned start
// whose hash differs from the stored config's is refused.
func (d *Driver) handleStart(ctx context.Context, cmd task.Command) {
	var tsk task.Task
	if err := d.cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return d.cfg.Task.NewRetrieve().
			Where(task.MatchKeys(cmd.Task)).
			Entry(&tsk).
			Exec(ctx, snap)
	}); err != nil {
		// A not-found task was deleted: a status written now would recreate the
		// one the delete removed. Other errors answer the start so the sender
		// does not wait forever.
		if !errors.Is(err, query.ErrNotFound) {
			d.cfg.L.Error("failed to retrieve task for start",
				zap.Stringer("task", cmd.Task),
				zap.Error(err),
			)
			d.ackFailure(ctx, task.Task{Key: cmd.Task}, cmd, err)
		}
		return
	}
	if tsk.Snapshot {
		return
	}
	if tsk.Rack != d.rack.Key {
		// The task moved racks: the start deploys it on the new rack, and doubles
		// as the teardown signal for the instance this driver still holds. The
		// teardown is silent: the new rack's driver owns status reporting.
		if d.release(cmd.Task, false) {
			d.cfg.L.Info("stopped task moved to another rack",
				zap.Stringer("task", cmd.Task),
			)
		}
		return
	}
	if cmd.Pinned && cmd.ConfigHash != tsk.ConfigHash {
		d.refusePinnedStart(ctx, tsk, cmd)
		return
	}
	d.mu.RLock()
	inst, ok := d.mu.instances[cmd.Task]
	d.mu.RUnlock()
	if !ok || inst.deleted || inst.hash != tsk.ConfigHash {
		if err := d.configure(ctx, tsk, cmd.Key); err != nil {
			d.cfg.L.Error("failed to configure task",
				zap.Stringer("task", tsk),
				zap.Error(err),
			)
			// Factories answer their own failures. Neither of these reaches one:
			// an unhandled type never gets to a factory, and a factory that ran
			// out of time has no live context to write with.
			if errors.Is(err, ErrTaskNotHandled) ||
				errors.Is(err, context.DeadlineExceeded) {
				d.ackFailure(ctx, tsk, cmd, err)
			}
			return
		}
		d.mu.RLock()
		inst, ok = d.mu.instances[cmd.Task]
		d.mu.RUnlock()
		if !ok {
			return
		}
	}
	d.exec(ctx, inst.task, cmd)
}

// ackFailure acknowledges a start command whose deploy failed without a factory
// reporting it.
func (d *Driver) ackFailure(
	ctx context.Context,
	t task.Task,
	cmd task.Command,
	err error,
) {
	if sErr := NewStatusHandler(d.cfg.DB, d.cfg.Status, t).Send(
		ctx,
		cmd.Key,
		status.VariantError,
		false,
		"Failed to start task",
		err.Error(),
	); sErr != nil {
		d.cfg.L.Error("failed to write start failure status", zap.Error(sErr))
	}
}

// refusePinnedStart answers a pinned start whose config changed after the sender saw
// it. The live instance runs on as it was.
func (d *Driver) refusePinnedStart(ctx context.Context, t task.Task, cmd task.Command) {
	if err := d.answer(ctx, t, cmd.Key, func(stat *task.Status) {
		stat.Variant = status.VariantWarning
		stat.Message = "Config changed"
		stat.Description = "The task config changed after you saw it. Review the " +
			"task, then start it again."
	}); err != nil {
		d.cfg.L.Error("failed to write refused start status", zap.Error(err))
	}
}

// refuseCommand answers cmd for a task with no live instance here. deployErr is the
// error of a failed deploy here, or nil. It leaves cmd to other drivers when the task
// is on another rack, or when the task moved here while another driver still runs it.
func (d *Driver) refuseCommand(
	ctx context.Context,
	cmd task.Command,
	deployErr error,
) {
	var tsk task.Task
	if err := d.cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return d.cfg.Task.NewRetrieve().
			Where(task.MatchKeys(cmd.Task)).
			Entry(&tsk).
			Exec(ctx, snap)
	}); err != nil {
		// A status written for a deleted task would recreate the one the delete
		// removed.
		if !errors.Is(err, query.ErrNotFound) {
			d.cfg.L.Error("failed to retrieve task for refused command", zap.Error(err))
		}
		return
	}
	if tsk.Rack != d.rack.Key {
		return
	}
	description := "The task is not running on this Driver. Start the task first."
	if deployErr != nil {
		description = deployErr.Error()
	} else {
		var stat task.Status
		if err := d.cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
			return d.cfg.Status.NewRetrieve[task.StatusDetails]().
				Where(status.MatchOwners[task.StatusDetails](tsk.ResourceID())).
				Entry(&stat).
				Exec(ctx, snap)
		}); err != nil && !errors.Is(err, query.ErrNotFound) {
			d.cfg.L.Error(
				"failed to retrieve status for refused command",
				zap.Error(err),
			)
			return
		}
		// The task moved here while another driver still runs it.
		if stat.Details.Running {
			return
		}
	}
	if err := d.answer(ctx, tsk, cmd.Key, func(stat *task.Status) {
		stat.Variant = status.VariantError
		stat.Message = "Task is not running"
		stat.Description = description
		stat.Details.Running = false
	}); err != nil {
		d.cfg.L.Error("failed to write refused command status", zap.Error(err))
	}
}

// answer writes a status for t that answers cmdKey. It starts from the stored status,
// so the name, running state, and config hash stay as they were, and edit sets the
// rest.
func (d *Driver) answer(
	ctx context.Context,
	t task.Task,
	cmdKey string,
	edit func(*task.Status),
) error {
	return d.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		stat := task.Status{Owner: new(t.ResourceID()), Name: t.Name}
		if err := d.cfg.Status.NewRetrieve[task.StatusDetails]().
			Where(status.MatchOwners[task.StatusDetails](*stat.Owner)).
			Entry(&stat).
			Exec(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
			return err
		}
		edit(&stat)
		stat.Details.Rack = t.Rack
		stat.Details.Cmd = cmdKey
		stat.Details.Data = nil
		stat.Time = telem.Now()
		return d.cfg.Status.NewWriter(tx).Set(ctx, &stat)
	})
}

// exec runs cmd on t. A task answers the commands it runs, so an error from t means no
// one answered cmd, and exec answers it.
func (d *Driver) exec(ctx context.Context, t Task, cmd task.Command) {
	sCtx, cancel := signal.WithTimeout(
		ctx,
		d.cfg.TaskTimeout,
		signal.WithInstrumentation(d.cfg.Instrumentation),
	)
	defer cancel()
	sCtx.Go(
		func(ctx context.Context) error { return t.Exec(ctx, cmd) },
		signal.RecoverWithErrOnPanic(),
	)
	err := sCtx.Wait()
	if err == nil {
		return
	}
	description := err.Error()
	switch {
	case errors.Is(err, ErrUnsupportedCommand):
		d.cfg.L.Warn(
			"unsupported command",
			zap.Stringer("command", cmd),
			zap.Stringer("task", cmd.Task),
		)
		description = unsupportedDescription(cmd.Type)
	case errors.Is(err, context.DeadlineExceeded):
		d.cfg.L.Error("command timed out", zap.Stringer("command", cmd))
		description = fmt.Sprintf(
			"The task did not finish the command within %s.",
			d.cfg.TaskTimeout,
		)
	default:
		d.cfg.L.Error("failed to execute command",
			zap.Stringer("command", cmd),
			zap.Error(err),
		)
	}
	var tsk task.Task
	if err = d.cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return d.cfg.Task.NewRetrieve().
			Where(task.MatchKeys(cmd.Task)).
			Entry(&tsk).
			Exec(ctx, snap)
	}); err != nil {
		// A status written for a deleted task would recreate the one the delete
		// removed.
		if !errors.Is(err, query.ErrNotFound) {
			d.cfg.L.Error("failed to retrieve task for command failure", zap.Error(err))
		}
		return
	}
	if err = d.answer(ctx, tsk, cmd.Key, func(stat *task.Status) {
		stat.Variant = status.VariantError
		stat.Message = failureMessage(cmd.Type)
		stat.Description = description
	}); err != nil {
		d.cfg.L.Error("failed to write command failure status", zap.Error(err))
	}
}

// failureMessage returns the message of an error status that answers a command of
// cmdType.
func failureMessage(cmdType string) string {
	switch cmdType {
	case startCommandType:
		return "Failed to start task"
	case stopCommandType:
		return "Failed to stop task"
	default:
		return "Failed to execute command"
	}
}

// unsupportedDescription returns the description of an error status that answers a
// command of cmdType, which the task does not support.
func unsupportedDescription(cmdType string) string {
	return fmt.Sprintf("The task does not accept the '%s' command.", cmdType)
}

func (d *Driver) configureExistingTasks(ctx context.Context) {
	var tasks []task.Task
	if err := d.cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return d.cfg.Task.NewRetrieve().
			Where(task.And(task.MatchRacks(d.rack.Key), task.MatchSnapshot(false))).
			Entries(&tasks).
			Exec(ctx, snap)
	}); err != nil {
		d.cfg.L.Error("failed to retrieve existing tasks", zap.Error(err))
		return
	}
	d.cfg.L.Info("configuring existing tasks", zap.Int("count", len(tasks)))
	sCtx, cancel := signal.WithTimeout(
		ctx,
		d.cfg.TaskTimeout,
		signal.WithInstrumentation(d.cfg.Instrumentation),
	)
	defer cancel()
	for _, t := range tasks {
		sCtx.Go(
			func(ctx context.Context) error {
				err := d.configure(ctx, t, NoCommand)
				canceled := ctx.Err() != nil && errors.Is(err, context.Canceled)
				if err != nil && !canceled {
					d.cfg.L.Error("failed to configure task",
						zap.Stringer("task", t),
						zap.Error(err),
					)
				}
				return nil
			},
			signal.RecoverWithErrOnPanic(),
		)
	}
	if err := sCtx.Wait(); err != nil {
		d.cfg.L.Error("timed out configuring existing tasks", zap.Error(err))
	}
}

func (d *Driver) configure(ctx context.Context, t task.Task, cmdKey string) error {
	d.mu.Lock()
	existing, hadExisting := d.mu.instances[t.Key]
	delete(d.mu.instances, t.Key)
	delete(d.mu.failures, t.Key)
	stopping := d.mu.stopping[t.Key]
	d.mu.Unlock()
	if stopping != nil {
		select {
		case <-stopping:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if hadExisting {
		if err := existing.task.Stop(false); err != nil {
			d.cfg.L.Error("failed to stop existing task for reconfiguration",
				zap.Stringer("task", t),
				zap.Error(err),
			)
		}
	}

	sCtx, cancel := signal.WithTimeout(
		ctx, d.cfg.TaskTimeout, signal.WithInstrumentation(d.cfg.Instrumentation),
	)
	defer cancel()

	sCtx.Go(func(ctx context.Context) error {
		// A factory can start the task, and a canceled Open stops no task.
		if err := ctx.Err(); err != nil {
			return err
		}
		for _, f := range d.cfg.Factories {
			newTask, err := f.ConfigureTask(ctx, t, cmdKey)
			if errors.Is(err, ErrTaskNotHandled) {
				continue
			}
			if err != nil {
				return err
			}
			d.mu.Lock()
			d.mu.instances[t.Key] = instance{task: newTask, hash: t.ConfigHash}
			d.mu.Unlock()
			d.cfg.L.Info("configured task", zap.Stringer("task", t))
			return nil
		}
		return errors.Wrapf(ErrTaskNotHandled, "task type '%s'", t.Type)
	}, signal.RecoverWithErrOnPanic())
	err := sCtx.Wait()
	if err != nil {
		d.mu.Lock()
		d.mu.failures[t.Key] = err
		d.mu.Unlock()
	}
	return err
}

// markDeleted marks the instance of each task that r deletes and forgets a failed
// deploy of it. It reports whether it marked an instance.
func (d *Driver) markDeleted(r gorp.TxReader[task.Key, task.Task]) bool {
	marked := false
	for ch := range r {
		if ch.Kind != gorp.ChangeDelete {
			continue
		}
		d.mu.Lock()
		if inst, ok := d.mu.instances[ch.Key]; ok {
			inst.deleted = true
			d.mu.instances[ch.Key] = inst
			marked = true
		}
		delete(d.mu.failures, ch.Key)
		d.mu.Unlock()
	}
	return marked
}

// stopDeleted stops and forgets every instance that markDeleted marked. The stop is
// silent: a terminal status written for a deleted task would recreate its status.
func (d *Driver) stopDeleted() {
	stops := make(map[task.Key]Task)
	d.mu.Lock()
	for key, inst := range d.mu.instances {
		if inst.deleted {
			stops[key] = inst.task
			delete(d.mu.instances, key)
			d.mu.stopping[key] = make(chan struct{})
		}
	}
	d.mu.Unlock()
	for key, t := range stops {
		err := t.Stop(false)
		d.mu.Lock()
		close(d.mu.stopping[key])
		delete(d.mu.stopping, key)
		d.mu.Unlock()
		if err != nil {
			d.cfg.L.Error(
				"failed to stop task",
				zap.Stringer("task", key),
				zap.Error(err),
			)
			continue
		}
		d.cfg.L.Info("deleted task", zap.Stringer("task", key))
	}
}

// release stops and forgets the live instance for key, reporting whether one existed.
// It also forgets a failed deploy of key.
func (d *Driver) release(key task.Key, sendStatus bool) bool {
	d.mu.Lock()
	inst, ok := d.mu.instances[key]
	delete(d.mu.instances, key)
	delete(d.mu.failures, key)
	d.mu.Unlock()
	if !ok {
		return false
	}
	if err := inst.task.Stop(sendStatus); err != nil {
		d.cfg.L.Error(
			"failed to stop task",
			zap.Stringer("task", key),
			zap.Error(err),
		)
	}
	return true
}

func (d *Driver) Close() error {
	// The command sink drains first, so no command deploys a task after the stops.
	err := d.closer.Close()
	d.mu.Lock()
	instances := d.mu.instances
	d.mu.instances = make(map[task.Key]instance)
	d.mu.Unlock()
	for key, inst := range instances {
		if err := inst.task.Stop(!inst.deleted); err != nil {
			d.cfg.L.Error(
				"failed to stop task during shutdown",
				zap.Stringer("task", key),
				zap.Error(err),
			)
		}
	}
	return err
}
