// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ingest

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"

	typev1 "github.com/sift-stack/sift/go/gen/sift/common/type/v1"
	ingestionconfigsv2 "github.com/sift-stack/sift/go/gen/sift/ingestion_configs/v2"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// maxNameLength is the longest channel or flow name Sift accepts.
const maxNameLength = 255

var dataTypes = map[telem.DataType]typev1.ChannelDataType{
	telem.Float64T:   typev1.ChannelDataType_CHANNEL_DATA_TYPE_DOUBLE,
	telem.Float32T:   typev1.ChannelDataType_CHANNEL_DATA_TYPE_FLOAT,
	telem.Int8T:      typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_32,
	telem.Int16T:     typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_32,
	telem.Int32T:     typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_32,
	telem.Int64T:     typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_64,
	telem.TimestampT: typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_64,
	telem.Uint8T:     typev1.ChannelDataType_CHANNEL_DATA_TYPE_UINT_32,
	telem.Uint16T:    typev1.ChannelDataType_CHANNEL_DATA_TYPE_UINT_32,
	telem.Uint32T:    typev1.ChannelDataType_CHANNEL_DATA_TYPE_UINT_32,
	telem.Uint64T:    typev1.ChannelDataType_CHANNEL_DATA_TYPE_UINT_64,
	telem.BooleanT:   typev1.ChannelDataType_CHANNEL_DATA_TYPE_BOOL,
	telem.StringT:    typev1.ChannelDataType_CHANNEL_DATA_TYPE_STRING,
	telem.JSONT:      typev1.ChannelDataType_CHANNEL_DATA_TYPE_STRING,
	telem.UUIDT:      typev1.ChannelDataType_CHANNEL_DATA_TYPE_STRING,
	telem.BytesT:     typev1.ChannelDataType_CHANNEL_DATA_TYPE_BYTES,
}

// flow is the Sift flow for the channels that share one index. Sift maps each value
// of a request to a channel by its position in channels.
type flow struct {
	index    channel.Channel
	channels []channel.Channel
}

// keys returns the keys the engine reads for the flow: the index first.
func (f flow) keys() channel.Keys {
	keys := make(channel.Keys, 0, len(f.channels)+1)
	keys = append(keys, f.index.Key())
	for _, ch := range f.channels {
		keys = append(keys, ch.Key())
	}
	return keys
}

func (f flow) config() *ingestionconfigsv2.FlowConfig {
	cfg := &ingestionconfigsv2.FlowConfig{
		Name:     f.index.Name,
		Channels: make([]*ingestionconfigsv2.ChannelConfig, len(f.channels)),
	}
	for i, ch := range f.channels {
		cfg.Channels[i] = &ingestionconfigsv2.ChannelConfig{
			Name:     ch.Name,
			DataType: dataTypes[ch.DataType],
		}
	}
	return cfg
}

// buildFlows groups the data channels in channels into one flow per index, ordered by
// index key, with each flow's channels ordered by key. indexes holds the index
// channels of the data channels. It returns a validation error when Sift cannot take a
// channel.
func buildFlows(channels, indexes []channel.Channel) ([]flow, error) {
	byIndex := make(map[channel.Key]*flow)
	for _, idx := range indexes {
		byIndex[idx.Key()] = &flow{index: idx}
	}
	for _, ch := range channels {
		if ch.IsIndex {
			continue
		}
		if err := validateChannel(ch); err != nil {
			return nil, err
		}
		f := byIndex[ch.Index()]
		f.channels = append(f.channels, ch)
	}
	flows := make([]flow, 0, len(byIndex))
	for _, f := range byIndex {
		if len(f.channels) == 0 {
			continue
		}
		// Sift names each flow after its index.
		if err := validateName(f.index); err != nil {
			return nil, err
		}
		slices.SortFunc(f.channels, func(a, b channel.Channel) int {
			return cmp.Compare(a.Key(), b.Key())
		})
		flows = append(flows, *f)
	}
	if len(flows) == 0 {
		return nil, validate.Errorf("select at least one channel that is not an index")
	}
	slices.SortFunc(flows, func(a, b flow) int {
		return cmp.Compare(a.index.Key(), b.index.Key())
	})
	return flows, nil
}

func validateChannel(ch channel.Channel) error {
	_, storable := dataTypes[ch.DataType]
	var reason string
	switch {
	case ch.IsCalculated():
		reason = "is calculated, and Sift can only take stored channels"
	case ch.Virtual:
		reason = "is virtual, and the Core stores no data for virtual channels"
	case ch.Index() == 0:
		reason = "has no index, so its samples have no timestamps"
	case !storable:
		reason = fmt.Sprintf("has data type %s, which Sift cannot store", ch.DataType)
	default:
		return validateName(ch)
	}
	return validate.Errorf("channel %s %s", ch.Name, reason)
}

func validateName(ch channel.Channel) error {
	if len(ch.Name) <= maxNameLength {
		return nil
	}
	return validate.Errorf(
		"channel %s has a name longer than the %d characters Sift allows",
		ch.Name,
		maxNameLength,
	)
}

// clientKey returns the ingestion config key for flows on asset. Equal flows on the
// same asset give the same key, so the engine finds the config it made before.
func clientKey(asset string, flows []flow) string {
	h := sha256.New()
	write := func(s string) {
		h.Write([]byte(s))
		h.Write([]byte{0})
	}
	write(asset)
	for _, f := range flows {
		write(f.index.Name)
		for _, ch := range f.channels {
			write(ch.Name)
			write(string(ch.DataType))
		}
	}
	return "synnax-" + hex.EncodeToString(h.Sum(nil))[:32]
}
