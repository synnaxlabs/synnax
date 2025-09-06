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
	"fmt"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/compiler/runtime/mock"
	"github.com/synnaxlabs/arc/module"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"go.uber.org/zap"
)

type Runtime struct {
	wasm wazero.Runtime
}

type Config struct {
	Module  *module.Module
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

	if err := mockRuntime.Bind(ctx, r.wasm); err != nil {
		return nil, err
	}

	mod, err := r.wasm.Instantiate(ctx, cfg.Module.Wasm)
	if err != nil {
		return nil, err
	}
	collectedReadChannelNames := make(set.Set[string])
	for _, task := range cfg.Module.Tasks {
		collectedReadChannelNames.Add(task.Channels.Read...)
	}
	channels := make([]channel.Channel, 0, len(collectedReadChannelNames))
	if err := cfg.Channel.NewRetrieve().
		WhereNames(collectedReadChannelNames.Keys()...).
		Entries(&channels).
		Exec(ctx, nil); err != nil {
		return nil, err
	}
	readChannelKeys := channel.KeysFromChannels(channels)
	channelNameMap := channel.NameMap(channels)
	streamer, err := cfg.Framer.NewStreamer(
		ctx,
		framer.StreamerConfig{Keys: readChannelKeys},
	)
	if err != nil {
		return nil, err
	}

	tasks := make(map[string]Task)

	createOnOutput := func(k string) func(ctx context.Context) {
		return func(ctx context.Context) {
			for _, edge := range cfg.Module.Edges {
				if edge.Source.Node == k {
					t := tasks[edge.Target.Node]
					t.Input() <- struct{}{}
				}
			}
		}
	}

	getReadChannels := func(t module.Task) channel.Keys {
		taskChannels := set.FromSlice[string](t.Channels.Read)
		return lo.FilterMapToSlice(channelNameMap, func(name string, key channel.Key) (channel.Key, bool) {
			return key, taskChannels.Contains(name)
		})
	}

	for _, node := range cfg.Module.Nodes {

		if strings.Contains(node.Type, "__expr") {
			taskType, _ := lo.Find(cfg.Module.Tasks, func(item module.Task) bool {
				return node.Type == item.Key
			})
			t := &reactiveExpression{}
			t.input = make(chan struct{}, 10)
			t.waFunc = mod.ExportedFunction(taskType.Key)
			t.readChannels = getReadChannels(taskType)
			t.baseTask.task = &taskType
			t.OnOutput(createOnOutput(node.Key))
			tasks[node.Key] = t
		} else if node.Type == "print" {
			t := &printTask{}
			t.input = make(chan struct{}, 10)
			t.node = &node
			t.OnOutput(createOnOutput(node.Key))
			tasks[node.Key] = t
		}
	}
	p := plumber.New()
	plumber.SetSegment(p, "streamer", streamer)
	streamProcessor := &streamProcessor{
		Module:     cfg.Module,
		tasks:      tasks,
		currValues: currValues,
	}
	streamProcessor.Sink = streamProcessor.sink
	plumber.SetSink[framer.StreamerResponse](p, "stream_processor", streamProcessor)
	streamer.InFrom(confluence.NewStream[framer.StreamerRequest]())
	plumber.MustConnect[framer.StreamerResponse](p, "streamer", "stream_processor", 10)
	sCtx, _ := signal.Isolated()
	p.Flow(sCtx)
	for _, task := range tasks {
		task.Flow(sCtx)
	}
	return r, err
}

type Task interface {
	ReadChannels() channel.Keys
	Input() chan<- struct{}
	OnOutput(func(ctx context.Context))
	Flow(ctx signal.Context)
}

type streamProcessor struct {
	Module *module.Module
	confluence.UnarySink[framer.StreamerResponse]
	currValues     map[channel.Key]telem.Series
	tasks          map[string]Task
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
	for _, tsk := range s.tasks {
		if len(lo.Intersect(tsk.ReadChannels(), keys)) > 0 {
			tsk.Input() <- struct{}{}
		}
	}
	return nil
}

type baseTask struct {
	input        chan struct{}
	readChannels channel.Keys
	onOutput     func(ctx context.Context)
	task         *module.Task
	node         *module.Node
}

func (b baseTask) ReadChannels() channel.Keys {
	return b.readChannels
}

func (b baseTask) Input() chan<- struct{} { return b.input }

func (b *baseTask) OnOutput(callback func(ctx context.Context)) {
	b.onOutput = callback
}

type reactiveExpression struct {
	baseTask
	waFunc    api.Function
	prevValue uint64
}

func (r reactiveExpression) Flow(ctx signal.Context) {
	signal.GoRange(ctx, r.input, func(ctx context.Context, _ struct{}) error {
		res, err := r.waFunc.Call(ctx)
		if err != nil {
			zap.S().Error(err)
		}
		resValue := res[0]
		if resValue != r.prevValue {
			r.prevValue = resValue
			r.onOutput(ctx)
		}
		return nil
	})
}

type printTask struct {
	baseTask
}

func (p printTask) Flow(ctx signal.Context) {
	signal.GoRange(ctx, p.input, func(ctx context.Context, _ struct{}) error {
		fmt.Println(p.node.Config["message"])
		p.onOutput(ctx)
		return nil
	})
}
