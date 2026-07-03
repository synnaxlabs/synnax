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

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	control "github.com/synnaxlabs/x/control/pb"
	"github.com/synnaxlabs/x/telem"
)

var (
	CreateMessageTranslator grpc.Translator[channel.CreateMessage, *CreateMessage] = createMessageTranslator{}
	DeleteRequestTranslator grpc.Translator[channel.DeleteRequest, *DeleteRequest] = deleteRequestTranslator{}
	RenameMessageTranslator grpc.Translator[channel.RenameRequest, *RenameRequest] = renameMessageTranslator{}
)

type createMessageTranslator struct{}

func (createMessageTranslator) Forward(
	_ context.Context,
	msg channel.CreateMessage,
) (*CreateMessage, error) {
	channels, err := lo.MapErr(
		msg.Channels,
		func(c channel.Channel, _ int) (*Channel, error) {
			concurrency, err := control.ConcurrencyToPB(c.Concurrency)
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
		})
	if err != nil {
		return nil, err
	}
	return &CreateMessage{Channels: channels}, nil
}

func (createMessageTranslator) Backward(
	_ context.Context,
	msg *CreateMessage,
) (channel.CreateMessage, error) {
	channels, err := lo.MapErr(
		msg.Channels,
		func(c *Channel, _ int) (channel.Channel, error) {
			if c == nil {
				return channel.Channel{}, nil
			}
			concurrency, err := control.ConcurrencyFromPB(c.Concurrency)
			if err != nil {
				return channel.Channel{}, err
			}
			return channel.Channel{
				Name:        c.Name,
				Leaseholder: node.Key(c.Leaseholder),
				DataType:    telem.DataType(c.DataType),
				IsIndex:     c.IsIndex,
				LocalKey:    channel.LocalKey(c.LocalKey),
				LocalIndex:  channel.LocalKey(c.LocalIndex),
				Virtual:     c.Virtual,
				Concurrency: concurrency,
			}, nil
		})
	if err != nil {
		return channel.CreateMessage{}, err
	}
	return channel.CreateMessage{Channels: channels}, nil
}

func (deleteRequestTranslator) Forward(
	_ context.Context,
	req channel.DeleteRequest,
) (*DeleteRequest, error) {
	return &DeleteRequest{Keys: req.Keys.Uint32()}, nil
}

type deleteRequestTranslator struct{}

func (deleteRequestTranslator) Backward(
	_ context.Context,
	req *DeleteRequest,
) (channel.DeleteRequest, error) {
	return channel.DeleteRequest{Keys: channel.KeysFromUint32(req.Keys)}, nil
}

type renameMessageTranslator struct{}

func (renameMessageTranslator) Forward(
	_ context.Context,
	req channel.RenameRequest,
) (*RenameRequest, error) {
	renames := lo.MapKeys(req.Renames, func(_ string, key channel.Key) uint32 {
		return uint32(key)
	})
	return &RenameRequest{Renames: renames}, nil
}

func (renameMessageTranslator) Backward(
	_ context.Context,
	req *RenameRequest,
) (channel.RenameRequest, error) {
	renames := lo.MapKeys(req.Renames, func(_ string, key uint32) channel.Key {
		return channel.Key(key)
	})
	return channel.RenameRequest{Renames: renames}, nil
}
