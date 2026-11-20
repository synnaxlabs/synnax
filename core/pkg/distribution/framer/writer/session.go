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

// session runs one Cesium writer for each clock of a writer's stored channels behind
// the
// writer's request and response streams. The first error ends the work of the session:
// every later request gets it. A refusal for no control is not an error. A refused
// write
// makes the next commit unauthorized, and a refused commit or end of domain is
// unauthorized.
type session struct {
	confluence.UnarySink[Request]
	confluence.AbstractUnarySource[Response]
	ws []*cesium.Writer
	// gates gate the writes of the session on its control of each channel.
	gates    *gates
	controls *controls
	// lacks reports whether the session lacks control of a channel. Cesium calls it
	// under the channel's claim lock.
	lacks func(cesium.ChannelKey) bool
	// release releases the hold of the session on its channels.
	release func()
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
	// err is the first error of the session.
	err     error
	errSent bool
	// writeRefused is true when a write was refused for no control since the last
	// commit.
	writeRefused bool
	sync         bool
}

func (s *Service) newSession(
	ws []*cesium.Writer,
	cfg Config,
	autoIndex *indexStamper,
	release func(),
	g *gates,
) *session {
	return &session{
		ws:              ws,
		gates:           g,
		controls:        s.controls,
		lacks:           g.lacks,
		release:         release,
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

// close closes the Cesium writers and releases the hold of the session on its channels
// and on their control. It returns the first error of the session or of the closes, and
// otherwise cause.
func (s *session) close(cause error) error {
	err := s.err
	for _, w := range s.ws {
		err = errors.Combine(err, w.Close())
	}
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
// session auto commits, and relays the frame that the Cesium writers return. A refusal
// for no control is not an error: write returns authorized false.
func (s *session) write(
	ctx context.Context,
	fr frame.Frame,
) (authorized bool, err error) {
	st := fr.ToStorage()
	if s.autoIndex != nil {
		st = s.autoIndex.stamp(st)
	}
	var refused bool
	// Each Cesium writer writes only its own channels, and returns the frame without
	// the channels it skipped, so the next writer gets that frame.
	for _, w := range s.ws {
		var clockRefused bool
		if st, clockRefused, err = w.Write(ctx, st, s.lacks); err != nil {
			return false, s.fail(err)
		}
		refused = refused || clockRefused
	}
	if s.autoCommit {
		_, commitRefused, err := s.commit(ctx)
		if err != nil {
			return false, err
		}
		refused = refused || commitRefused
	}
	s.writeRefused = s.writeRefused || refused
	return !refused, s.relay(ctx, st)
}

// commit commits the channels the session controls, then syncs the Cesium writers when
// the persist rule asks, also after a commit refused for no control. It returns the
// latest end of the clocks, and true with a zero end when the session lacked control of
// a channel with samples to commit.
func (s *session) commit(ctx context.Context) (telem.TimeStamp, bool, error) {
	var (
		end     = telem.TimeStampMin
		refused bool
	)
	for _, w := range s.ws {
		clockEnd, clockRefused, err := w.Commit(ctx, s.lacks)
		if err != nil {
			return 0, false, s.fail(err)
		}
		refused = refused || clockRefused
		end = max(end, clockEnd)
	}
	if syncErr := s.maybeSync(); syncErr != nil {
		return 0, false, syncErr
	}
	if refused {
		return 0, true, nil
	}
	return end, false, nil
}

// endDomain commits and ends the stretches of the indexes the session may end, and
// commits the channels it controls of the others. It returns an error that wraps
// control.ErrUnauthorized for an index it may not end.
func (s *session) endDomain(ctx context.Context, indexes channel.Keys) error {
	ended, skip, refusal := s.gates.endDomain(indexes)
	if skip != nil {
		for _, w := range s.ws {
			if _, _, err := w.Commit(ctx, skip.Contains); err != nil {
				return s.fail(err)
			}
		}
	}
	for _, w := range s.ws {
		if !slices.Contains(ended, w.Clock) {
			continue
		}
		if err := w.EndStretch(ctx); err != nil {
			return s.fail(err)
		}
	}
	return refusal
}

// fail names the holder in a claim refusal of err and makes err the error of the
// session. An error of one Cesium writer then also ends the writes of the others.
func (s *session) fail(err error) error {
	named, _ := s.controls.nameHolder(err)
	if s.err == nil {
		s.err = named
	}
	return named
}

// maybeSync syncs the Cesium writers on every commit when the session does not auto
// commit, and otherwise once persistInterval has passed since the last sync.
func (s *session) maybeSync() error {
	if s.autoCommit && s.persistInterval > 0 {
		now := s.clock.Now()
		if now.Sub(s.lastSync) < s.persistInterval.Duration() {
			return nil
		}
		s.lastSync = now
	}
	for _, w := range s.ws {
		if err := w.Sync(); err != nil {
			return s.fail(err)
		}
	}
	return nil
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
