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
	"fmt"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/stage"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/value"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/unsafe"
)

var symbolChannelSink = ir.Symbol{
	Name: "write",
	Kind: ir.KindStage,
	Type: ir.Stage{
		Config: ir.NamedTypes{
			Keys:   []string{"channel", "value"},
			Values: []ir.Type{ir.Chan{}, ir.F32{}},
		},
		Params: ir.NamedTypes{
			Keys:   []string{"input"},
			Values: []ir.Type{ir.NewTypeVariable("T", nil)},
		},
		Return: ir.U8{},
	},
}

type channelSink struct {
	base
	writeChannel channel.Channel
	value        value.Value
	write        func(ctx context.Context, fr core.Frame) error
}

func (c *channelSink) Next(ctx context.Context, _ string, _ value.Value) {
	values := value.ToSeries([]value.Value{c.value}, c.writeChannel.DataType)
	baseFrame := core.UnaryFrame(c.writeChannel.Key(), values)
	if c.writeChannel.Index() != 0 {
		baseFrame = baseFrame.Append(
			c.writeChannel.Index(),
			telem.NewSeriesV[telem.TimeStamp](telem.Now()),
		)
	}
	if err := c.write(ctx, baseFrame); err != nil {
		fmt.Println("failed to write to channel")
	}
}

func createChannelSink(ctx context.Context, cfg Config) (stage.Stage, error) {
	sink := &channelSink{base: base{key: cfg.Node.Key}}
	sink.write = cfg.Write
	sink.value = value.Value{}.Put(cfg.Node.Config["value"])
	writeChanKeys := unsafe.ReinterpretSlice[uint32, channel.Key](cfg.Node.Channels.Read.Keys())
	writeChanKey := writeChanKeys[0]
	if err := cfg.Channel.NewRetrieve().WhereKeys(writeChanKey).Entry(&sink.writeChannel).Exec(ctx, nil); err != nil {
		return nil, err
	}
	if sink.writeChannel.Index() != 0 {
		writeChanKeys = append(writeChanKeys, sink.writeChannel.Index())
	}
	sink.writeChannels = writeChanKeys
	return sink, nil
}
