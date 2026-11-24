// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package index

import (
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/x/telem"
)

type TimeStampApproximation = domain.Approximation[telem.TimeStamp]

type DistanceApproximation struct {
	domain.Approximation[int64]
	// StartExact denotes whether the start timestamp of the requested time range is in
	// the index.
	StartExact bool
	// EndExact denotes whether the end timestamp of the requested time range is in the
	// index.
	EndExact bool
}
