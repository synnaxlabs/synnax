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
	"time"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/breaker"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	xtime "github.com/synnaxlabs/x/time"
	"github.com/synnaxlabs/x/validate"
)

// ErrTemporary marks an error that a task recovers from by trying again, such as a
// lost connection. A task reports it as a warning and keeps running.
var ErrTemporary = errors.New("temporary error")

// ErrDegraded marks a condition that loses data while the task keeps up its work,
// such as a full queue. A task reports it as a warning and continues at once.
var ErrDegraded = errors.New("degraded")

// Source produces the frames that a ReadTask writes to the Core.
type Source interface {
	// Start acquires what the source needs to produce frames.
	Start(ctx context.Context) error
	// Read blocks until the next frame is ready or ctx is cancelled. An empty frame
	// with a nil error writes nothing and reports the source as healthy. An error
	// that matches ErrTemporary makes the task warn, back off, and call Read again.
	// An error that matches ErrDegraded makes the task warn and write the frame that
	// came with it. Any other error stops the task.
	Read(ctx context.Context) (framer.Frame, error)
	// Stop releases what Start acquired. The task calls it after the last Read
	// returns.
	Stop() error
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
	v.Ternary("task", c.Task.Key == uuid.Nil(), "must have a key")
	v.Exec(c.Breaker.Validate)
	return v.Error()
}

// ReadTask runs a Source and writes its frames to the Core. It handles the start and
// stop commands, opens its writer on the first frame, and reports the health of the
// source through the task status. Safe for concurrent use.
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
		Open:            cfg.Source.Start,
		Run:             t.run,
	})
	return t, err
}

func (t *ReadTask) run(ctx context.Context) (err error) {
	var w *framer.Writer
	defer func() {
		if w != nil {
			err = errors.Combine(err, w.Close())
		}
		err = errors.Combine(err, t.cfg.Source.Stop())
	}()
	brk, err := breaker.NewBreaker(ctx, t.cfg.Breaker)
	if err != nil {
		return err
	}
	for {
		fr, err := t.cfg.Source.Read(ctx)
		if ctx.Err() != nil {
			return nil
		}
		if errors.Is(err, ErrTemporary) {
			t.Report(ctx, err)
			if !brk.Wait() {
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
			continue
		}
		if err != nil && !errors.Is(err, ErrDegraded) {
			return err
		}
		brk.Reset()
		t.Report(ctx, err)
		if fr.Empty() {
			continue
		}
		if w == nil {
			if w, err = t.cfg.Framer.OpenWriter(ctx, framer.WriterConfig{
				ControlSubject: control.Subject{
					Key:  t.cfg.Task.Key.String(),
					Name: t.cfg.Task.Name,
				},
				Keys:              t.cfg.Channels,
				Start:             startOf(fr),
				Mode:              t.cfg.Mode,
				ErrOnUnauthorized: new(true),
				// A push source can wait a long time for its next frame. Without an
				// acknowledgment, a failed write stays hidden until that frame.
				Sync: new(true),
			}); err != nil {
				return err
			}
		}
		if _, err = w.Write(fr); err != nil {
			return err
		}
	}
}

// startOf returns the first timestamp in fr, so that the writer's domain starts at
// the first sample. It returns the current time for a frame with no timestamps.
func startOf(fr framer.Frame) telem.TimeStamp {
	for s := range fr.Series() {
		if s.DataType == telem.TimestampT && s.Len() > 0 {
			return s.ValueAt[telem.TimeStamp](0)
		}
	}
	return telem.Now()
}
