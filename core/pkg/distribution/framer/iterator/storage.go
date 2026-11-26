// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator

import (
	"context"
	"github.com/synnaxlabs/cesium"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
)

// storage runs the requests of a stream on one storage iterator, in order. It answers
// each request with an acknowledgement, after one data response for each chunk that a
// Next or a Prev reads.
type storage struct {
	confluence.UnarySink[Request]
	confluence.AbstractUnarySource[Response]
	iter *cesium.Iterator
	// seqNum is the sequence number of the next request.
	seqNum int
}

var validateCommand = validate.NewInclusiveBoundsChecker(CommandNext, CommandSetBounds)

func (s *Service) newStorage(
	cfg Config,
) (confluence.Segment[Request, Response], error) {
	iter, err := s.cfg.TS.OpenIterator(cesium.IteratorConfig{
		Bounds:        cfg.Bounds,
		Channels:      cfg.Keys.Uint32(),
		AutoChunkSize: cfg.ChunkSize,
		Reduction:     cfg.Reduction,
	})
	if err != nil {
		return nil, err
	}
	return &storage{iter: iter}, nil
}

// Flow implements confluence.Segment.
func (s *storage) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(s.Out)
	sCtx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return errors.Combine(s.iter.Close(), ctx.Err())
			case req, ok := <-s.In.Outlet():
				if !ok {
					return s.iter.Close()
				}
				seqNum := s.seqNum
				s.seqNum++
				ok, err := s.exec(ctx, req, seqNum)
				if err := signal.SendUnderContext(ctx, s.Out.Inlet(), Response{
					Variant: ResponseVariantAck,
					Command: req.Command,
					SeqNum:  seqNum,
					Ack:     ok,
					Error:   err,
				}); err != nil {
					return errors.Combine(s.iter.Close(), err)
				}
			}
		}
	}, o.Signal...)
}

func (s *storage) exec(ctx context.Context, req Request, seqNum int) (bool, error) {
	if err := validateCommand(req.Command); err != nil {
		return false, err
	}
	emit := func(fr cesium.Frame) error {
		return signal.SendUnderContext(ctx, s.Out.Inlet(), Response{
			Variant: ResponseVariantData,
			Command: CommandNext,
			SeqNum:  seqNum,
			Frame:   frame.NewFromStorage(fr),
		})
	}
	switch req.Command {
	case CommandNext:
		return s.iter.NextChunks(ctx, req.Span, emit)
	case CommandPrev:
		return s.iter.PrevChunks(ctx, req.Span, emit)
	case CommandSeekFirst:
		return s.iter.SeekFirst(), nil
	case CommandSeekLast:
		return s.iter.SeekLast(), nil
	case CommandSeekLE:
		return s.iter.SeekLE(req.Stamp), nil
	case CommandSeekGE:
		return s.iter.SeekGE(req.Stamp), nil
	case CommandValid:
		return s.iter.Valid(), nil
	case CommandError:
		return false, s.iter.Error()
	default:
		s.iter.SetBounds(req.Bounds)
		return true, nil
	}
}
