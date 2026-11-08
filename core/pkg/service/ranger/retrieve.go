// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/telem"
)

// Retrieve is used to retrieve ranges from the cluster using a builder pattern.
type Retrieve struct {
	baseTX  gorp.Tx
	gorp    gorp.Retrieve[Key, Range]
	indexes indexes
}

// MatchOverlaps returns a filter that matches ranges whose TimeRange overlaps tr.
func MatchOverlaps(tr telem.TimeRange) Filter {
	return func(_ Retrieve) gorp.Filter[Key, Range] {
		return gorp.Match(func(_ gorp.Context, rng *Range) (bool, error) {
			return rng.TimeRange.OverlapsWith(tr), nil
		})
	}
}
