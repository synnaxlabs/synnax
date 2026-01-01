// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package domain

import "github.com/synnaxlabs/x/telem"

const pointerByteSize = 26

// pointer is a reference to a telemetry blob occupying a particular time domain.
type pointer struct {
	// Bounds is the time interval occupied by the domain. This interval is guaranteed
	// to be unique i.e.it won't overlap with any other domain within the DB. bounds
	// follows the behavior of telem.TimeRange in that the starting point is inclusive,
	// while the ending point is exclusive. If two domains share a common start and end
	// point, they are considered continuous.
	// 16 bytes
	telem.TimeRange
	// fileKey is the numeric key of the file where the data is stored.
	// 2 bytes
	fileKey uint16
	// offset is the offset of the domain within the file.
	// 4 bytes
	offset uint32
	// size is the size of the domain within the file.
	// 4 bytes
	size uint32
}
