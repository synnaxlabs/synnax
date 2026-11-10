// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task

import (
	"context"
	"time"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

const (
	stopCommandType = "stop"
	// stopPollInterval paces the status reads of a stop. The Core drops status
	// notifications under load, so a read is the only reliable ack.
	stopPollInterval = 20 * time.Millisecond
)

// StopperConfig is the configuration for creating a Stopper.
type StopperConfig struct {
	// DB runs the status reads of a stop.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Task provides the channel that carries the stop commands.
	//
	// [REQUIRED]
	Task *Service
	// Status reports which tasks are running.
	//
	// [REQUIRED]
	Status *status.Service
	// Framer writes the stop commands.
	//
	// [REQUIRED]
	Framer *framer.Service
	// Timeout caps how long one Stop call takes. Keep it under the 5 second default
	// read timeout of the Python client.
	//
	// [OPTIONAL] - Defaults to 4 seconds.
	Timeout time.Duration
}

var (
	_ xconfig.Config[StopperConfig] = StopperConfig{}
	// DefaultStopperConfig is the default configuration for a Stopper.
	DefaultStopperConfig = StopperConfig{Timeout: 4 * time.Second}
)

// Override implements xconfig.Config.
func (c StopperConfig) Override(other StopperConfig) StopperConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Task = override.Nil(c.Task, other.Task)
	c.Status = override.Nil(c.Status, other.Status)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Timeout = override.Numeric(c.Timeout, other.Timeout)
	return c
}

// Validate implements xconfig.Config.
func (c StopperConfig) Validate() error {
	v := validate.New("task.stopper")
	v.NotNil("db", c.DB)
	v.NotNil("task", c.Task)
	v.NotNil("status", c.Status)
	v.NotNil("framer", c.Framer)
	v.Positive("timeout", c.Timeout)
	return v.Error()
}

// Stopper stops running tasks ahead of a delete, so that the channels of a task are
// free once the delete returns.
type Stopper struct{ cfg StopperConfig }

// NewStopper creates a Stopper from the given configurations.
func NewStopper(cfgs ...StopperConfig) (*Stopper, error) {
	cfg, err := xconfig.New(DefaultStopperConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Stopper{cfg: cfg}, nil
}

// Stop stops the tasks with the given keys that report running, and waits for each to
// report that it stopped. The caller must authorize keys. The Core marks the tasks of a
// silent rack as not running, so an offline Driver costs no wait. A timeout is not an
// error: the caller deletes the tasks anyway. Stop returns the error of a canceled ctx.
func (s *Stopper) Stop(ctx context.Context, keys []Key) error {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
	defer cancel()
	return errors.Skip(s.stopAndWait(ctx, keys), context.DeadlineExceeded)
}

func (s *Stopper) stopAndWait(ctx context.Context, keys []Key) error {
	running, err := s.retrieveRunning(ctx, keys)
	if err != nil || len(running) == 0 {
		return err
	}
	if err = s.sendStops(ctx, running); err != nil {
		return err
	}
	ticker := time.NewTicker(stopPollInterval)
	defer ticker.Stop()
	for len(running) > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		if running, err = s.retrieveRunning(ctx, running); err != nil {
			return err
		}
	}
	return nil
}

func (s *Stopper) retrieveRunning(
	ctx context.Context,
	keys []Key,
) ([]Key, error) {
	// Status details are caller-provided, so a stop target must come from owners.
	byOwner := lo.SliceToMap(keys, func(k Key) (resource.ID, Key) {
		return ResourceID(k), k
	})
	var statuses []Status
	if err := s.cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return s.cfg.Status.NewRetrieve[StatusDetails]().
			Where(status.MatchOwners[StatusDetails](lo.Keys(byOwner)...)).
			Entries(&statuses).
			Exec(ctx, snap)
	}); errors.Skip(err, query.ErrNotFound) != nil {
		return nil, err
	}
	running := make([]Key, 0, len(statuses))
	for _, stat := range statuses {
		if stat.Details.Running && stat.Owner != nil {
			running = append(running, byOwner[*stat.Owner])
		}
	}
	return running, nil
}

func (s *Stopper) sendStops(ctx context.Context, keys []Key) error {
	series, err := telem.NewJSONSeriesV(lo.Map(
		keys,
		func(k Key, _ int) Command {
			return Command{
				Task: k,
				Type: stopCommandType,
				Key:  uuid.New().String(),
				Args: msgpack.EncodedJSON{},
			}
		},
	)...)
	if err != nil {
		return err
	}
	sw, err := s.cfg.Framer.NewStreamWriter(ctx, framer.WriterConfig{
		Keys:  channel.Keys{s.cfg.Task.CommandChannelKey()},
		Start: telem.Now(),
	})
	if err != nil {
		return err
	}
	var (
		requests  = confluence.NewStream[framer.WriterRequest]()
		responses = confluence.NewStream[framer.WriterResponse]()
	)
	sw.InFrom(requests)
	sw.OutTo(responses)
	// The writer runs apart from ctx, and every wait on it selects on ctx.
	sCtx, cancel := signal.Isolated()
	defer cancel()
	sw.Flow(
		sCtx, confluence.CloseOutputInletsOnExit(), confluence.RecoverWithErrOnPanic(),
	)
	sCtx.Go(func(context.Context) error {
		for res := range responses.Outlet() {
			if res.Err != nil {
				return res.Err
			}
		}
		return nil
	}, signal.RecoverWithErrOnPanic())
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-sCtx.Stopped():
		return sCtx.Wait()
	case requests.Inlet() <- framer.WriterRequest{
		Command: framer.WriterCommandWrite,
		Frame:   frame.NewUnary(s.cfg.Task.CommandChannelKey(), series),
	}:
	}
	requests.Close()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-sCtx.Stopped():
		return sCtx.Wait()
	}
}
