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

	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/set"
)

func translateResponse(
	_ context.Context,
	res cesium.IteratorResponse,
) (Response, bool, error) {
	return Response{
		Ack:     res.Ack,
		Variant: ResponseVariant(res.Variant),
		SeqNum:  res.SeqNum,
		Error:   res.Err,
		Command: Command(res.Command),
		Frame:   frame.NewFromStorage(res.Frame),
	}, true, nil
}

// keepTimed returns a response translator that keeps the series of cfg.Clocks only
// where the series of another key of cfg names them as its clock.
func keepTimed(
	cfg Config,
) func(context.Context, cesium.IteratorResponse) (Response, bool, error) {
	clocks := set.New(cfg.Clocks...)
	kept := set.New(lo.Reject(cfg.Keys, func(k channel.Key, _ int) bool {
		return clocks.Contains(k)
	})...)
	untimed := set.New(cfg.UntimedKeys...)
	return func(ctx context.Context, res cesium.IteratorResponse) (Response, bool, error) {
		out, ok, err := translateResponse(ctx, res)
		if out.Variant != ResponseVariantData {
			return out, ok, err
		}
		out.Frame = out.Frame.KeepTimed(kept, untimed)
		return out, out.Frame.Count() > 0, err
	}
}

func newStorageRequestTranslator() func(
	ctx context.Context,
	in Request,
) (cesium.IteratorRequest, bool, error) {
	seqNum := 0
	return func(ctx context.Context, req Request) (cesium.IteratorRequest, bool, error) {
		oReq := cesium.IteratorRequest{
			Command: cesium.IteratorCommand(req.Command),
			Span:    req.Span,
			Stamp:   req.Stamp,
			Bounds:  req.Bounds,
			SeqNum:  seqNum,
		}
		seqNum++
		return oReq, true, nil
	}
}

func (s *Service) newStorage(
	cfg Config,
) (confluence.Segment[Request, Response], error) {
	iter, err := s.cfg.TS.NewStreamIterator(cesium.IteratorConfig{
		Bounds:        cfg.Bounds,
		Channels:      cfg.Keys.Uint32(),
		AutoChunkSize: cfg.ChunkSize,
		Reduction:     cfg.Reduction,
	})
	if err != nil {
		return nil, err
	}
	pipe := plumber.New()
	reqT := &confluence.LinearTransform[Request, cesium.IteratorRequest]{}
	reqT.Transform = newStorageRequestTranslator()
	resT := &confluence.LinearTransform[cesium.IteratorResponse, Response]{}
	resT.Transform = translateResponse
	if len(cfg.Clocks) > 0 {
		resT.Transform = keepTimed(cfg)
	}
	pipe.SetSegment[cesium.IteratorRequest, cesium.IteratorResponse]("storage", iter)
	pipe.SetSegment[Request, cesium.IteratorRequest]("requests", reqT)
	pipe.SetSegment[cesium.IteratorResponse, Response]("responses", resT)
	pipe.MustConnect[cesium.IteratorRequest]("requests", "storage", 1)
	pipe.MustConnect[cesium.IteratorResponse]("storage", "responses", 1)
	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	lo.Must0(seg.RouteInletTo("requests"))
	lo.Must0(seg.RouteOutletFrom("responses"))
	return seg, nil
}
