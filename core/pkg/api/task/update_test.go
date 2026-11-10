// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	apitask "github.com/synnaxlabs/synnax/pkg/api/task"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/modbus"
	"github.com/synnaxlabs/synnax/pkg/service/ni"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/patch"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Dispatch update", func() {
	var u user.User
	BeforeEach(func(ctx SpecContext) { u = createUser(ctx) })
	record := func() *Recorder[task.Key, task.Action] {
		rec := &Recorder[task.Key, task.Action]{}
		DeferCleanup(svc.Task.OnAction(rec.Record))
		return rec
	}
	createWith := func(ctx SpecContext, typ string, cfg any) task.Task {
		GinkgoHelper()
		t := task.Task{
			Type:   typ,
			Rack:   svc.Rack.EmbeddedKey,
			Name:   "Read",
			Config: MustSucceed(msgpack.NewEncodedJSON(cfg)),
		}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Task.NewWriter(tx).Create(ctx, &t)
		})).To(Succeed())
		grant(ctx, access.ActionUpdate, u.ResourceID(), t.ResourceID())
		grant(ctx, access.ActionRetrieve, u.ResourceID(), t.ResourceID())
		return t
	}
	voltage := func(key string, port uint32) ni.AIChannel {
		var v ni.AIVoltageChannel
		v.ApplyDefaults()
		v.Key, v.Name, v.Port, v.MinVal = key, "Pressure", port, -10
		return ni.AIChannel{Variant: v}
	}
	createAnalogRead := func(ctx SpecContext) task.Task {
		GinkgoHelper()
		cfg := ni.AnalogReadConfig{
			Channels: []ni.AIChannel{voltage("c3", 1), voltage("c4", 2)},
		}
		cfg.ApplyDefaults()
		return createWith(ctx, svc.NI.AnalogRead.Type(), cfg)
	}
	updateConfig := func(
		ctx SpecContext,
		t task.Task,
		cfg msgpack.EncodedJSON,
	) (apitask.DispatchResponse, error) {
		return dispatch(AuthedCtx(ctx, u), apitask.DispatchRequest{
			Key:         t.Key,
			DispatchKey: "dk",
			Actions: []task.Action{task.NewUpdateAction(task.UpdatePayload{
				Value: task.Update{Config: cfg},
			})},
		})
	}
	channels := func(ops map[string]any) msgpack.EncodedJSON {
		return msgpack.EncodedJSON{"channels": ops}
	}
	analogRead := func(ctx SpecContext, t task.Task) ni.AnalogReadConfig {
		GinkgoHelper()
		var cfg ni.AnalogReadConfig
		Expect(retrieveTask(ctx, t.Key).Config.Unmarshal(&cfg)).To(Succeed())
		return cfg
	}
	keys := func(cfg ni.AnalogReadConfig) []string {
		return lo.Map(cfg.Channels, func(ch ni.AIChannel, _ int) string {
			return ch.Variant.(ni.AIVoltageChannel).Key
		})
	}

	It("Should merge an update into a channel of the same variant", func(
		ctx SpecContext,
	) {
		t := createAnalogRead(ctx)
		MustSucceed(updateConfig(ctx, t, channels(map[string]any{
			"update": []any{
				map[string]any{"key": "c3", "type": "ai_voltage", "max_val": 5},
			},
		})))
		c3 := analogRead(ctx, t).Channels[0].Variant.(ni.AIVoltageChannel)
		Expect(c3.MaxVal).To(Equal(5.0))
		Expect(c3.MinVal).To(Equal(-10.0))
		Expect(c3.Port).To(Equal(uint32(1)))
	})

	It("Should insert a channel at the end", func(ctx SpecContext) {
		t := createAnalogRead(ctx)
		MustSucceed(updateConfig(ctx, t, channels(map[string]any{
			"insert": []any{MustSucceed(msgpack.NewEncodedJSON(voltage("c5", 3)))},
		})))
		Expect(keys(analogRead(ctx, t))).To(Equal([]string{"c3", "c4", "c5"}))
	})

	It("Should remove a channel", func(ctx SpecContext) {
		t := createAnalogRead(ctx)
		MustSucceed(updateConfig(ctx, t, channels(map[string]any{
			"remove": []any{"c3"},
		})))
		Expect(keys(analogRead(ctx, t))).To(Equal([]string{"c4"}))
	})

	It("Should move a channel after another and first", func(ctx SpecContext) {
		t := createAnalogRead(ctx)
		MustSucceed(updateConfig(ctx, t, channels(map[string]any{
			"insert": []any{MustSucceed(msgpack.NewEncodedJSON(voltage("c5", 3)))},
			"move": []any{
				map[string]any{"key": "c3", "after": "c4"},
				map[string]any{"key": "c5", "after": nil},
			},
		})))
		Expect(keys(analogRead(ctx, t))).To(Equal([]string{"c5", "c4", "c3"}))
	})

	It("Should change the variant and fill the fields of the new one", func(
		ctx SpecContext,
	) {
		t := createAnalogRead(ctx)
		MustSucceed(updateConfig(ctx, t, channels(map[string]any{
			"update": []any{map[string]any{"key": "c3", "type": "ai_current"}},
		})))
		c3 := analogRead(ctx, t).Channels[0].Variant.(ni.AICurrentChannel)
		Expect(c3.Key).To(Equal("c3"))
		Expect(c3.Name).To(Equal("Pressure"))
		Expect(c3.ExtShuntResistorVal).To(Equal(1.0))
		Expect(c3.MinVal).To(Equal(0.0))
	})

	It("Should change the variant of a nested custom scale", func(ctx SpecContext) {
		t := createAnalogRead(ctx)
		MustSucceed(updateConfig(ctx, t, channels(map[string]any{
			"update": []any{map[string]any{
				"key":          "c3",
				"type":         "ai_voltage",
				"custom_scale": map[string]any{"type": "linear", "y_intercept": 2},
			}},
		})))
		c3 := analogRead(ctx, t).Channels[0].Variant.(ni.AIVoltageChannel)
		linear := c3.CustomScale.CustomScale.Variant.(ni.LinearScale)
		Expect(linear.YIntercept).To(Equal(2.0))
		Expect(linear.Slope).To(Equal(1.0))
	})

	It("Should name the channel and the field that a variant change needs", func(
		ctx SpecContext,
	) {
		t := createWith(ctx, svc.Modbus.Read.Type(), modbus.ReadConfig{
			Channels: []modbus.ReadChannel{{Variant: modbus.CoilReadChannel{
				BaseReadChannel: modbus.BaseReadChannel{Key: "c1", Name: "Valve"},
			}}},
		})
		Expect(updateConfig(ctx, t, channels(map[string]any{
			"update": []any{
				map[string]any{"key": "c1", "type": "holding_register_string"},
			},
		}))).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("c1: length is required")),
		))
		Expect(retrieveTask(ctx, t.Key).ConfigHash).To(Equal(t.ConfigHash))
	})

	It("Should return a stale error for a channel that does not exist", func(
		ctx SpecContext,
	) {
		t := createAnalogRead(ctx)
		rec := record()
		Expect(updateConfig(ctx, t, channels(map[string]any{
			"update": []any{
				map[string]any{"key": "c9", "type": "ai_voltage", "max_val": 5},
			},
		}))).Error().To(SatisfyAll(
			MatchError(patch.ErrStale),
			MatchError(ContainSubstring("c9 not found")),
		))
		Expect(keys(analogRead(ctx, t))).To(Equal([]string{"c3", "c4"}))
		Expect(rec.Snapshot()).To(BeEmpty())
	})

	It("Should reply with the config hash and broadcast the reply", func(
		ctx SpecContext,
	) {
		t := createAnalogRead(ctx)
		rec := record()
		res := MustSucceed(updateConfig(ctx, t, channels(map[string]any{
			"remove": []any{"c4"},
		})))
		hash := retrieveTask(ctx, t.Key).ConfigHash
		Expect(hash).ToNot(Equal(t.ConfigHash))
		Expect(res.Actions).To(Equal([]task.Action{
			task.NewUpdateAction(task.UpdatePayload{
				Value: task.Update{Config: channels(map[string]any{
					"remove": []any{"c4"},
				})},
			}),
			task.NewUpdateAction(task.UpdatePayload{
				Value: task.Update{ConfigHash: &hash},
			}),
		}))
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		Expect(frames[0].DispatchKey).To(Equal("dk"))
		Expect(frames[0].Actions).To(Equal(res.Actions))
	})

	It("Should keep an integer above 2^53 exact through the JSON transport", func(
		ctx SpecContext,
	) {
		t := createWith(ctx, svc.HTTP.Read.Type(), map[string]any{
			"endpoints": []any{map[string]any{
				"key":    "e1",
				"method": "GET",
				"path":   "/",
				"fields": []any{map[string]any{
					"key":       "f1",
					"pointer":   "/v",
					"data_type": "int64",
					"optional":  true,
				}},
			}},
		})
		rec := record()
		body := fmt.Sprintf(`{"key": %q, "dispatch_key": "dk", "actions": [{
			"type": "update",
			"update": {"value": {"config": {"endpoints": {"update": [{
				"key": "e1",
				"fields": {"update": [{"key": "f1", "fill_value": 9007199254740993}]}
			}]}}}}
		}]}`, t.Key)
		var req apitask.DispatchRequest
		Expect(json.Codec.Decode(ctx, []byte(body), &req)).To(Succeed())
		res := MustSucceed(dispatch(AuthedCtx(ctx, u), req))
		const exact = `"fill_value":9007199254740993`
		Expect(string(MustSucceed(json.Codec.Encode(ctx, res)))).
			To(ContainSubstring(exact))
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		Expect(string(MustSucceed(json.Codec.Encode(ctx, frames[0].Actions)))).
			To(ContainSubstring(exact))
	})

	It("Should reply with the stored config when storing fills a default", func(
		ctx SpecContext,
	) {
		t := createAnalogRead(ctx)
		sparse := channels(map[string]any{
			"insert": []any{map[string]any{"key": "c5", "type": "ai_voltage"}},
		})
		res := MustSucceed(updateConfig(ctx, t, sparse))
		stored := retrieveTask(ctx, t.Key)
		c5 := analogRead(ctx, t).Channels[2].Variant.(ni.AIVoltageChannel)
		Expect(c5.TerminalConfig).To(Equal(ni.TerminalConfigCfgDefault))
		Expect(res.Actions).To(Equal([]task.Action{
			task.NewUpdateAction(
				task.UpdatePayload{Value: task.Update{Config: sparse}},
			),
			task.NewUpdateAction(task.UpdatePayload{
				Value: task.Update{Config: msgpack.EncodedJSON{"set": stored.Config}},
			}),
			task.NewUpdateAction(task.UpdatePayload{
				Value: task.Update{ConfigHash: &stored.ConfigHash},
			}),
		}))
	})
})
