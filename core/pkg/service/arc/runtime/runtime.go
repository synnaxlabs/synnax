// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package runtime

import (
	"context"

	"github.com/envoyproxy/protoc-gen-validate/module"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/compiler/runtime/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/stage"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"github.com/tetratelabs/wazero"
)

type Runtime struct {
	wasm wazero.Runtime
}

type Config struct {
	Module  *arc.Module
	Channel channel.Readable
	Framer  *framer.Service
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Module = override.Nil(c.Module, other.Module)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Framer = override.Nil(c.Framer, other.Framer)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("arc.runtime")
	validate.NotNil(v, "module", c.Module)
	validate.NotNil(v, "Framer", c.Framer)
	validate.NotNil(v, "Channel", c.Channel)
	return v.Error()
}

func New(ctx context.Context, cfgs ...Config) (*Runtime, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	r := &Runtime{wasm: wazero.NewRuntime(ctx)}
	currValues := make(map[channel.Key]telem.Series)
	mockRuntime := mock.New()
	mockRuntime.ChannelRead["f32"] = func(ctx context.Context, channelKey uint32) float32 {
		ser, ok := currValues[channel.Key(channelKey)]
		if !ok || ser.Len() == 0 {
			return 0.0
		}
		return telem.ValueAt[float32](ser, 0)
	}

	if err = mockRuntime.Bind(ctx, r.wasm); err != nil {
		return nil, err
	}

	mod, err := r.wasm.Instantiate(ctx, cfg.Module.WASM)
	if err != nil {
		return nil, err
	}
	collectedReadChannelNames := make(set.Set[channel.Key])
	for _, task := range cfg.Module.Tasks {
		collectedReadChannelNames.Add(task.Channels.Read...)
	}
	channels := make([]channel.Channel, 0, len(collectedReadChannelNames))
	if err := cfg.Channel.NewRetrieve().
		WhereKeys(collectedReadChannelNames.Keys()...).
		Entries(&channels).
		Exec(ctx, nil); err != nil {
		return nil, err
	}
	readChannelKeys := channel.KeysFromChannels(channels)
	streamer, err := cfg.Framer.NewStreamer(
		ctx,
		framer.StreamerConfig{Keys: readChannelKeys},
	)
	if err != nil {
		return nil, err
	}

	stages := make(map[string]stage.Stage)

	createOnOutput := func(k string) stage.OutputHandler {
		return func(ctx context.Context, value stage.Value) {
			for _, edge := range cfg.Module.Edges {
				if edge.Source.Node == k && edge.Source.Param == k {
					value.Param = edge.Target.Param
					stages[edge.Target.Node].Next(ctx, value)
				}
			}
		}
	}

	p := plumber.New()
	plumber.SetSegment(p, "streamer", streamer)
	sp := &streamProcessor{
		Module:     cfg.Module,
		stages:     stages,
		currValues: currValues,
	}
	sp.Sink = sp.sink
	plumber.SetSink[framer.StreamerResponse](p, "stream_processor", sp)
	streamer.InFrom(confluence.NewStream[framer.StreamerRequest]())
	plumber.MustConnect[framer.StreamerResponse](p, "streamer", "stream_processor", 10)
	sCtx, _ := signal.Isolated()
	p.Flow(sCtx)
	for _, task := range stages {
		task.Flow(sCtx)
	}
	return r, err
}

type streamProcessor struct {
	Module *module.Module
	confluence.UnarySink[framer.StreamerResponse]
	currValues     map[channel.Key]telem.Series
	stages         map[string]stage.Stage
	streamRequests confluence.Inlet[framer.StreamerRequest]
}

func (s *streamProcessor) sink(ctx context.Context, value framer.StreamerResponse) error {
	for i, ser := range value.Frame.RawSeries() {
		if value.Frame.ShouldExcludeRaw(i) {
			continue
		}
		s.currValues[value.Frame.RawKeyAt(i)] = ser
	}
	keys := value.Frame.KeysSlice()
	for _, tsk := range s.stages {
		if len(lo.Intersect(tsk.ReadChannels(), keys)) > 0 {
			tsk.Next(ctx, stage.Value{
				Value:,
			})
		}
	}
	return nil
}
