// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package index

import (
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

type Rate struct {
	Rate   telem.Rate
	Logger *zap.Logger
}

var _ Index = Rate{}

// Distance implements Index.
func (r Rate) Distance(tr telem.TimeRange, continuous bool) (approx DistanceApproximation, err error) {
	r.Logger.Debug("idx distance",
		zap.Stringer("timeRange", tr),
		zap.Bool("continuous", continuous),
	)
	defer func() {
		r.Logger.Debug("idx distance done",
			zap.Stringer("timeRange", tr),
			zap.Bool("continuous", continuous),
			zap.Stringer("approx", approx),
		)
	}()

	approx = Between(
		int64(r.Rate.ClosestGE(tr.Start).Span(r.Rate.ClosestLE(tr.End))/r.Rate.Period()),
		int64(r.Rate.ClosestLE(tr.Start).Span(r.Rate.ClosestGE(tr.End))/r.Rate.Period()),
	)
	return
}

// Stamp implements Searcher.
func (r Rate) Stamp(ref telem.TimeStamp, distance int64, _ bool) (approx TimeStampApproximation, err error) {
	approx = Between(
		r.Rate.ClosestLE(ref).Add(r.Rate.Span(int(distance))),
		r.Rate.ClosestGE(ref).Add(r.Rate.Span(int(distance))),
	)
	return
}
