// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"
	"sync"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// dropStorage drops the storage of keys, then releases their delete claim. A failed
// drop raises an error status.
func (s *Service) dropStorage(ctx context.Context, keys Keys, release func()) {
	defer release()
	if err := s.cfg.TS.Drop(keys.Uint32()); err != nil {
		s.cfg.L.Error("failed to drop the storage of deleted channels", zap.Error(err))
		stat := status.Status[any]{
			Variant:     status.VariantError,
			Message:     "Failed to remove the data of deleted channels",
			Description: err.Error(),
			Time:        telem.Now(),
		}
		if err = s.db.WithTx(ctx, func(tx gorp.Tx) error {
			return s.cfg.Status.NewWriter(tx).Set(ctx, &stat)
		}); err != nil {
			s.cfg.L.Error("failed to set the status of a failed drop", zap.Error(err))
		}
	}
}

// dropQueue runs queued work in order on one goroutine, so that a commit that queues a
// drop never waits on storage.
type dropQueue struct {
	wake    chan struct{}
	done    chan struct{}
	pending []func()
	mu      sync.Mutex
	closed  bool
}

func openDropQueue() *dropQueue {
	q := &dropQueue{wake: make(chan struct{}, 1), done: make(chan struct{})}
	go q.run()
	return q
}

// push queues work. After Close, push runs work in line.
func (q *dropQueue) push(work func()) {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		work()
		return
	}
	q.pending = append(q.pending, work)
	select {
	case q.wake <- struct{}{}:
	default:
	}
	q.mu.Unlock()
}

func (q *dropQueue) run() {
	defer close(q.done)
	for range q.wake {
		for {
			q.mu.Lock()
			if len(q.pending) == 0 {
				q.mu.Unlock()
				break
			}
			work := q.pending[0]
			q.pending = q.pending[1:]
			q.mu.Unlock()
			work()
		}
	}
}

// Close runs the queued work, then stops the goroutine of the queue.
func (q *dropQueue) Close() error {
	q.mu.Lock()
	q.closed = true
	close(q.wake)
	q.mu.Unlock()
	<-q.done
	return nil
}
