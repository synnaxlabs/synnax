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
	"fmt"
	"github.com/synnaxlabs/cesium/internal/version"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type ChannelKey = uint32

type ChannelVariant uint8

const (
	Index ChannelVariant = iota + 1
	Indexed
	Fixed
	Event
)

// Channel is a logical collection of telemetry samples across a time-range. The data
// within a channel typically arrives from a single source. This can be a physical sensor,
// metric, event, or other entity that emits regular, consistent, and time-order values.
// A channel can also be used for storing derived data, such as a moving average or signal
// processing result.
type Channel struct {
	// Key is a unique identifier to the channel within the cesium data engine.
	// [REQUIRED]
	Key ChannelKey `json:"key" msgpack:"key"`
	// Name is a non-unique, human-readable identifier to the channel within the data
	// engine. Note that it is never used to index or retrieve a channel.
	// [OPTIONAL]
	Name string `json:"name" msgpack:"name"`
	// IsIndex determines whether the channel acts as an index channel. If false, then
	// either the channel is a rate-based channel or uses another channel as its Index.
	IsIndex bool
	// DataType is the type of data stored in the channel.
	// [REQUIRED]
	DataType telem.DataType `json:"data_type" msgpack:"data_type"`
	// Rate sets the rate at which the channels values are written. This is used to
	// determine the timestamp of each sample. Rate based channels are far more efficient
	// than index based channels, and should be used whenever channel's values are
	// regularly spaced. One of Index or Rate must be non-zero.
	// [OPTIONAL]
	Rate telem.Rate `json:"rate" msgpack:"rate"`
	// Index is the key of the channel used to index the channel's values. The Index is
	// used to associate a value with a timestamp. If zero, the channel's data will be
	// indexed using its rate. One of Index or Rate must be non-zero.
	// [OPTIONAL]
	Index ChannelKey `json:"index" msgpack:"index"`
	// Virtual specifies whether the channel is virtual.
	// [OPTIONAL]
	Virtual bool `json:"virtual" msgpack:"virtual"`
	// Concurrency specifies the concurrency setting for the channel's controller
	// (Exclusive or shared).
	// [OPTIONAL]
	Concurrency control.Concurrency `json:"concurrency" msgpack:"concurrency"`
	// Version specifies the format of files stored in this channel.
	Version version.Version `json:"version" msgpack:"version"`
}

func (c Channel) String() string {
	if c.Name != "" {
		return fmt.Sprintf("[%s]<%d>", c.Name, c.Key)
	}
	return fmt.Sprintf("<%d>", c.Key)
}

func (c Channel) ValidateSeries(series telem.Series) error {
	if (series.DataType == telem.Int64T || series.DataType == telem.TimeStampT) && (c.DataType == telem.Int64T || c.DataType == telem.TimeStampT) {
		return nil
	}
	if series.DataType != c.DataType {
		return errors.Wrapf(
			validate.Error,
			"invalid array data type for channel %s, expected %s, got %s",
			c.Key,
			c.DataType,
			series.DataType,
		)
	}
	return nil
}
