// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package std

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/stage"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/value"
	"github.com/synnaxlabs/x/unsafe"
)

var symbolChannelSource = ir.Symbol{
	Name: "on",
	Kind: ir.KindStage,
	Type: ir.Stage{
		Config: ir.NamedTypes{
			Keys:   []string{"channel"},
			Values: []ir.Type{ir.Chan{}},
		},
		Return: ir.NewTypeVariable("T", nil),
	},
}

type channelSource struct {
	base
	values ChannelData
}

func (c *channelSource) Next(ctx context.Context, _ string, _ value.Value) {
	key := c.readChannels[0]
	values := value.FromSeries(c.values.Get(key))
	for _, v := range values {
		c.outputHandler(ctx, "output", v)
	}
}

func createChannelSource(_ context.Context, cfg Config) (stage.Stage, error) {
	source := &channelSource{base: base{key: cfg.Node.Key}, values: cfg.ChannelData}
	source.readChannels = unsafe.ReinterpretSlice[uint32, channel.Key](cfg.Node.Channels.Read.Keys())
	return source, nil
}
