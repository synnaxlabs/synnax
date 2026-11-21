// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	apichannelpb "github.com/synnaxlabs/synnax/pkg/api/channel/pb"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	xjson "github.com/synnaxlabs/x/encoding/json"
	xmsgpack "github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/http"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/protobuf/proto"
)

// v062Operation is an operation as a core/v0.62.0 client sends and reads it.
type v062Operation struct {
	Type         string         `json:"type"          msgpack:"type"`
	ResetChannel uint32         `json:"reset_channel" msgpack:"reset_channel"`
	Duration     telem.TimeSpan `json:"duration"      msgpack:"duration"`
}

// v062Channel is the part of a channel that a core/v0.62.0 client reads back.
type v062Channel struct {
	Key        uint32          `json:"key"        msgpack:"key"`
	Expression string          `json:"expression" msgpack:"expression"`
	Operations []v062Operation `json:"operations" msgpack:"operations"`
}

var _ = Describe("Old API breaker a4", func() {
	var u user.User
	BeforeEach(func(ctx SpecContext) {
		u = MustSucceed(rbactestutil.CreateUser(
			ctx,
			db,
			rbacSvc,
			userSvc,
			user.User{Name: "breaker-a4-" + uuid.New().String()},
		))
		grantOn(
			ctx,
			u.ResourceID(),
			access.AllActions,
			resource.ID{Type: resource.TypeChannel},
			resource.ID{Type: resource.TypeCalculation},
			resource.ID{Type: resource.TypeFramer},
			resource.ID{Type: resource.TypeGroup},
		)
	})
	calculated := func(input channel.Channel) apichannel.Channel {
		return apichannel.Channel{
			Name:       UniqueChannelName(),
			DataType:   telem.Float32T,
			Virtual:    true,
			Expression: fmt.Sprintf("return %s * 2", input.Name),
		}
	}
	retrieveOne := func(ctx SpecContext, key channel.Key) apichannel.Channel {
		GinkgoHelper()
		res := MustSucceed(retrieveChannels(
			AuthedCtx(ctx, u),
			apichannel.RetrieveRequest{
				Where: &channel.Where{Key: &filter.Set[channel.Key]{Eq: &key}},
			},
		))
		Expect(res.Channels).To(HaveLen(1))
		return res.Channels[0]
	}

	Describe("Wire round trip", func() {
		// roundTrip sends a create of a calculated channel with ops as a v0.62 client
		// encodes it through codec, then reads the channel back through codec as a
		// v0.62 client decodes it. v0.62 stored the operations as sent and returned
		// them as stored.
		roundTrip := func(
			ctx SpecContext,
			codec http.Codec,
			ops []v062Operation,
		) []v062Operation {
			GinkgoHelper()
			input := createChannel(ctx, telem.Float32T)
			body := MustSucceed(codec.Encode(map[string]any{
				"channels": []any{map[string]any{
					"name":       UniqueChannelName(),
					"data_type":  "float32",
					"virtual":    true,
					"expression": fmt.Sprintf("return %s * 2", input.Name),
					"operations": ops,
				}},
			}))
			var req apichannel.CreateRequest
			Expect(codec.Decode(body, &req)).To(Succeed())
			created := MustSucceed(createChannels(AuthedCtx(ctx, u), req)).Channels[0]
			res := MustSucceed(retrieveChannels(
				AuthedCtx(ctx, u),
				apichannel.RetrieveRequest{
					Where: &channel.Where{
						Key: &filter.Set[channel.Key]{Eq: &created.Key},
					},
				},
			))
			var read struct {
				Channels []v062Channel `json:"channels" msgpack:"channels"`
			}
			Expect(codec.Decode(MustSucceed(codec.Encode(res)), &read)).To(Succeed())
			Expect(read.Channels).To(HaveLen(1))
			return read.Channels[0].Operations
		}
		codecs := map[string]http.Codec{
			"JSON":    xjson.Codec,
			"msgpack": xmsgpack.Codec,
		}
		type opsOf func(reset channel.Key) []v062Operation
		cases := map[string]opsOf{
			"min with a duration": func(channel.Key) []v062Operation {
				return []v062Operation{{Type: "min", Duration: 5 * telem.Second}}
			},
			"max with a reset": func(reset channel.Key) []v062Operation {
				return []v062Operation{{Type: "max", ResetChannel: uint32(reset)}}
			},
			"avg with a duration and a reset": func(reset channel.Key) []v062Operation {
				return []v062Operation{{
					Type:         "avg",
					ResetChannel: uint32(reset),
					Duration:     telem.Second,
				}}
			},
			"derivative": func(channel.Key) []v062Operation {
				return []v062Operation{{Type: "derivative"}}
			},
			"none": func(channel.Key) []v062Operation {
				return []v062Operation{{Type: "none"}}
			},
			"derivative with a duration": func(channel.Key) []v062Operation {
				return []v062Operation{{
					Type:     "derivative",
					Duration: 5 * telem.Second,
				}}
			},
			"none with the window and reset the Console form kept": func(
				reset channel.Key,
			) []v062Operation {
				return []v062Operation{{
					Type:         "none",
					ResetChannel: uint32(reset),
					Duration:     5 * telem.Second,
				}}
			},
			"none before avg": func(channel.Key) []v062Operation {
				return []v062Operation{
					{Type: "none"},
					{Type: "avg", Duration: telem.Second},
				}
			},
		}
		for name, ops := range cases {
			for codecName, codec := range codecs {
				It(
					fmt.Sprintf(
						"Should read back %s as v0.62 did in %s",
						name,
						codecName,
					),
					func(ctx SpecContext) {
						reset := createChannel(ctx, telem.BooleanT)
						sent := ops(reset.Key())
						Expect(roundTrip(ctx, codec, sent)).To(Equal(sent))
					},
				)
			}
		}

		It("Should translate each operation type through proto and back", func(
			ctx SpecContext,
		) {
			input := createChannel(ctx, telem.Float32T)
			reset := createChannel(ctx, telem.BooleanT)
			for _, ops := range [][]apichannel.Operation{
				{{Type: apichannel.OperationTypeMin, Duration: telem.Second}},
				{{Type: apichannel.OperationTypeMax, ResetChannel: reset.Key()}},
				{{
					Type:         apichannel.OperationTypeAvg,
					ResetChannel: reset.Key(),
					Duration:     telem.Second,
				}},
				{{Type: apichannel.OperationTypeDerivative}},
				{{Type: apichannel.OperationTypeNone}},
			} {
				req := calculated(input)
				req.Operations = ops
				created := MustSucceed(createChannels(
					AuthedCtx(ctx, u),
					apichannel.CreateRequest{Channels: []apichannel.Channel{req}},
				)).Channels[0]
				read := retrieveOne(ctx, created.Key)
				pb := MustSucceed(apichannelpb.ChannelToPB(read))
				var decoded apichannelpb.Channel
				Expect(proto.Unmarshal(MustSucceed(proto.Marshal(pb)), &decoded)).
					To(Succeed())
				back := MustSucceed(apichannelpb.ChannelFromPB(&decoded))
				Expect(back.Operations).To(Equal(ops))
			}
		})
	})

	Describe("Dispatch", func() {
		dispatchTo := func(
			ctx SpecContext,
			key channel.Key,
			acts ...channel.Action,
		) error {
			_, err := dispatch(AuthedCtx(ctx, u), apichannel.DispatchRequest{
				Key:     key,
				Actions: acts,
			})
			return err
		}
		set := func(value any, path ...string) channel.Action {
			return channel.NewSetAction(channel.SetPayload{Path: path, Value: value})
		}

		It("Should set operations that a msgpack client sends as plain maps", func(
			ctx SpecContext,
		) {
			input := createChannel(ctx, telem.Float32T)
			created := MustSucceed(createChannels(
				AuthedCtx(ctx, u),
				apichannel.CreateRequest{
					Channels: []apichannel.Channel{calculated(input)},
				},
			)).Channels[0]
			body := MustSucceed(xmsgpack.Codec.Encode(map[string]any{
				"key": created.Key,
				"actions": []any{map[string]any{
					"type": "set",
					"set": map[string]any{
						"path": []string{"operations"},
						"value": []any{map[string]any{
							"type":          "avg",
							"reset_channel": 0,
							"duration":      int64(telem.Second),
						}},
					},
				}},
			}))
			var req apichannel.DispatchRequest
			Expect(xmsgpack.Codec.Decode(body, &req)).To(Succeed())
			MustSucceed(dispatch(AuthedCtx(ctx, u), req))
			Expect(retrieveOne(ctx, created.Key).Operations).To(Equal(
				[]apichannel.Operation{{
					Type:     apichannel.OperationTypeAvg,
					Duration: telem.Second,
				}},
			))
		})

		It("Should set the expression and the operations in one dispatch", func(
			ctx SpecContext,
		) {
			input := createChannel(ctx, telem.Float32T)
			created := MustSucceed(createChannels(
				AuthedCtx(ctx, u),
				apichannel.CreateRequest{
					Channels: []apichannel.Channel{calculated(input)},
				},
			)).Channels[0]
			expression := fmt.Sprintf("return %s * 5", input.Name)
			ops := []apichannel.Operation{
				{Type: apichannel.OperationTypeMin, Duration: telem.Second},
			}
			Expect(dispatchTo(
				ctx,
				created.Key,
				set(expression, "expression"),
				set(ops, "operations"),
				channel.NewRenameAction(
					channel.RenamePayload{Name: UniqueChannelName()},
				),
			)).To(Succeed())
			read := retrieveOne(ctx, created.Key)
			Expect(read.Expression).To(Equal(expression))
			Expect(read.Operations).To(Equal(ops))
		})
	})

	Describe("Batch positions", func() {
		// pathOf returns the path of the error of a create of channels.
		pathOf := func(ctx SpecContext, channels []apichannel.Channel) []string {
			GinkgoHelper()
			_, err := createChannels(
				AuthedCtx(ctx, u),
				apichannel.CreateRequest{Channels: channels},
			)
			pathErr, ok := errors.AsType[validate.PathError](err)
			Expect(ok).To(BeTrue(), "%v is a path error", err)
			return pathErr.Path
		}
		// batch returns four channels, where bit i of calcMask makes channel i
		// calculated.
		batch := func(input channel.Channel, calcMask int) []apichannel.Channel {
			out := make([]apichannel.Channel, 4)
			for i := range out {
				if calcMask&(1<<i) != 0 {
					out[i] = calculated(input)
					continue
				}
				out[i] = apichannel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.Float32T,
					Virtual:  true,
				}
			}
			return out
		}

		It("Should give a bad name at each position of each mixed batch its position",
			func(ctx SpecContext) {
				input := createChannel(ctx, telem.Float32T)
				taken := createChannel(ctx, telem.Float32T)
				for calcMask := range 16 {
					for at := range 4 {
						for _, name := range []string{"", "bad name", taken.Name} {
							channels := batch(input, calcMask)
							channels[at].Name = name
							Expect(pathOf(ctx, channels)).To(
								Equal([]string{fmt.Sprint(at), "name"}),
								"mask %04b, position %d, name %q",
								calcMask,
								at,
								name,
							)
						}
					}
				}
			},
		)

		It("Should give a name repeated in one request the position v0.62 gave it",
			func(ctx SpecContext) {
				input := createChannel(ctx, telem.Float32T)
				for calcMask := range 16 {
					for first := range 4 {
						for second := first + 1; second < 4; second++ {
							channels := batch(input, calcMask)
							channels[second].Name = channels[first].Name
							_, err := createChannels(
								AuthedCtx(ctx, u),
								apichannel.CreateRequest{Channels: channels},
							)
							Expect(pathOf(ctx, channels)[0]).To(
								Equal(fmt.Sprint(second)),
								"mask %04b, name of %d repeated at %d: %v",
								calcMask,
								first,
								second,
								err,
							)
						}
					}
				}
			},
		)

		It("Should give a failed index of a calculated channel its channel's position",
			func(ctx SpecContext) {
				input := createChannel(ctx, telem.Float32T)
				for calcAt := range 4 {
					channels := batch(input, 1<<calcAt)
					createNamedChannel(
						ctx,
						channels[calcAt].Name+"_time",
						telem.Float32T,
					)
					Expect(pathOf(ctx, channels)[0]).To(Equal(fmt.Sprint(calcAt)))
				}
			},
		)

		It("Should give a group error at each position its position", func(
			ctx SpecContext,
		) {
			input := createChannel(ctx, telem.Float32T)
			missing := uuid.New()
			for calcMask := range 16 {
				for at := range 4 {
					channels := batch(input, calcMask)
					channels[at].Group = &missing
					Expect(pathOf(ctx, channels)).To(
						Equal([]string{fmt.Sprint(at), "group"}),
						"mask %04b, position %d",
						calcMask,
						at,
					)
				}
			}
		})
	})
})
