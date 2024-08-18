// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	channelv1 "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/channel/v1"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
)

type (
	createMessageTranslator struct{}
	deleteRequestTranslator struct{}
	renameMessageTranslator struct{}
)

var (
	_ fgrpc.Translator[channel.CreateMessage, *channelv1.CreateMessage] = (*createMessageTranslator)(nil)
	_ fgrpc.Translator[channel.DeleteRequest, *channelv1.DeleteRequest] = (*deleteRequestTranslator)(nil)
	_ fgrpc.Translator[channel.RenameRequest, *channelv1.RenameRequest] = (*renameMessageTranslator)(nil)
)

func (c createMessageTranslator) Forward(
	_ context.Context,
	msg channel.CreateMessage,
) (*channelv1.CreateMessage, error) {
	tr := &channelv1.CreateMessage{RetrieveIfNameExists: msg.RetrieveIfNameExists}
	for _, ch := range msg.Channels {
		tr.Channels = append(tr.Channels, &channelv1.Channel{
			Name:        ch.Name,
			Leaseholder: int32(ch.Leaseholder),
			DataType:    string(ch.DataType),
			IsIndex:     ch.IsIndex,
			Rate:        float64(ch.Rate),
			LocalKey:    uint32(ch.LocalKey),
			LocalIndex:  int32(ch.LocalIndex),
			Concurrency: uint32(ch.Concurrency),
			Internal:    ch.Internal,
		})
	}
	return tr, nil
}

func (c createMessageTranslator) Backward(
	_ context.Context,
	msg *channelv1.CreateMessage,
) (channel.CreateMessage, error) {
	tr := channel.CreateMessage{RetrieveIfNameExists: msg.RetrieveIfNameExists}
	for _, ch := range msg.Channels {
		tr.Channels = append(tr.Channels, channel.Channel{
			Name:        ch.Name,
			Leaseholder: dcore.NodeKey(ch.Leaseholder),
			DataType:    telem.DataType(ch.DataType),
			IsIndex:     ch.IsIndex,
			Rate:        telem.Rate(ch.Rate),
			LocalKey:    channel.LocalKey(ch.LocalKey),
			LocalIndex:  channel.LocalKey(ch.LocalIndex),
			Virtual:     ch.Virtual,
			Concurrency: control.Concurrency(ch.Concurrency),
			Internal:    ch.Internal,
		})
	}
	return tr, nil
}

func (d deleteRequestTranslator) Forward(
	_ context.Context,
	msg channel.DeleteRequest,
) (*channelv1.DeleteRequest, error) {
	return &channelv1.DeleteRequest{Keys: msg.Keys.Uint32()}, nil
}

func (d deleteRequestTranslator) Backward(
	_ context.Context,
	msg *channelv1.DeleteRequest,
) (channel.DeleteRequest, error) {
	return channel.DeleteRequest{Keys: channel.KeysFromUint32(msg.Keys)}, nil
}

func (r renameMessageTranslator) Forward(
	_ context.Context,
	msg channel.RenameRequest,
) (*channelv1.RenameRequest, error) {
	return &channelv1.RenameRequest{
		Names: msg.Names,
		Keys:  msg.Keys.Uint32(),
	}, nil
}

func (r renameMessageTranslator) Backward(
	_ context.Context,
	msg *channelv1.RenameRequest,
) (channel.RenameRequest, error) {
	return channel.RenameRequest{
		Names: msg.Names,
		Keys:  channel.KeysFromUint32(msg.Keys),
	}, nil
}
