// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer

import (
	"context"
	"slices"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
)

// raise is the authority that the first write of a clock raises it to.
type raise struct {
	index     channel.Channel
	authority control.Authority
}

// raisesOf returns the raise of each clock of cfg that cfg.Keys does not name, to the
// authority of the first key of cfg on it.
func raisesOf(
	cfg Config,
	channels map[channel.Key]channel.Channel,
) map[channel.Key]raise {
	var raises map[channel.Key]raise
	for _, clock := range cfg.Clocks {
		if cfg.Keys.Contains(clock) {
			continue
		}
		i := slices.IndexFunc(cfg.Keys, func(k channel.Key) bool {
			return channels[k].Index() == clock
		})
		if i < 0 {
			continue
		}
		if raises == nil {
			raises = make(map[channel.Key]raise)
		}
		raises[clock] = raise{index: channels[clock], authority: authorityOf(cfg, i)}
	}
	return raises
}

// withClocks returns cfg with the clock of each raise added to its keys at authority 0,
// so the writer claims nothing on it against another writer.
func withClocks(cfg Config, raises map[channel.Key]raise) Config {
	if len(raises) == 0 {
		return cfg
	}
	authorities := make([]control.Authority, len(cfg.Keys), len(cfg.Keys)+len(raises))
	for i := range cfg.Keys {
		authorities[i] = authorityOf(cfg, i)
	}
	keys := slices.Clone(cfg.Keys)
	for _, clock := range cfg.Clocks {
		if _, ok := raises[clock]; ok {
			keys = append(keys, clock)
			authorities = append(authorities, 0)
		}
	}
	cfg.Keys, cfg.Authorities = keys, authorities
	return cfg
}

// authorityOf returns the authority that cfg gives the key at position i.
func authorityOf(cfg Config, i int) control.Authority {
	switch len(cfg.Authorities) {
	case 0:
		return control.AuthorityAbsolute
	case 1:
		return cfg.Authorities[0]
	default:
		return cfg.Authorities[i]
	}
}

// wrappedBufferSize is the buffer of the streams between a raiser and the writer it
// wraps.
const wrappedBufferSize = 30

// errWriterClosed reports that the wrapped writer closed while a raise waited.
var errWriterClosed = errors.New("the writer closed during a raise")

// raiser wraps a writer whose clocks wait to rise. At the first write that holds a
// clock, it raises the clock and waits in line for the answer, passing on earlier
// responses. A raise that fails answers the write with its error, and the clock waits
// for the next write that holds it.
type raiser struct {
	confluence.AbstractLinear[Request, Response]
	wrapped StreamWriter
	pending map[channel.Key]raise
	// sent counts the requests sent to wrapped, which numbers its responses the same
	// way.
	sent      int
	requests  confluence.Inlet[Request]
	responses confluence.Outlet[Response]
}

func newRaiser(wrapped StreamWriter, raises map[channel.Key]raise) *raiser {
	return &raiser{wrapped: wrapped, pending: raises}
}

// Flow implements confluence.Flow.
func (r *raiser) Flow(ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(r.Out)
	r.requests, r.responses = confluence.Attach(r.wrapped, wrappedBufferSize)
	r.wrapped.Flow(ctx, confluence.CloseOutputInletsOnExit())
	ctx.Go(func(ctx context.Context) error {
		in := r.In.Outlet()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case req, ok := <-in:
				if !ok {
					r.requests.Close()
					in = nil
					continue
				}
				if err := r.forward(ctx, req); err != nil {
					if errors.Is(err, errWriterClosed) {
						return nil
					}
					return err
				}
			case res, ok := <-r.responses.Outlet():
				if !ok {
					return nil
				}
				if err := signal.SendUnderContext(ctx, r.Out.Inlet(), res); err != nil {
					return err
				}
			}
		}
	}, o.Signal...)
}

// forward raises each clock that req is the first write to hold, then sends req on.
func (r *raiser) forward(ctx context.Context, req Request) error {
	if req.Command == CommandWrite {
		for key := range req.Frame.Keys() {
			pending, ok := r.pending[key]
			if !ok {
				continue
			}
			res, err := r.raise(ctx, key, pending)
			if err != nil {
				return err
			}
			if res.Err == nil && !res.Authorized {
				res.Err = errors.Wrapf(
					control.ErrUnauthorized,
					"the writer has no control over the index %s, so the write cannot "+
						"hold its time",
					pending.index,
				)
			}
			if res.Err != nil {
				return signal.SendUnderContext(
					ctx,
					r.Out.Inlet(),
					Response{Command: CommandWrite, Err: res.Err},
				)
			}
			delete(r.pending, key)
		}
	}
	return r.send(ctx, req)
}

// raise sends the raise of clock and returns its answer, passing on the responses
// before it.
func (r *raiser) raise(
	ctx context.Context,
	clock channel.Key,
	pending raise,
) (Response, error) {
	if err := r.send(ctx, Request{
		Command: CommandSetAuthority,
		Config: Config{
			Keys:        channel.Keys{clock},
			Authorities: []control.Authority{pending.authority},
		},
	}); err != nil {
		return Response{}, err
	}
	seqNum := r.sent
	for {
		select {
		case <-ctx.Done():
			return Response{}, ctx.Err()
		case res, ok := <-r.responses.Outlet():
			if !ok {
				return Response{}, errWriterClosed
			}
			if res.SeqNum == seqNum {
				return res, nil
			}
			if err := signal.SendUnderContext(ctx, r.Out.Inlet(), res); err != nil {
				return Response{}, err
			}
		}
	}
}

func (r *raiser) send(ctx context.Context, req Request) error {
	r.sent++
	return signal.SendUnderContext(ctx, r.requests.Inlet(), req)
}
