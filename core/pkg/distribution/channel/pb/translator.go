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

	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
)

type (
	CreateMessageTranslator struct{}
	DeleteRequestTranslator struct{}
	RenameMessageTranslator struct{}
)

var (
	_ fgrpc.Translator[channel.CreateMessage, *CreateMessage] = (*CreateMessageTranslator)(nil)
	_ fgrpc.Translator[channel.DeleteRequest, *DeleteRequest] = (*DeleteRequestTranslator)(nil)
	_ fgrpc.Translator[channel.RenameRequest, *RenameRequest] = (*RenameMessageTranslator)(nil)
)

func translateOptionsForward(opts channel.CreateOptions) *CreateOptions {
	return &CreateOptions{
		RetrieveIfNameExists:  opts.RetrieveIfNameExists,
		OverwriteIfNameExists: opts.OverwriteIfNameExistsAndDifferentProperties,
	}
}

func translateOptionsBackward(opts *CreateOptions) channel.CreateOptions {
	return channel.CreateOptions{
		RetrieveIfNameExists:                        opts.RetrieveIfNameExists,
		OverwriteIfNameExistsAndDifferentProperties: opts.OverwriteIfNameExists,
	}
}

func (c CreateMessageTranslator) Forward(
	ctx context.Context,
	msg channel.CreateMessage,
) (*CreateMessage, error) {
	channels, err := ChannelsToPB(ctx, msg.Channels)
	if err != nil {
		return nil, err
	}
	return &CreateMessage{
		Channels: channels,
		Opts:     translateOptionsForward(msg.Opts),
	}, nil
}

func (c CreateMessageTranslator) Backward(
	ctx context.Context,
	msg *CreateMessage,
) (channel.CreateMessage, error) {
	channels, err := ChannelsFromPB(ctx, msg.Channels)
	if err != nil {
		return channel.CreateMessage{}, err
	}
	return channel.CreateMessage{
		Channels: channels,
		Opts:     translateOptionsBackward(msg.Opts),
	}, nil
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
