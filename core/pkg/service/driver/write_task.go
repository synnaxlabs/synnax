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
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
)

// Sink opens the writes of a WriteTask.
type Sink interface {
	// Open acquires what the writes of one run need and returns the Writer that owns
	// it. Its error answers the start command.
	Open(ctx context.Context) (Writer, error)
}

// Writer sends the frames that one run of a WriteTask streams from the Core to hardware
// or a network service. The task calls Write and Health from different goroutines.
type Writer interface {
	// Write sends one frame. For an error that matches ErrTemporary or ErrDegraded the
	// task warns, drops the frame, and keeps running: a command that waits out an
	// outage is stale when it arrives. Any other error stops the task.
	Write(ctx context.Context, fr framer.Frame) error
	// Health blocks until the health of the writer changes or ctx is cancelled. A nil
	// error reports a healthy writer. For an error that matches ErrTemporary the task
	// warns until the next nil error. Any other error stops the task.
	Health(ctx context.Context) error
	// Close releases what Open acquired. The task calls it after the last Write and
	// Health return.
	Close() error
}

// WriteTaskConfig is the configuration for a WriteTask.
type WriteTaskConfig struct {
	// Sink receives the streamed frames.
	//
	// [REQUIRED]
	Sink Sink
	// DB opens the transactions that status writes run in.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Status writes the statuses of the task.
	//
	// [REQUIRED]
	Status *status.Service
	// Framer opens the streamer of the task.
	//
	// [REQUIRED]
	Framer *framer.Service
	alamos.Instrumentation
	// Channels are the keys of the channels to stream to the sink.
	//
	// [REQUIRED]
	Channels channel.Keys
	// Task is the stored task this instance runs.
	//
	// [REQUIRED]
	Task task.Task
}

var _ config.Config[WriteTaskConfig] = WriteTaskConfig{}

// Override implements config.Config.
func (c WriteTaskConfig) Override(other WriteTaskConfig) WriteTaskConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Sink = override.Nil(c.Sink, other.Sink)
	c.DB = override.Nil(c.DB, other.DB)
	c.Status = override.Nil(c.Status, other.Status)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Channels = override.Slice(c.Channels, other.Channels)
	if other.Task.Key != uuid.Nil() {
		c.Task = other.Task
	}
	return c
}

// Validate implements config.Config.
func (c WriteTaskConfig) Validate() error {
	v := validate.New("driver.write_task")
	v.NotNil("sink", c.Sink)
	v.NotNil("db", c.DB)
	v.NotNil("status", c.Status)
	v.NotNil("framer", c.Framer)
	v.NotEmptySlice("channels", c.Channels)
	v.Ternary("task", c.Task.Key == uuid.Nil(), "must have a key")
	return v.Error()
}

// WriteTask streams channels from the Core and sends their frames to a Sink. It
// handles the start and stop commands and reports the health of the sink through the
// task status. Safe for concurrent use.
type WriteTask struct {
	*Runner
	cfg WriteTaskConfig
}

var _ Task = (*WriteTask)(nil)

// NewWriteTask validates the configuration and returns a stopped WriteTask.
func NewWriteTask(cfgs ...WriteTaskConfig) (*WriteTask, error) {
	cfg, err := config.New(WriteTaskConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	t := &WriteTask{cfg: cfg}
	t.Runner, err = NewRunner(RunnerConfig{
		DB:              cfg.DB,
		Status:          cfg.Status,
		Instrumentation: cfg.Instrumentation,
		Task:            cfg.Task,
		Open:            t.open,
	})
	if err != nil {
		return nil, err
	}
	return t, nil
}

// stream is an open streamer of a WriteTask, which runs apart from the run context.
type stream struct {
	requests  confluence.Inlet[framer.StreamerRequest]
	responses confluence.Outlet[framer.StreamerResponse]
	sCtx      signal.Context
	cancel    context.CancelFunc
}

// close stops the streamer and returns its error.
func (s stream) close() error {
	defer s.cancel()
	s.requests.Close()
	confluence.Drain(s.responses)
	if err := s.sCtx.Wait(); !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func (t *WriteTask) open(ctx context.Context) (func(context.Context) error, error) {
	w, err := t.cfg.Sink.Open(ctx)
	if err != nil {
		return nil, err
	}
	s, err := t.openStream(ctx)
	if err != nil {
		return nil, errors.Combine(err, w.Close())
	}
	return func(ctx context.Context) error { return t.run(ctx, w, s) }, nil
}

// openStream returns once the relay streams the channels, so the task misses no
// command written after it reports that it started.
func (t *WriteTask) openStream(ctx context.Context) (stream, error) {
	streamer, err := t.cfg.Framer.NewStreamer(
		ctx, framer.StreamerConfig{Keys: t.cfg.Channels, SendOpenAck: true},
	)
	if err != nil {
		return stream{}, err
	}
	var (
		requests  = confluence.NewStream[framer.StreamerRequest]()
		responses = confluence.NewStream[framer.StreamerResponse](10)
	)
	streamer.InFrom(requests)
	streamer.OutTo(responses)
	sCtx, cancel := signal.Isolated(
		signal.WithInstrumentation(t.cfg.Instrumentation),
	)
	s := stream{requests: requests, responses: responses, sCtx: sCtx, cancel: cancel}
	streamer.Flow(
		sCtx, confluence.CloseOutputInletsOnExit(), confluence.RecoverWithErrOnPanic(),
	)
	select {
	case <-ctx.Done():
		// The streamer ignores its requests until the relay acks, so only a cancel
		// stops it.
		cancel()
		err = ctx.Err()
	case _, ok := <-responses.Outlet():
		if ok {
			return s, nil
		}
		err = errors.New("streamer closed before it opened")
	}
	return stream{}, errors.Combine(err, s.close())
}

func (t *WriteTask) run(ctx context.Context, w Writer, s stream) (err error) {
	defer func() { err = errors.Combine(err, w.Close()) }()
	var (
		health                = make(chan error)
		healthCtx, stopHealth = context.WithCancel(ctx)
	)
	s.sCtx.Go(func(context.Context) error {
		for {
			hErr := w.Health(healthCtx)
			if healthCtx.Err() != nil {
				return nil
			}
			select {
			case <-healthCtx.Done():
				return nil
			case health <- hErr:
			}
		}
	}, signal.RecoverWithErrOnPanic())
	defer func() {
		stopHealth()
		err = errors.Combine(err, s.close())
	}()
	// The sink's health holds until the sink reports a change: a write that succeeds
	// says nothing about it.
	var healthErr error
	for {
		select {
		case <-ctx.Done():
			return nil
		case res, ok := <-s.responses.Outlet():
			if !ok {
				// A stop closes the stream too, and select may take this case first.
				if ctx.Err() != nil {
					return nil
				}
				return errors.New("command stream closed unexpectedly")
			}
			if res.Frame.Empty() {
				continue
			}
			wErr := w.Write(ctx, res.Frame)
			if wErr != nil && !errors.IsAny(wErr, ErrTemporary, ErrDegraded) {
				return wErr
			}
			t.Report(ctx, "Failed to write data", errors.Combine(healthErr, wErr))
		case healthErr = <-health:
			if healthErr != nil && !errors.Is(healthErr, ErrTemporary) {
				return healthErr
			}
			t.Report(ctx, "Failed to write data", healthErr)
		}
	}
}
