// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"

	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
)

type (
	CreateMessageTranslator struct{}
	DeleteRequestTranslator struct{}
	RenameMessageTranslator struct{}
)

var (
	_ fgrpc.Translator[CreateMessage, *CreateMessage] = (*CreateMessageTranslator)(nil)
	_ fgrpc.Translator[DeleteRequest, *DeleteRequest] = (*DeleteRequestTranslator)(nil)
	_ fgrpc.Translator[RenameRequest, *RenameRequest] = (*RenameMessageTranslator)(nil)
)

func translateOptionsForward(opts CreateOptions) *CreateOptions {
	return &CreateOptions{
		RetrieveIfNameExists:  opts.RetrieveIfNameExists,
		OverwriteIfNameExists: opts.OverwriteIfNameExistsAndDifferentProperties,
	}
}

func translateOptionsBackward(opts *CreateOptions) CreateOptions {
	return CreateOptions{
		RetrieveIfNameExists:                        opts.RetrieveIfNameExists,
		OverwriteIfNameExistsAndDifferentProperties: opts.OverwriteIfNameExists,
	}
}

func (c CreateMessageTranslator) Forward(
	_ context.Context,
	msg CreateMessage,
) (*CreateMessage, error) {
	tr := &CreateMessage{Opts: translateOptionsForward(msg.Opts)}
	for _, ch := range msg.Channels {
		tr.Channels = append(tr.Channels, &Channel{
			Name:        ch.Name,
			Leaseholder: int32(ch.Leaseholder),
			DataType:    string(ch.DataType),
			IsIndex:     ch.IsIndex,
			LocalKey:    uint32(ch.LocalKey),
			LocalIndex:  int32(ch.LocalIndex),
			Concurrency: uint32(ch.Concurrency),
			Internal:    ch.Internal,
			Virtual:     ch.Virtual,
		})
	}
	return tr, nil
}

func (c CreateMessageTranslator) Backward(
	_ context.Context,
	msg *CreateMessage,
) (CreateMessage, error) {
	tr := CreateMessage{Opts: translateOptionsBackward(msg.Opts)}
	for _, ch := range msg.Channels {
		tr.Channels = append(tr.Channels, Channel{
			Name:        ch.Name,
			Leaseholder: cluster.NodeKey(ch.Leaseholder),
			DataType:    telem.DataType(ch.DataType),
			IsIndex:     ch.IsIndex,
			LocalKey:    LocalKey(ch.LocalKey),
			LocalIndex:  LocalKey(ch.LocalIndex),
			Virtual:     ch.Virtual,
			Concurrency: control.Concurrency(ch.Concurrency),
			Internal:    ch.Internal,
		})
	}
	return tr, nil
}

func (d DeleteRequestTranslator) Forward(
	_ context.Context,
	msg DeleteRequest,
) (*DeleteRequest, error) {
	return &DeleteRequest{Keys: msg.Keys.Uint32()}, nil
}

func (d DeleteRequestTranslator) Backward(
	_ context.Context,
	msg *DeleteRequest,
) (DeleteRequest, error) {
	return DeleteRequest{Keys: KeysFromUint32(msg.Keys)}, nil
}

func (r RenameMessageTranslator) Forward(
	_ context.Context,
	msg RenameRequest,
) (*RenameRequest, error) {
	return &RenameRequest{
		Names: msg.Names,
		Keys:  msg.Keys.Uint32(),
	}, nil
}

func (r RenameMessageTranslator) Backward(
	_ context.Context,
	msg *RenameRequest,
) (RenameRequest, error) {
	return RenameRequest{
		Names: msg.Names,
		Keys:  KeysFromUint32(msg.Keys),
	}, nil
}
