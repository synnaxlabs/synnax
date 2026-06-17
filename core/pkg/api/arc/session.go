// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"
	"sync"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
)

// SessionFrameType tags the kind of payload carried by a SessionFrame.
type SessionFrameType uint8

const (
	// SessionFrameDocUpdate carries an opaque collaborative-document update.
	SessionFrameDocUpdate SessionFrameType = 0
	// SessionFrameAwareness carries ephemeral presence such as a cursor, selection, or
	// editor identity.
	SessionFrameAwareness SessionFrameType = 1
)

// SessionFrame is a single message exchanged over a collaborative editing session. The
// first frame a client sends joins an arc by setting Arc; every subsequent frame is
// relayed verbatim to the other clients editing that arc. The transport never
// interprets Payload.
type SessionFrame struct {
	// Arc is the arc the client is joining. It is read from the first frame of a
	// session and ignored thereafter.
	Arc Key `json:"arc" msgpack:"arc"`
	// Type tags the payload so clients can route it without the server decoding it.
	Type SessionFrameType `json:"type" msgpack:"type"`
	// Payload is opaque application data relayed unchanged to the other clients.
	Payload []byte `json:"payload" msgpack:"payload"`
}

// SessionStream is the server side of the collaborative session stream.
type SessionStream = freighter.ServerStream[SessionFrame, SessionFrame]

const sessionFrameBufferSize = 64

// Session relays opaque frames between the clients editing a single arc. The first frame
// received joins the arc named by its Arc field; the join is rejected unless the caller
// holds update access to that arc. Subsequent frames are fanned out to every other
// client in the same arc's session and never echoed back to the sender. The stream ends
// when the client disconnects, at which point the client is removed from the session.
func (s *Service) Session(ctx context.Context, stream SessionStream) error {
	join, err := stream.Receive()
	if err != nil {
		return err
	}
	if err = s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: arc.OntologyIDs([]arc.Key{join.Arc}),
	}); err != nil {
		return err
	}
	sub := s.sessions.join(join.Arc)
	defer s.sessions.leave(join.Arc, sub)

	sCtx, cancel := signal.WithCancel(ctx, signal.WithInstrumentation(s.Child("arc_session")))
	defer cancel()
	sCtx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case frame := <-sub.frames:
				if err := stream.Send(frame); err != nil {
					return err
				}
			}
		}
	}, signal.WithKey("send"))
	sCtx.Go(func(ctx context.Context) error {
		for {
			frame, err := stream.Receive()
			if err != nil {
				return err
			}
			s.sessions.broadcast(join.Arc, sub, frame)
		}
	}, signal.WithKey("receive"))
	return sCtx.Wait()
}

// subscriber is one client connected to an arc's session.
type subscriber struct {
	// frames buffers frames bound for this client.
	frames chan SessionFrame
	// done is closed when the client leaves, unblocking any in-flight broadcast.
	done chan struct{}
}

// sessionHub fans out frames between the clients editing each arc. It is safe for
// concurrent use.
type sessionHub struct {
	mu   sync.Mutex
	arcs map[Key]set.Set[*subscriber]
}

func newSessionHub() *sessionHub {
	return &sessionHub{arcs: make(map[Key]set.Set[*subscriber])}
}

// join registers a new subscriber for the given arc and returns it.
func (h *sessionHub) join(key Key) *subscriber {
	sub := &subscriber{
		frames: make(chan SessionFrame, sessionFrameBufferSize),
		done:   make(chan struct{}),
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	subs, ok := h.arcs[key]
	if !ok {
		subs = set.New[*subscriber]()
		h.arcs[key] = subs
	}
	subs.Add(sub)
	return sub
}

// leave removes sub from the given arc and releases any broadcast blocked on it.
func (h *sessionHub) leave(key Key, sub *subscriber) {
	h.mu.Lock()
	if subs, ok := h.arcs[key]; ok {
		subs.Remove(sub)
		if len(subs) == 0 {
			delete(h.arcs, key)
		}
	}
	h.mu.Unlock()
	close(sub.done)
}

// broadcast delivers frame to every subscriber of the given arc except from. Delivery
// blocks on a slow client until it drains or leaves, applying backpressure only to the
// sender.
func (h *sessionHub) broadcast(key Key, from *subscriber, frame SessionFrame) {
	h.mu.Lock()
	targets := make([]*subscriber, 0, len(h.arcs[key]))
	for sub := range h.arcs[key] {
		if sub != from {
			targets = append(targets, sub)
		}
	}
	h.mu.Unlock()
	for _, sub := range targets {
		select {
		case sub.frames <- frame:
		case <-sub.done:
		}
	}
}
