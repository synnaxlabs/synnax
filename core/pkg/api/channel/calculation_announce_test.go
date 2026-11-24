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
	"encoding/json/v2"
	"fmt"
	"slices"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	freightertestutil "github.com/synnaxlabs/freighter/testutil"
	"github.com/synnaxlabs/gorp"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	apiframer "github.com/synnaxlabs/synnax/pkg/api/framer"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Calculation announcements", func() {
	type batch = actions.Scoped[channel.Key, channel.Action]
	var (
		admin    user.User
		streams  *apiframer.Service
		actionCh channel.Channel
		client   freighter.ClientStream[
			apiframer.StreamerRequest,
			apiframer.StreamerResponse,
		]
	)
	BeforeEach(func(ctx SpecContext) {
		admin = MustSucceed(rbactestutil.CreateUser(
			ctx,
			db,
			rbacSvc,
			userSvc,
			user.User{Name: "announce-check-" + uuid.New().String()},
		))
		grantOn(
			ctx,
			admin.ResourceID(),
			access.AllActions,
			resource.ID{Type: resource.TypeChannel},
			resource.ID{Type: resource.TypeCalculation},
			resource.ID{Type: resource.TypeFramer},
			resource.ID{Type: resource.TypeGroup},
		)
		streams = MustSucceed(apiframer.NewService(apiframer.ServiceConfig{
			DB:          db,
			Framer:      layer.Framer,
			Channel:     channelSvc,
			Calculation: layer.Calculation,
			RBAC:        rbacSvc,
			Records:     layer.Signals.Records(),
			Errors:      errors.NewCodec(),
		}))
		Expect(channelSvc.NewRetrieve().
			Where(channel.MatchNames("sy_channel_action_set")).
			Entry(&actionCh).
			Exec(ctx, db)).To(Succeed())
	})
	// reader creates a user that may retrieve the channel action channel and granted.
	reader := func(ctx SpecContext, granted ...resource.ID) user.User {
		GinkgoHelper()
		u := MustSucceed(rbactestutil.CreateUser(
			ctx,
			db,
			rbacSvc,
			userSvc,
			user.User{Name: "announce-check-reader-" + uuid.New().String()},
		))
		grantOn(
			ctx,
			u.ResourceID(),
			[]access.Action{access.ActionRetrieve},
			append(svcframer.ResourceIDs(channel.Keys{actionCh.Key()}), granted...)...,
		)
		return u
	}
	// open opens a stream of the channel action channel as u.
	open := func(ctx SpecContext, u user.User) {
		GinkgoHelper()
		var server apiframer.StreamerStream
		client, server = freightertestutil.NewStreams[
			apiframer.StreamerRequest,
			apiframer.StreamerResponse,
		](ctx)
		done := make(chan error, 1)
		go func() { done <- streams.Stream(AuthedCtx(ctx, u), server) }()
		DeferCleanup(func() {
			Expect(client.CloseSend()).To(Succeed())
			Eventually(done).WithTimeout(5 * time.Second).Should(Receive())
		})
		Expect(client.Send(apiframer.StreamerRequest{
			Keys: channel.Keys{actionCh.Key()},
		})).To(Succeed())
		Expect(MustSucceed(client.Receive()).Frame.Empty()).To(BeTrue())
	}
	// rename renames witness, so the stream receives a batch that marks the end of the
	// batches before it.
	rename := func(ctx SpecContext, witness channel.Channel) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).
				Rename(ctx, witness.Key(), UniqueChannelName(), false)
		})).To(Succeed())
	}
	// until returns the batches that the stream receives before the batch of witness.
	until := func(witness channel.Channel) []batch {
		GinkgoHelper()
		var out []batch
		for {
			res := MustSucceed(client.Receive())
			for _, series := range res.Frame.Get(actionCh.Key()).Series {
				for sample := range series.Samples() {
					var b batch
					Expect(json.Unmarshal(sample, &b)).To(Succeed())
					if b.Key == witness.Key() {
						return out
					}
					out = append(out, b)
				}
			}
		}
	}
	// paths returns the first segment of the path of each set in b.
	paths := func(b batch) []string {
		var out []string
		for _, a := range b.Actions {
			if a.Set != nil {
				out = append(out, a.Set.Path[0])
			}
		}
		return out
	}
	// expressions returns each expression that a set in batches holds.
	expressions := func(batches []batch) []string {
		var out []string
		for _, b := range batches {
			for _, a := range b.Actions {
				if a.Set != nil && a.Set.Path[0] == "expression" {
					out = append(out, fmt.Sprint(a.Set.Value))
				}
			}
		}
		return out
	}
	// calculatedSets returns the value of each set of calculated in batches, by
	// channel.
	calculatedSets := func(batches []batch) map[channel.Key][]any {
		out := map[channel.Key][]any{}
		for _, b := range batches {
			for _, a := range b.Actions {
				if a.Set != nil && a.Set.Path[0] == "calculated" {
					out[b.Key] = append(out[b.Key], a.Set.Value)
				}
			}
		}
		return out
	}
	createOld := func(ctx SpecContext, ch apichannel.Channel) apichannel.Channel {
		GinkgoHelper()
		return MustSucceed(createChannels(
			AuthedCtx(ctx, admin),
			apichannel.CreateRequest{Channels: []apichannel.Channel{ch}},
		)).Channels[0]
	}
	setOf := func(path string, value any) channel.Action {
		return channel.NewSetAction(channel.SetPayload{
			Path:  []string{path},
			Value: value,
		})
	}
	times := func(ch channel.Channel, n int) string {
		return fmt.Sprintf("return %s * %d", ch.Name, n)
	}
	calcOf := func(ctx SpecContext, output channel.Key) calculation.Calculation {
		GinkgoHelper()
		var c calculation.Calculation
		Expect(layer.Calculation.NewRetrieve().
			Where(calculation.MatchOutputs(output)).
			Entry(&c).
			Exec(ctx, db)).To(Succeed())
		return c
	}

	It(
		"Should send no hidden expression from a commit with two calculation creates",
		func(ctx SpecContext) {
			visible := createChannel(ctx, telem.Float32T)
			secret := createChannel(ctx, telem.Float32T)
			open1 := createChannel(ctx, telem.Float32T)
			open2 := createChannel(ctx, telem.Float32T)
			witness := createChannel(ctx, telem.Float32T)
			s := reader(
				ctx,
				resource.ID{Type: resource.TypeCalculation},
				channel.ResourceID(visible.Key()),
				channel.ResourceID(open1.Key()),
				channel.ResourceID(open2.Key()),
				channel.ResourceID(witness.Key()),
			)
			open(ctx, s)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				w := layer.Calculation.NewWriter(tx)
				hiddenCalc := calculation.Calculation{
					Output:     open2.Key(),
					Expression: times(secret, 3),
				}
				shownCalc := calculation.Calculation{
					Output:     open1.Key(),
					Expression: times(visible, 2),
				}
				if err := w.Create(ctx, &hiddenCalc); err != nil {
					return err
				}
				if err := w.Create(ctx, &shownCalc); err != nil {
					return err
				}
				return channelSvc.NewWriter(tx).
					Rename(ctx, open2.Key(), UniqueChannelName(), false)
			})).To(Succeed())
			rename(ctx, witness)
			received := until(witness)
			Expect(expressions(received)).To(Equal([]string{times(visible, 2)}))
			Expect(received).To(ContainElement(SatisfyAll(
				HaveField("Key", open2.Key()),
				HaveField("Actions", ContainElement(HaveField("Rename", Not(BeNil())))),
			)))
		},
		SpecTimeout(60*time.Second),
	)

	It(
		"Should send only the calculated flag of a calculation that one commit "+
			"creates and deletes to a reader who cannot read it",
		func(ctx SpecContext) {
			secret := createChannel(ctx, telem.Float32T)
			output := createChannel(ctx, telem.Float32T)
			witness := createChannel(ctx, telem.Float32T)
			s := reader(
				ctx,
				resource.ID{Type: resource.TypeCalculation},
				channel.ResourceID(output.Key()),
				channel.ResourceID(witness.Key()),
			)
			open(ctx, s)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				w := layer.Calculation.NewWriter(tx)
				c := calculation.Calculation{
					Output:     output.Key(),
					Expression: times(secret, 2),
				}
				if err := w.Create(ctx, &c); err != nil {
					return err
				}
				return w.Delete(ctx, c.Key)
			})).To(Succeed())
			rename(ctx, witness)
			received := until(witness)
			Expect(expressions(received)).To(BeEmpty())
			Expect(calculatedSets(received)).To(Equal(map[channel.Key][]any{
				output.Key(): {true, false},
			}))
		},
		SpecTimeout(60*time.Second),
	)

	It(
		"Should clear the calculated flag of every output and the fields only of "+
			"the calculations a reader could read at a delete of several",
		func(ctx SpecContext) {
			visible := createChannel(ctx, telem.Float32T)
			secret := createChannel(ctx, telem.Float32T)
			witness := createChannel(ctx, telem.Float32T)
			shownOut := createOld(ctx, apichannel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Float32T,
				Virtual:    true,
				Expression: times(visible, 2),
			})
			hiddenOut := createOld(ctx, apichannel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Float32T,
				Virtual:    true,
				Expression: times(secret, 2),
			})
			shown, hidden := calcOf(ctx, shownOut.Key), calcOf(ctx, hiddenOut.Key)
			s := reader(
				ctx,
				resource.ID{Type: resource.TypeCalculation},
				channel.ResourceID(visible.Key()),
				channel.ResourceID(shownOut.Key),
				channel.ResourceID(hiddenOut.Key),
				channel.ResourceID(witness.Key()),
			)
			open(ctx, s)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Calculation.NewWriter(tx).
					Delete(ctx, hidden.Key, uuid.New(), shown.Key, shown.Key)
			})).To(Succeed())
			rename(ctx, witness)
			received := until(witness)
			Expect(calculatedSets(received)).To(Equal(map[channel.Key][]any{
				shownOut.Key:  {false},
				hiddenOut.Key: {false},
			}))
			Expect(expressions(received)).To(Equal([]string{""}))
			Expect(slices.ContainsFunc(received, func(b batch) bool {
				return b.Key == hiddenOut.Key && slices.Contains(paths(b), "expression")
			})).To(BeFalse())
		},
		SpecTimeout(60*time.Second),
	)

	It(
		"Should send a reader the group set and no expression of an old API edit "+
			"that makes the calculation read a hidden channel",
		func(ctx SpecContext) {
			visible := createChannel(ctx, telem.Float32T)
			secret := createChannel(ctx, telem.Float32T)
			witness := createChannel(ctx, telem.Float32T)
			out := createOld(ctx, apichannel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Float32T,
				Virtual:    true,
				Expression: times(visible, 2),
			})
			g := createChannelGroup(ctx, "announce-check-"+uuid.New().String())
			s := reader(
				ctx,
				resource.ID{Type: resource.TypeCalculation},
				channel.ResourceID(visible.Key()),
				channel.ResourceID(out.Key),
				channel.ResourceID(witness.Key()),
			)
			open(ctx, s)
			Expect(dispatch(AuthedCtx(ctx, admin), apichannel.DispatchRequest{
				Key:         out.Key,
				DispatchKey: "announce-check",
				Actions: []channel.Action{
					setOf("expression", times(secret, 4)),
					setOf("group", g.Key.String()),
				},
			})).Error().To(Succeed())
			rename(ctx, witness)
			received := until(witness)
			Expect(expressions(received)).To(BeEmpty())
			Expect(slices.ContainsFunc(received, func(b batch) bool {
				return b.Key == out.Key && slices.Contains(paths(b), "group")
			})).To(BeTrue())
		},
		SpecTimeout(60*time.Second),
	)

	Describe("A create that retypes an existing output", func() {
		// framesOf returns the batches of key in each frame that the stream receives
		// before the batch of witness.
		framesOf := func(key channel.Key, witness channel.Channel) [][]batch {
			GinkgoHelper()
			var out [][]batch
			for {
				res := MustSucceed(client.Receive())
				var (
					frame []batch
					done  bool
				)
				for _, series := range res.Frame.Get(actionCh.Key()).Series {
					for sample := range series.Samples() {
						var b batch
						Expect(json.Unmarshal(sample, &b)).To(Succeed())
						switch b.Key {
						case witness.Key():
							done = true
						case key:
							frame = append(frame, b)
						}
					}
				}
				if len(frame) > 0 {
					out = append(out, frame)
				}
				if done {
					return out
				}
			}
		}
		// retype streams as a reader of granted and of the output, then creates a
		// calculation that writes the output with a wider data type than it holds. It
		// returns the paths of each batch of the output, by frame.
		retype := func(ctx SpecContext, granted ...resource.ID) [][][]string {
			GinkgoHelper()
			input := createChannel(ctx, telem.Float64T)
			output := createChannel(ctx, telem.Float32T)
			witness := createChannel(ctx, telem.Float32T)
			open(ctx, reader(ctx, append(
				granted,
				channel.ResourceID(input.Key()),
				channel.ResourceID(output.Key()),
				channel.ResourceID(witness.Key()),
			)...))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				c := calculation.Calculation{
					Output:     output.Key(),
					Expression: times(input, 2),
				}
				return layer.Calculation.NewWriter(tx).Create(ctx, &c)
			})).To(Succeed())
			rename(ctx, witness)
			frames := framesOf(output.Key(), witness)
			out := make([][][]string, 0, len(frames))
			for _, frame := range frames {
				batches := make([][]string, 0, len(frame))
				for _, b := range frame {
					batches = append(batches, paths(b))
				}
				out = append(out, batches)
			}
			return out
		}

		It(
			"Should send a reader who cannot read the calculation only the data type "+
				"and the calculated flag",
			func(ctx SpecContext) {
				Expect(retype(ctx)).
					To(Equal([][][]string{{{"data_type"}, {"calculated"}}}))
			},
			SpecTimeout(60*time.Second),
		)

		It(
			"Should send a calculation reader the data type, the calculated flag, "+
				"then the calculation fields, in one frame",
			func(ctx SpecContext) {
				Expect(retype(ctx, resource.ID{Type: resource.TypeCalculation})).
					To(Equal([][][]string{
						{{"data_type"}, {"calculated"}, {"expression", "operations"}},
					}))
			},
			SpecTimeout(60*time.Second),
		)
	})
})
