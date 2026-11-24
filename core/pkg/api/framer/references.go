// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"context"
	"slices"
	"sync"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
)

// errReferencesOverflowed fails a stream whose reference changes arrive faster than it
// sends them.
var errReferencesOverflowed = errors.New(
	"[framer] - the stream fell behind the reference changes",
)

// referenceStream is the sender of one stream. The relay holds only the target of each
// reference change, so while the stream's keys hold the reference change channel, the
// stream reads the changes from the provider in place of the relay.
type referenceStream struct {
	*recordSender
	source signals.ReferenceChanges
	// out holds the frames of the changes for the goroutine that sends the stream.
	out chan StreamerResponse
	// queue holds the changes of the commits that wait to be sent. full closes when a
	// commit finds queue full, and closed closes when the stream sends nothing more.
	queue     chan []signals.ReferenceChange
	full      chan struct{}
	fullOnce  sync.Once
	closed    chan struct{}
	closeOnce sync.Once
	// mu guards disconnect, which is nil while the stream does not want the channel.
	mu         sync.Mutex
	disconnect observe.Disconnect
}

var _ freighter.StreamSenderCloser[StreamerResponse] = (*referenceStream)(nil)

func (s *Service) newReferenceStream(records *recordSender) *referenceStream {
	return &referenceStream{
		recordSender: records,
		source:       s.references,
		out:          make(chan StreamerResponse),
		queue:        make(chan []signals.ReferenceChange, s.referenceQueue),
		full:         make(chan struct{}),
		closed:       make(chan struct{}),
	}
}

// route returns the keys that the relay serves: keys without the reference change
// channel. While keys hold it, the stream reads its changes from the provider.
func (r *referenceStream) route(keys channel.Keys) channel.Keys {
	wants := r.source.Key != 0 && slices.Contains(keys, r.source.Key)
	r.mu.Lock()
	defer r.mu.Unlock()
	switch {
	case wants && r.disconnect == nil:
		r.disconnect = r.source.OnChange(r.enqueue)
	case !wants && r.disconnect != nil:
		r.disconnect()
		r.disconnect = nil
	}
	if !wants {
		return keys
	}
	return slices.DeleteFunc(slices.Clone(keys), func(k channel.Key) bool {
		return k == r.source.Key
	})
}

// enqueue runs on the committing goroutine, so it never blocks.
func (r *referenceStream) enqueue(
	_ context.Context,
	changes []signals.ReferenceChange,
) {
	select {
	case r.queue <- changes:
	default:
		r.fullOnce.Do(func() { close(r.full) })
	}
}

// CloseSend implements freighter.StreamSenderCloser. It stops run.
func (r *referenceStream) CloseSend() error {
	r.closeOnce.Do(func() { close(r.closed) })
	return nil
}

// run hands the frames of the queued changes to out until the stream closes or ctx
// ends. It fails when the queue overflows.
func (r *referenceStream) run(ctx context.Context) error {
	for {
		// An overflow fails the stream before it sends any later change.
		select {
		case <-r.full:
			return errReferencesOverflowed
		default:
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-r.closed:
			return nil
		case <-r.full:
			return errReferencesOverflowed
		case changes := <-r.queue:
			res, ok, err := r.referenceResponse(r.source.Key, changes)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			select {
			case r.out <- res:
			case <-ctx.Done():
				return ctx.Err()
			case <-r.closed:
				return nil
			}
		}
	}
}
