// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	channelv1 "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/gen/proto/go/channel/v1"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/telem"
)

type createMessageTranslator struct{}

var _ fgrpc.Translator[channel.CreateMessage, *channelv1.CreateMessage] = (*createMessageTranslator)(nil)

// Forward implements the fgrpc.Translator interface.
func (c createMessageTranslator) Forward(msg channel.CreateMessage) (*channelv1.CreateMessage, error) {
	tr := &channelv1.CreateMessage{}
	for _, ch := range msg.Channels {
		tr.Channels = append(tr.Channels, &channelv1.Channel{
			StorageKey:   int32(ch.StorageKey),
			Name:         ch.Name,
			NodeId:       int32(ch.NodeID),
			DataType:     string(ch.DataType),
			StorageIndex: int32(ch.LocalIndex),
			IsIndex:      ch.IsIndex,
			Rate:         float64(ch.Rate),
		})
	}
	return tr, nil
}

// Backward implements the fgrpc.Translator interface.
func (c createMessageTranslator) Backward(msg *channelv1.CreateMessage) (channel.CreateMessage, error) {
	var tr channel.CreateMessage
	for _, ch := range msg.Channels {
		tr.Channels = append(tr.Channels, channel.Channel{
			StorageKey: storage.ChannelKey(ch.StorageKey),
			Name:       ch.Name,
			NodeID:     dcore.NodeID(ch.NodeId),
			DataType:   telem.DataType(ch.DataType),
			LocalIndex: storage.ChannelKey(ch.StorageIndex),
			IsIndex:    ch.IsIndex,
			Rate:       telem.Rate(ch.Rate),
		})
	}
	return tr, nil
}
