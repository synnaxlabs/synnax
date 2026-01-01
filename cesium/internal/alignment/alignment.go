// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alignment

import (
	"math"

	"github.com/synnaxlabs/x/telem"
)

// ZeroLeading represents the start of a region reserved for written data that has not
// yet been persisted. This is useful for correctly ordering new data while ensuring
// that it is significantly after any persisted data.
const ZeroLeading uint32 = math.MaxUint32 - 1e6

// Leading returns an Alignment whose array index is the maximum possible value
// and whose sample index is the provided value.
func Leading(domainIdx, sampleIdx uint32) telem.Alignment {
	return telem.NewAlignment(ZeroLeading+domainIdx, sampleIdx)
}
