// Copyright 2025 Synnax Labs, Inc.
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

// Channel is a logical collection of telemetry samples across a time-range. The data
// within a channel typically arrives from a single source. This can be a physical sensor,
// metric, event, or entity that emits regular, consistent, and time-order values.
// A channel can also be used for storing derived data, such as a moving average or signal
// processing result.
type Channel struct {
	// Key is a unique identifier to the channel within the cesium.
	// [REQUIRED]
	Key ChannelKey `json:"key" msgpack:"key"`
	// Name is a non-unique, human-readable identifier to the channel within the data
	// engine. Note that it is never used to index or retrieve a channel.
	// [REQUIRED]
	Name string `json:"name" msgpack:"name"`
	// IsIndex determines whether the channel acts as an index channel. If false, then
	// the channel is a data channel, and the Index field must be set to the key of
	// an existing, valid index channel.
	// [OPTIONAL]
	IsIndex bool
	// DataType is the type of data stored in the channel.
	// [REQUIRED]
	DataType telem.DataType `json:"data_type" msgpack:"data_type"`
	// Index is the key of the channel used to index the channel's values. The Index is
	// used to associate a value in a data channel with a corresponding timestamp.
	// [OPTIONAL if IsIndex is true and REQUIRED if IsIndex is false]
	Index ChannelKey `json:"index" msgpack:"index"`
	// Virtual specifies whether the channel is virtual. Virtual channels do not store
	// any data and do not require an index.
	// [OPTIONAL]
	Virtual bool `json:"virtual" msgpack:"virtual"`
	// Concurrency specifies the concurrency setting for the channel's controller
	// (Exclusive or shared).
	// [OPTIONAL]
	Concurrency control.Concurrency `json:"concurrency" msgpack:"concurrency"`
	// Version specifies the format of files stored in this channel.
	// [OPTIONAL]
	Version version.Version `json:"version" msgpack:"version"`
}

func (c Channel) String() string {
	if c.Name != "" {
		return fmt.Sprintf("[%s]<%d>", c.Name, c.Key)
	}
	return fmt.Sprintf("<%d>", c.Key)
}

func (c Channel) ValidateSeries(series telem.Series) error {
	sDt := series.DataType
	cDt := c.DataType
	isEquivalent := (sDt == telem.Int64T || sDt == telem.TimeStampT) && (cDt == telem.Int64T || cDt == telem.TimeStampT)
	if cDt != sDt && !isEquivalent {
		return errors.Wrapf(
			validate.Error,
			"invalid data type for channel %v, expected %s, got %s",
			c,
			cDt,
			sDt,
		)
	}
	return nil
}
