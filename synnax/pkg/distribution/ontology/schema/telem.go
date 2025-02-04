// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schema

const TimeRangeType Type = "time-range"

var TimeRange = &Schema{
	Type: TimeRangeType,
	Fields: map[string]Field{
		"start": {Type: Int64},
		"end":   {Type: Int64},
	},
}
