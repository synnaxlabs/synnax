// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package index

import (
	"context"
	"fmt"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/telem"
)

type Rate struct {
	Rate    telem.Rate
	Channel core.Channel
}

var _ Index = Rate{}

func (r Rate) Distance(_ context.Context, tr telem.TimeRange, _ bool) (DistanceApproximation, DomainBounds, error) {
	return DistanceApproximation{
		Approximation: Between(
			int64(r.Rate.ClosestGE(tr.Start).Span(r.Rate.ClosestLE(tr.End))/r.Rate.Period()),
			int64(r.Rate.ClosestLE(tr.Start).Span(r.Rate.ClosestGE(tr.End))/r.Rate.Period()),
		),
		StartExact: true,
		EndExact:   true,
	}, DomainBounds{}, nil
}

func (r Rate) Stamp(_ context.Context, ref telem.TimeStamp, distance int64, _ bool) (TimeStampApproximation, error) {
	return Between(
		r.Rate.ClosestLE(ref).Add(r.Rate.Span(int(distance))),
		r.Rate.ClosestGE(ref).Add(r.Rate.Span(int(distance))),
	), nil
}

func (r Rate) Info() string {
	return fmt.Sprintf("rate index: %v", r.Channel)
}

func (r Rate) Domain(_ context.Context, ref telem.TimeStamp) (DomainBounds, error) {
	return ExactDomainBounds(0), nil
}
