// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb

import (
	"context"

	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	controlpb "github.com/synnaxlabs/x/control/pb"
	"github.com/synnaxlabs/x/telem"
)

type (
	CreateMessageTranslator struct{}
	DeleteRequestTranslator struct{}
	RenameMessageTranslator struct{}
)

var (
	_ grpc.Translator[channel.CreateMessage, *CreateMessage] = (*CreateMessageTranslator)(nil)
	_ grpc.Translator[channel.DeleteRequest, *DeleteRequest] = (*DeleteRequestTranslator)(nil)
	_ grpc.Translator[channel.RenameRequest, *RenameRequest] = (*RenameMessageTranslator)(nil)
)

// channelToPB converts a minimal distribution channel to its protobuf representation.
// Only the storage and routing fields the distribution layer carries are mapped; the
// rich metadata fields on the protobuf message (internal/operations/expression) are
// owned by the service layer and left zero on the cluster-internal wire.
func channelToPB(c channel.Channel) (*Channel, error) {
	concurrency, err := controlpb.ConcurrencyToPB(c.Concurrency)
	if err != nil {
		return nil, err
	}
	return &Channel{
		Name:        string(c.Name),
		Leaseholder: uint32(c.Leaseholder),
		DataType:    string(c.DataType),
		IsIndex:     c.IsIndex,
		LocalKey:    uint32(c.LocalKey),
		LocalIndex:  uint32(c.LocalIndex),
		Virtual:     c.Virtual,
		Concurrency: concurrency,
	}, nil
}

func channelFromPB(pb *Channel) (channel.Channel, error) {
	if pb == nil {
		return channel.Channel{}, nil
	}
	concurrency, err := controlpb.ConcurrencyFromPB(pb.Concurrency)
	if err != nil {
		return channel.Channel{}, err
	}
	return channel.Channel{
		Name:        channel.Name(pb.Name),
		Leaseholder: node.Key(pb.Leaseholder),
		DataType:    telem.DataType(pb.DataType),
		IsIndex:     pb.IsIndex,
		LocalKey:    channel.LocalKey(pb.LocalKey),
		LocalIndex:  channel.LocalKey(pb.LocalIndex),
		Virtual:     pb.Virtual,
		Concurrency: concurrency,
	}, nil
}

func channelsToPB(cs []channel.Channel) ([]*Channel, error) {
	res := make([]*Channel, len(cs))
	for i := range cs {
		var err error
		if res[i], err = channelToPB(cs[i]); err != nil {
			return nil, err
		}
	}
	return res, nil
}

func channelsFromPB(pbs []*Channel) ([]channel.Channel, error) {
	res := make([]channel.Channel, len(pbs))
	for i, pb := range pbs {
		var err error
		if res[i], err = channelFromPB(pb); err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (c CreateMessageTranslator) Forward(
	_ context.Context,
	msg channel.CreateMessage,
) (*CreateMessage, error) {
	channels, err := channelsToPB(msg.Channels)
	if err != nil {
		return nil, err
	}
	return &CreateMessage{Channels: channels}, nil
}

func (c CreateMessageTranslator) Backward(
	_ context.Context,
	msg *CreateMessage,
) (channel.CreateMessage, error) {
	channels, err := channelsFromPB(msg.Channels)
	if err != nil {
		return channel.CreateMessage{}, err
	}
	return channel.CreateMessage{Channels: channels}, nil
}

func (d DeleteRequestTranslator) Forward(
	_ context.Context,
	msg channel.DeleteRequest,
) (*DeleteRequest, error) {
	return &DeleteRequest{Keys: msg.Keys.Uint32()}, nil
}

func (d DeleteRequestTranslator) Backward(
	_ context.Context,
	msg *DeleteRequest,
) (channel.DeleteRequest, error) {
	return channel.DeleteRequest{Keys: channel.KeysFromUint32(msg.Keys)}, nil
}

func (r RenameMessageTranslator) Forward(
	_ context.Context,
	msg channel.RenameRequest,
) (*RenameRequest, error) {
	return &RenameRequest{
		Names: msg.Names,
		Keys:  msg.Keys.Uint32(),
	}, nil
}

func (r RenameMessageTranslator) Backward(
	_ context.Context,
	msg *RenameRequest,
) (channel.RenameRequest, error) {
	return channel.RenameRequest{
		Names: msg.Names,
		Keys:  channel.KeysFromUint32(msg.Keys),
	}, nil
}
