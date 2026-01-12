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
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
)

func (s *Service) newGateway(cfg Config, generateSeqNums bool) (confluence.Segment[Request, Response], error) {
	iter, err := s.cfg.TS.NewStreamIterator(ts.IteratorConfig{
		Bounds:        cfg.Bounds,
		Channels:      cfg.Keys.Storage(),
		AutoChunkSize: cfg.ChunkSize,
	})
	if err != nil {
		return nil, err
	}
	pipe := plumber.New()
	reqT := &confluence.LinearTransform[Request, ts.IteratorRequest]{}
	reqT.Transform = newStorageRequestTranslator(generateSeqNums)
	resT := &confluence.LinearTransform[ts.IteratorResponse, Response]{}
	resT.Transform = newStorageResponseTranslator(s.cfg.HostResolver.HostKey())
	plumber.SetSegment[ts.IteratorRequest, ts.IteratorResponse](pipe, "storage", iter)
	plumber.SetSegment[Request, ts.IteratorRequest](pipe, "requests", reqT)
	plumber.SetSegment[ts.IteratorResponse, Response](pipe, "responses", resT)
	plumber.MustConnect[ts.IteratorRequest](pipe, "requests", "storage", 1)
	plumber.MustConnect[ts.IteratorResponse](pipe, "storage", "responses", 1)
	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	lo.Must0(seg.RouteInletTo("requests"))
	lo.Must0(seg.RouteOutletFrom("responses"))
	return seg, nil
}
