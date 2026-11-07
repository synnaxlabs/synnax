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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

const (
	stopCommandType = "stop"
	// stopPollInterval paces the status reads of a stop. The Core drops status
	// notifications under load, so a read is the only reliable ack.
	stopPollInterval = 20 * time.Millisecond
)

// stopRunning stops the tasks that report running and waits for each to report that it
// stopped. The Core marks the tasks of a silent rack as not running, so an offline
// Driver costs no wait. A timeout is not an error: the caller deletes the tasks anyway.
func (s *Service) stopRunning(ctx context.Context, keys []task.Key) error {
	ctx, cancel := context.WithTimeout(ctx, s.stopTimeout)
	defer cancel()
	return errors.Skip(s.stopAndWait(ctx, keys), context.DeadlineExceeded)
}

func (s *Service) stopAndWait(ctx context.Context, keys []task.Key) error {
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

func (s *Service) retrieveRunning(
	ctx context.Context,
	keys []task.Key,
) ([]task.Key, error) {
	// Status details are caller-provided, so a stop target must come from owners.
	byOwner := lo.SliceToMap(keys, func(k task.Key) (resource.ID, task.Key) {
		return task.ResourceID(k), k
	})
	var statuses []task.Status
	if err := s.status.NewRetrieve[task.StatusDetails]().
		Where(status.MatchOwners[task.StatusDetails](lo.Keys(byOwner)...)).
		Entries(&statuses).
		Exec(ctx, nil); errors.Skip(err, query.ErrNotFound) != nil {
		return nil, err
	}
	running := make([]task.Key, 0, len(statuses))
	for _, stat := range statuses {
		if stat.Details.Running && stat.Owner != nil {
			running = append(running, byOwner[*stat.Owner])
		}
	}
	return running, nil
}

func (s *Service) sendStops(ctx context.Context, keys []task.Key) error {
	series, err := telem.NewJSONSeriesV(lo.Map(
		keys,
		func(k task.Key, _ int) task.Command {
			return task.Command{
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
	sw, err := s.framer.NewStreamWriter(ctx, framer.WriterConfig{
		Keys:  channel.Keys{s.task.CommandChannelKey()},
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
	// A routine never starts under a canceled context, and then never closes its
	// outlets. The writer runs apart from ctx, and every wait on it selects on ctx.
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
		Frame:   frame.NewUnary(s.task.CommandChannelKey(), series),
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
