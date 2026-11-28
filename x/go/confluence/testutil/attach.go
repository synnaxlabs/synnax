// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil holds test helpers for confluence segments.
package testutil

import "github.com/synnaxlabs/x/confluence"

// Attach connects new streams with buffer slots to the inlet and the outlet of seg,
// and returns the stream that feeds seg and the stream that seg feeds.
func Attach[I, O confluence.Value](
	seg confluence.Segment[I, O],
	buffer ...int,
) (confluence.Inlet[I], confluence.Outlet[O]) {
	req := confluence.NewStream[I](buffer...)
	res := confluence.NewStream[O](buffer...)
	seg.InFrom(req)
	seg.OutTo(res)
	return req, res
}
