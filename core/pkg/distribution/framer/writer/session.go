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
	"time"

	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	xtime "github.com/synnaxlabs/x/time"
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
	w *cesium.Writer
	// gates gate the writes of the session on its control of each channel.
	gates    *gates
	controls *controls
	// release releases the hold of the session on its channels.
	release func()
	stamper *unindexedStamper[cesium.ChannelKey]
	// autoIndex is nil when the writer does not stamp its indexes.
	autoIndex *indexStamper
	// autoCommit is true when the session commits after each write.
	autoCommit bool
	// persistInterval is the least time between two syncs of auto commits. The session
	// syncs every commit when it does not auto commit, and every auto commit when
	// persistInterval is not positive.
	persistInterval telem.TimeSpan
	clock           xtime.Clock
	lastSync        time.Time
	// writes is where the session sends each written frame when mode streams.
	writes confluence.Inlet[relay.Response]
	mode   Mode
	group  uint32
	// err is the first error that the session raised itself. The Cesium writer keeps
	// its own first error and returns it from every later call.
	err     error
	errSent bool
	// writeRefused is true when a write was refused for no control since the last
	// commit.
	writeRefused bool
	sync         bool
}

func (s *Service) newSession(
	w *cesium.Writer,
	cfg Config,
	virtual []cesium.ChannelKey,
	autoIndex *indexStamper,
	release func(),
	g *gates,
) *session {
	return &session{
		w:               w,
		gates:           g,
		controls:        s.controls,
		release:         release,
		stamper:         newUnindexedStamper(virtual),
		autoIndex:       autoIndex,
		writes:          s.cfg.Writes,
		autoCommit:      *cfg.EnableAutoCommit,
		persistInterval: cfg.AutoIndexPersistInterval,
		clock:           s.cfg.Clock,
		lastSync:        s.cfg.Clock.Now(),
		mode:            cfg.Mode,
		group:           cfg.ControlSubject.Group,
		sync:            *cfg.Sync,
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

// close closes the Cesium writer and releases the hold of the session on its channels
// and on their control. It returns the first error of the session or of the close, and
// otherwise cause.
func (s *session) close(cause error) error {
	err := errors.Combine(s.err, s.w.Close())
	s.controls.notify(s.gates.release())
	s.release()
	if err != nil {
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
		authorized, err = s.write(ctx, req.Frame)
		return 0, authorized && err == nil, err
	case CommandCommit:
		var refused bool
		end, refused, err = s.commit(ctx)
		writeRefused := s.writeRefused
		s.writeRefused = false
		if err != nil || refused {
			return 0, false, err
		}
		return end, !writeRefused, nil
	case CommandSetAuthority:
		var transfers []transfer
		if transfers, err = s.gates.setAuthority(req.Config); err != nil {
			s.err = err
			return 0, false, err
		}
		s.controls.notify(transfers)
	case CommandEndDomain:
		err = s.endDomain(ctx, req.Config.Keys)
		if err == nil || errors.Is(err, control.ErrUnauthorized) {
			if syncErr := s.maybeSync(); syncErr != nil {
				err = syncErr
			}
		}
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

// write writes fr except the channels the session does not control, commits it when the
// session auto commits, and relays the frame that the Cesium writer returns. A refusal
// for no control is not an error: write returns authorized false.
func (s *session) write(
	ctx context.Context,
	fr frame.Frame,
) (authorized bool, err error) {
	st := fr.ToStorage()
	s.stamper.stamp(st)
	if s.autoIndex != nil {
		st = s.autoIndex.stamp(st)
	}
	written, refused, err := s.w.WriteExcept(ctx, st, s.gates.refused())
	if err != nil {
		return false, s.fail(err)
	}
	if s.autoCommit {
		_, commitRefused, err := s.commit(ctx)
		if err != nil {
			return false, err
		}
		refused = refused || commitRefused
	}
	s.writeRefused = s.writeRefused || refused
	return !refused, s.relay(ctx, written)
}

// commit commits the channels the session controls, then syncs the Cesium writer when
// the persist rule asks, also after a commit refused for no control. It returns true
// when the session lacked control of a channel with samples to commit.
func (s *session) commit(ctx context.Context) (telem.TimeStamp, bool, error) {
	end, refused, err := s.w.CommitExcept(ctx, s.gates.refused())
	if err != nil {
		return 0, false, s.fail(err)
	}
	if syncErr := s.maybeSync(); syncErr != nil {
		return 0, false, syncErr
	}
	return end, refused, nil
}

// endDomain commits and ends the domains of the indexes the session may end, and
// commits the channels it controls of the others. It returns an error that wraps
// control.ErrUnauthorized for an index it may not end.
func (s *session) endDomain(ctx context.Context, indexes channel.Keys) error {
	ended, skip, refusal := s.gates.endDomain(indexes)
	if skip != nil {
		if _, _, err := s.w.CommitExcept(ctx, skip); err != nil {
			return s.fail(err)
		}
	}
	if err := s.w.EndDomain(ctx, ended...); err != nil {
		return s.fail(err)
	}
	return refusal
}

// fail names the holder in a claim refusal of err and makes the named refusal the error
// of the session.
func (s *session) fail(err error) error {
	named, ok := s.controls.nameHolder(err)
	if ok && s.err == nil {
		s.err = named
	}
	return named
}

// maybeSync syncs the Cesium writer on every commit when the session does not auto
// commit, and otherwise once persistInterval has passed since the last sync.
func (s *session) maybeSync() error {
	if s.autoCommit && s.persistInterval > 0 {
		now := s.clock.Now()
		if now.Sub(s.lastSync) < s.persistInterval.Duration() {
			return nil
		}
		s.lastSync = now
	}
	return s.w.Sync()
}

// relay sends fr to the relay when the writer streams.
func (s *session) relay(ctx context.Context, fr cesium.Frame) error {
	if !s.mode.Stream() {
		return nil
	}
	return signal.SendUnderContext(ctx, s.writes.Inlet(), relay.Response{
		Frame: frame.NewFromStorage(fr),
		Group: s.group,
	})
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
