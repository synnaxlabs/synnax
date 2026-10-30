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
	"slices"
	"time"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/breaker"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	xtime "github.com/synnaxlabs/x/time"
	"github.com/synnaxlabs/x/validate"
)

// Source opens the reads of a ReadTask.
type Source interface {
	// Open acquires what the reads of one run need and returns the Reader that owns it.
	// Its error answers the start command.
	Open(ctx context.Context) (Reader, error)
}

// Reader produces the frames of one run of a ReadTask.
type Reader interface {
	// Read blocks until the next frame is ready or ctx is cancelled. An empty frame
	// with a nil error writes nothing and reports the source as healthy. An error
	// that matches ErrTemporary makes the task warn, back off, and call Read again.
	// An error that matches ErrDegraded makes the task warn and write the frame that
	// came with it. Any other error stops the task.
	Read(ctx context.Context) (framer.Frame, error)
	// Close releases what Open acquired. The task calls it after the last Read returns.
	Close() error
}

// ReadTaskConfig is the configuration for a ReadTask.
type ReadTaskConfig struct {
	// Source produces the frames to write.
	//
	// [REQUIRED]
	Source Source
	// DB opens the transactions that status writes run in.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Status writes the statuses of the task.
	//
	// [REQUIRED]
	Status *status.Service
	// Framer opens the writer of the task.
	//
	// [REQUIRED]
	Framer *framer.Service
	alamos.Instrumentation
	// Channels are the keys of every channel the source writes to.
	//
	// [REQUIRED]
	Channels channel.Keys
	// Indexes are the index channels among Channels. The writer starts at the earliest
	// timestamp that the first frame writes to one of them.
	//
	// [OPTIONAL] - Without them, the writer starts at the current time.
	Indexes channel.Keys
	// Task is the stored task this instance runs.
	//
	// [REQUIRED]
	Task task.Task
	// Breaker sets the backoff between reads that fail with ErrTemporary.
	//
	// [OPTIONAL] - Defaults to retrying forever, from 1s up to 30s between reads.
	Breaker breaker.Config
	// Mode selects whether the writer persists, streams, or does both.
	//
	// [OPTIONAL] - Defaults to persisting and streaming.
	Mode framer.WriterMode
}

var (
	_ config.Config[ReadTaskConfig] = ReadTaskConfig{}
	// DefaultReadTaskConfig is the default configuration for a ReadTask.
	DefaultReadTaskConfig = ReadTaskConfig{
		Breaker: breaker.Config{
			BaseInterval: time.Second,
			Scale:        2,
			MaxInterval:  30 * time.Second,
			MaxRetries:   breaker.InfiniteRetries,
			Clock:        xtime.Real,
		},
	}
)

// Override implements config.Config.
func (c ReadTaskConfig) Override(other ReadTaskConfig) ReadTaskConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Source = override.Nil(c.Source, other.Source)
	c.DB = override.Nil(c.DB, other.DB)
	c.Status = override.Nil(c.Status, other.Status)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Channels = override.Slice(c.Channels, other.Channels)
	c.Indexes = override.Slice(c.Indexes, other.Indexes)
	if other.Task.Key != uuid.Nil() {
		c.Task = other.Task
	}
	c.Breaker = c.Breaker.Override(other.Breaker)
	c.Mode = override.Numeric(c.Mode, other.Mode)
	return c
}

// Validate implements config.Config.
func (c ReadTaskConfig) Validate() error {
	v := validate.New("driver.read_task")
	v.NotNil("source", c.Source)
	v.NotNil("db", c.DB)
	v.NotNil("status", c.Status)
	v.NotNil("framer", c.Framer)
	v.NotEmptySlice("channels", c.Channels)
	for _, key := range c.Indexes {
		v.Ternaryf(
			"indexes", !slices.Contains(c.Channels, key), "%d is not in channels", key,
		)
	}
	v.Ternary("task", c.Task.Key == uuid.Nil(), "must have a key")
	v.Exec(c.Breaker.Validate)
	return v.Error()
}

// ReadTask runs a Source and writes its frames to the Core. It handles the start and
// stop commands, opens its writer on the first frame, and reports the health of the
// source through the task status. A failed write stops the task, also while the source
// waits for its next frame. Safe for concurrent use.
type ReadTask struct {
	*Runner
	cfg ReadTaskConfig
}

var _ Task = (*ReadTask)(nil)

// NewReadTask validates the configuration and returns a stopped ReadTask.
func NewReadTask(cfgs ...ReadTaskConfig) (*ReadTask, error) {
	cfg, err := config.New(DefaultReadTaskConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	t := &ReadTask{cfg: cfg}
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

func (t *ReadTask) open(ctx context.Context) (func(context.Context) error, error) {
	r, err := t.cfg.Source.Open(ctx)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context) error { return t.run(ctx, r) }, nil
}

// errWriterClosed is the error of a run whose writer stops before the run does.
var errWriterClosed = errors.New("writer closed unexpectedly")

func (t *ReadTask) run(ctx context.Context, r Reader) (err error) {
	// A failed write cancels readCtx, so that a source that waits for its next frame
	// does not hide the failure.
	readCtx, fail := context.WithCancelCause(ctx)
	var w *frameWriter
	defer func() {
		fail(nil)
		if w != nil {
			err = errors.Combine(w.close(), err)
		}
		err = errors.Combine(err, r.Close())
	}()
	// stopped returns the error of a run whose reads are cancelled: nil for a stop.
	stopped := func() error {
		if ctx.Err() != nil {
			return nil
		}
		return context.Cause(readCtx)
	}
	brk, err := breaker.NewBreaker(readCtx, t.cfg.Breaker)
	if err != nil {
		return err
	}
	for {
		fr, err := r.Read(readCtx)
		if fr.Empty() && readCtx.Err() != nil {
			return stopped()
		}
		if errors.Is(err, ErrTemporary) {
			t.Report(ctx, "Failed to read data", err)
			if !brk.Wait() {
				if readCtx.Err() != nil {
					return stopped()
				}
				return err
			}
			continue
		}
		if err != nil && !errors.Is(err, ErrDegraded) {
			return err
		}
		brk.Reset()
		t.Report(ctx, "Failed to read some data", err)
		if fr.Empty() {
			continue
		}
		if w == nil {
			start := startOf(fr, t.cfg.Indexes)
			if w, err = t.openWriter(ctx, start, fail); err != nil {
				return err
			}
		}
		if !w.write(fr) {
			return stopped()
		}
	}
}

// frameWriter streams the frames of one run to the Core.
type frameWriter struct {
	requests confluence.Inlet[framer.WriterRequest]
	sCtx     signal.Context
	cancel   context.CancelFunc
}

// openWriter opens a writer that calls fail with its first error.
func (t *ReadTask) openWriter(
	ctx context.Context,
	start telem.TimeStamp,
	fail context.CancelCauseFunc,
) (*frameWriter, error) {
	// A stop cancels ctx, but the writer must still write its last frames and commit.
	ctx = context.WithoutCancel(ctx)
	sw, err := t.cfg.Framer.NewStreamWriter(ctx, framer.WriterConfig{
		ControlSubject: control.Subject{
			Key:  t.cfg.Task.Key.String(),
			Name: t.cfg.Task.Name,
		},
		Keys:              t.cfg.Channels,
		Start:             start,
		Mode:              t.cfg.Mode,
		ErrOnUnauthorized: new(true),
	})
	if err != nil {
		return nil, err
	}
	var (
		requests  = confluence.NewStream[framer.WriterRequest]()
		responses = confluence.NewStream[framer.WriterResponse]()
	)
	sw.InFrom(requests)
	sw.OutTo(responses)
	// A routine never starts under a cancelled context, and then never closes its
	// outlets. The writer runs apart from ctx, so that close always ends it.
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(t.cfg.Instrumentation))
	sw.Flow(
		sCtx, confluence.CloseOutputInletsOnExit(), confluence.RecoverWithErrOnPanic(),
	)
	sCtx.Go(func(context.Context) error {
		for res := range responses.Outlet() {
			if res.Err != nil {
				fail(res.Err)
			}
		}
		fail(errWriterClosed)
		return nil
	}, signal.RecoverWithErrOnPanic())
	return &frameWriter{requests: requests, sCtx: sCtx, cancel: cancel}, nil
}

// write sends fr to the writer, also during a stop, so that a stop drops no frame that
// the source returned. It returns false when the writer has stopped.
func (w *frameWriter) write(fr framer.Frame) bool {
	select {
	case w.requests.Inlet() <- framer.WriterRequest{
		Command: framer.WriterCommandWrite,
		Frame:   fr,
	}:
		return true
	case <-w.sCtx.Stopped():
		return false
	}
}

// close commits the written frames and returns the first error of the writer.
func (w *frameWriter) close() error {
	defer w.cancel()
	w.requests.Close()
	return w.sCtx.Wait()
}

// startOf returns the earliest timestamp that fr writes to one of indexes, so that the
// writer's domain holds every sample. It returns the current time for a frame that
// writes to no index.
func startOf(fr framer.Frame, indexes channel.Keys) telem.TimeStamp {
	start := telem.TimeStampMax
	for key, s := range fr.Entries() {
		if s.Len() > 0 && slices.Contains(indexes, key) {
			start = min(start, s.ValueAt[telem.TimeStamp](0))
		}
	}
	if start == telem.TimeStampMax {
		return telem.Now()
	}
	return start
}
