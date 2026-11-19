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

	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// session runs the Cesium writer of a writer's stored channels behind the writer's
// request and response streams. The first error ends the work of the session: every
// later request gets it. A refusal for no control is not an error. A refused write
// makes the next commit unauthorized, and a refused commit or end of domain is
// unauthorized.
type session struct {
	confluence.UnarySink[Request]
	confluence.AbstractUnarySource[Response]
	w       *cesium.Writer
	stamper *unindexedStamper[cesium.ChannelKey]
	// err is the first error that the session raised itself. The Cesium writer keeps
	// its own first error and returns it from every later call.
	err     error
	errSent bool
	// writeRefused is true when a write was refused for no control since the last
	// commit.
	writeRefused bool
	sync         bool
}

func newSession(
	w *cesium.Writer,
	virtual []cesium.ChannelKey,
	sync bool,
) *session {
	return &session{
		w:       w,
		stamper: newUnindexedStamper(virtual),
		sync:    sync,
	}
}

// Flow implements confluence.Flow.
func (s *session) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(s.Out)
	sCtx.Go(func(ctx context.Context) error { return s.close(s.run(ctx)) }, o.Signal...)
}

// run serves requests until the inlet closes or ctx ends.
func (s *session) run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case req, ok := <-s.In.Outlet():
			if !ok {
				return nil
			}
			end, authorized, err := s.exec(ctx, req)
			if err = s.respond(ctx, req, end, authorized, err); err != nil {
				return err
			}
		}
	}
}

// close releases the Cesium writer. It returns the first error of the session or of
// the Cesium writer, joined with any error of the release, and otherwise cause.
func (s *session) close(cause error) error {
	if err := errors.Combine(s.err, s.w.Close()); err != nil {
		return err
	}
	return cause
}

func (s *session) exec(
	ctx context.Context,
	req Request,
) (end telem.TimeStamp, authorized bool, err error) {
	if s.err != nil {
		return 0, false, s.err
	}
	switch req.Command {
	case CommandWrite:
		fr := req.Frame.ToStorage()
		s.stamper.stamp(fr)
		if _, err = s.w.Write(ctx, fr); errors.Is(err, control.ErrUnauthorized) {
			s.writeRefused = true
		}
	case CommandCommit:
		end, err = s.w.Commit(ctx)
		refused := s.writeRefused
		s.writeRefused = false
		if err == nil {
			return end, !refused, nil
		}
	case CommandSetAuthority:
		err = s.w.SetAuthority(req.Config.toStorage())
	case CommandEndDomain:
		err = s.w.EndDomain(ctx, req.Config.Keys.Uint32()...)
	case CommandOpen:
		fallthrough
	default:
		s.err = errors.Wrapf(
			validate.ErrValidation,
			"invalid writer command %v",
			req.Command,
		)
		return 0, false, s.err
	}
	if errors.Is(err, control.ErrUnauthorized) {
		return 0, false, nil
	}
	return 0, err == nil, err
}

// respond sends the response to req. A write gets a response only when the writer is
// sync, or with its first error.
func (s *session) respond(
	ctx context.Context,
	req Request,
	end telem.TimeStamp,
	authorized bool,
	err error,
) error {
	if req.Command == CommandWrite && !s.sync && (err == nil || s.errSent) {
		return nil
	}
	if err != nil {
		s.errSent = true
	}
	return signal.SendUnderContext(ctx, s.Out.Inlet(), Response{
		Command:    req.Command,
		SeqNum:     req.SeqNum,
		End:        end,
		Authorized: authorized,
		Err:        err,
	})
}
