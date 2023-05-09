// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package core

import (
	"github.com/synnaxlabs/x/telem"
)

// Channel is a logical collection of telemetry samples across a time-range. The data
// within a channel typically arrives from a single source. This can be a physical sensor,
// metric, event, or other entity that emits regular, consistent, and time-order values.
// A channel can also be used for storing derived data, such as a moving average or signal
// processing result.
type Channel struct {
	// Key is a unique identifier to the channel within the cesium.
	// [REQUIRED]
	Key string `json:"key" msgpack:"key"`
	// DataType is the type of data stored in the channel.
	// [REQUIRED]
	DataType telem.DataType `json:"data_type" msgpack:"data_type"`
	// Rate sets the rate at which the channels values are written. This is used to
	// determine the timestamp of each sample. Rate based channels are far more efficient
	// than index based channels, and should be used whenever channel's values are
	// regularly spaced.
	Rate telem.Rate `json:"rate" msgpack:"rate"`
	// Index is the key of the channel used to index the channel's values. The Index is
	// used to associate a value with a timestamp. If zero, the channel's data will be
	// indexed using its rate. One of Index or Rate must be non-zero.
	Index string `json:"index" msgpack:"index"`
	// IsIndex should be set to true if the channel should be indexed. Index channels
	// must use int64 nanosecond timestamps in ascending order.
	IsIndex bool `json:"is_index" msgpack:"is_index"`
}
