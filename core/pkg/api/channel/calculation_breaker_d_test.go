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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Old API breaker d", func() {
	var u user.User
	BeforeEach(func(ctx SpecContext) {
		u = MustSucceed(rbactestutil.CreateUser(
			ctx,
			db,
			rbacSvc,
			userSvc,
			user.User{Name: "breaker-d-" + uuid.New().String()},
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
	create := func(ctx SpecContext, ch apichannel.Channel) apichannel.Channel {
		GinkgoHelper()
		return MustSucceed(createChannels(
			AuthedCtx(ctx, u),
			apichannel.CreateRequest{Channels: []apichannel.Channel{ch}},
		)).Channels[0]
	}
	set := func(path string, value any) channel.Action {
		return channel.NewSetAction(
			channel.SetPayload{Path: []string{path}, Value: value},
		)
	}
	exists := func(ctx SpecContext, key channel.Key) bool {
		GinkgoHelper()
		return MustSucceed(channelSvc.NewRetrieve().
			Where(channel.MatchKeys(key)).
			Exists(ctx, db))
	}

	Describe("Index of a calculated channel", func() {
		It("Should delete the index when the old API deletes the channel", func(
			ctx SpecContext,
		) {
			input := createChannel(ctx, telem.Float32T)
			ch := create(ctx, calculated(input))
			Expect(ch.Index).ToNot(BeZero())
			Expect(deleteChannels(AuthedCtx(ctx, u), apichannel.DeleteRequest{
				Keys: channel.Keys{ch.Key},
			})).Error().To(Succeed())
			Expect(exists(ctx, ch.Key)).To(BeFalse())
			Expect(exists(ctx, ch.Index)).To(BeFalse())
		})

		It("Should let a folder go once its calculated channels are deleted", func(
			ctx SpecContext,
		) {
			input := createChannel(ctx, telem.Float32T)
			g := createChannelGroup(ctx, "Breaker d "+uuid.New().String())
			req := calculated(input)
			req.Group = &g.Key
			ch := create(ctx, req)
			Expect(deleteChannels(AuthedCtx(ctx, u), apichannel.DeleteRequest{
				Keys: channel.Keys{ch.Key},
			})).Error().To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return groupSvc.NewWriter(tx).Delete(ctx, g.Key)
			})).To(Succeed())
		})

		It("Should move the index with the channel into a folder", func(
			ctx SpecContext,
		) {
			input := createChannel(ctx, telem.Float32T)
			ch := create(ctx, calculated(input))
			g := createChannelGroup(ctx, "Breaker d "+uuid.New().String())
			Expect(dispatch(AuthedCtx(ctx, u), apichannel.DispatchRequest{
				Key:     ch.Key,
				Actions: []channel.Action{set("group", g.Key.String())},
			})).Error().To(Succeed())
			Expect(retrieveChannel(ctx, ch.Key).Group).To(Equal(&g.Key))
			Expect(retrieveChannel(ctx, ch.Index).Group).To(Equal(&g.Key))
		})
	})

	Describe("Virtual filter", func() {
		retrieve := func(ctx SpecContext, key channel.Key, virtual bool) []channel.Key {
			GinkgoHelper()
			res := MustSucceed(retrieveChannels(
				AuthedCtx(ctx, u),
				apichannel.RetrieveRequest{Where: &channel.Where{
					Key:     &filter.Set[channel.Key]{In: []channel.Key{key}},
					Virtual: new(virtual),
				}},
			))
			keys := make([]channel.Key, len(res.Channels))
			for i, ch := range res.Channels {
				keys[i] = ch.Key
			}
			return keys
		}

		It("Should leave a calculated channel out of the virtual channels", func(
			ctx SpecContext,
		) {
			input := createChannel(ctx, telem.Float32T)
			ch := create(ctx, calculated(input))
			Expect(retrieve(ctx, ch.Key, true)).To(BeEmpty())
			Expect(retrieve(ctx, ch.Key, false)).To(ConsistOf(ch.Key))
		})
	})

	Describe("Broadcasts", func() {
		var rec *Recorder[channel.Key, channel.Action]
		BeforeEach(func() {
			rec = &Recorder[channel.Key, channel.Action]{}
			DeferCleanup(channelSvc.Actions().OnChange(rec.Record))
		})
		actionsOn := func(key channel.Key, dispatchKey string) []channel.Action {
			var out []channel.Action
			for _, batch := range rec.Snapshot() {
				if batch.Key == key && batch.DispatchKey == dispatchKey {
					out = append(out, batch.Actions...)
				}
			}
			return out
		}

		It(
			"Should broadcast the created channel with no expression, then its "+
				"expression",
			func(ctx SpecContext) {
				input := createChannel(ctx, telem.Float32T)
				req := calculated(input)
				ch := create(ctx, req)
				acts := actionsOn(ch.Key, "")
				created := slices.IndexFunc(acts, func(a channel.Action) bool {
					return a.Create != nil
				})
				Expect(created).To(BeNumerically(">=", 0))
				Expect(acts[created].Create.Channel.Expression).To(BeEmpty())
				Expect(acts[created+1:]).
					To(ContainElement(set("expression", req.Expression)))
			},
		)

		It("Should broadcast an old API set of the expression on the channel", func(
			ctx SpecContext,
		) {
			input := createChannel(ctx, telem.Float32T)
			ch := create(ctx, calculated(input))
			edit := set("expression", fmt.Sprintf("return %s * 3", input.Name))
			Expect(dispatch(AuthedCtx(ctx, u), apichannel.DispatchRequest{
				Key:         ch.Key,
				DispatchKey: "breaker-d",
				Actions:     []channel.Action{edit},
			})).Error().To(Succeed())
			Expect(actionsOn(ch.Key, "breaker-d")).To(ContainElement(edit))
		})

		It("Should broadcast the operations the calculation holds after an old API set",
			func(ctx SpecContext) {
				input := createChannel(ctx, telem.Float32T)
				ch := create(ctx, calculated(input))
				avg := channel.Operation{
					Type:     channel.OperationTypeAvg,
					Duration: telem.Second,
				}
				// The record drops the none operation.
				sent := []channel.Operation{{Type: channel.OperationTypeNone}, avg}
				Expect(dispatch(AuthedCtx(ctx, u), apichannel.DispatchRequest{
					Key:         ch.Key,
					DispatchKey: "breaker-d",
					Actions:     []channel.Action{set("operations", sent)},
				})).Error().To(Succeed())
				Expect(actionsOn(ch.Key, "breaker-d")).To(ContainElement(
					set("operations", []channel.Operation{avg}),
				))
			},
		)
	})

	Describe("Status", func() {
		It("Should compose the status with the channel as its owner", func(
			ctx SpecContext,
		) {
			input := createChannel(ctx, telem.Float32T)
			ch := create(ctx, calculated(input))
			// The graph writes the error status once the input is gone.
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Delete(ctx, input.Key(), false)
			})).To(Succeed())
			var st *apichannel.Status
			Eventually(func(g Gomega) {
				res, err := retrieveChannels(
					AuthedCtx(ctx, u),
					apichannel.RetrieveRequest{
						Where: &channel.Where{
							Key: &filter.Set[channel.Key]{Eq: &ch.Key},
						},
						IncludeStatus: true,
					},
				)
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(res.Channels).To(HaveLen(1))
				g.Expect(res.Channels[0].Status).ToNot(BeNil())
				st = res.Channels[0].Status
			}).Should(Succeed())
			Expect(st.Owner).To(Equal(new(channel.ResourceID(ch.Key))))
		})
	})

	Describe("Access", func() {
		var (
			input channel.Channel
			ch    apichannel.Channel
		)
		BeforeEach(func(ctx SpecContext) {
			input = createChannel(ctx, telem.Float32T)
			ch = create(ctx, calculated(input))
		})
		retrieveAs := func(ctx SpecContext, subject user.User) apichannel.Channel {
			GinkgoHelper()
			res := MustSucceed(retrieveChannels(
				AuthedCtx(ctx, subject),
				apichannel.RetrieveRequest{
					Where: &channel.Where{Key: &filter.Set[channel.Key]{Eq: &ch.Key}},
				},
			))
			Expect(res.Channels).To(HaveLen(1))
			return res.Channels[0]
		}
		reader := func(ctx SpecContext, objects ...resource.ID) user.User {
			GinkgoHelper()
			r := MustSucceed(rbactestutil.CreateUser(
				ctx,
				db,
				rbacSvc,
				userSvc,
				user.User{Name: "breaker-d-reader-" + uuid.New().String()},
			))
			grantOn(
				ctx,
				r.ResourceID(),
				[]access.Action{access.ActionRetrieve},
				objects...,
			)
			return r
		}

		It(
			"Should hide the expression from a subject that cannot retrieve the calculation",
			func(ctx SpecContext) {
				r := reader(ctx, resource.ID{Type: resource.TypeChannel})
				got := retrieveAs(ctx, r)
				Expect(got.Expression).To(BeEmpty())
			},
		)

		It(
			"Should tell a subject that cannot retrieve the calculation that the channel is calculated",
			func(ctx SpecContext) {
				Expect(ch.Calculated).To(BeTrue())
				r := reader(ctx, resource.ID{Type: resource.TypeChannel})
				Expect(retrieveAs(ctx, r).Calculated).To(BeTrue())
			},
		)

		It("Should not call a virtual channel that no calculation writes calculated",
			func(ctx SpecContext) {
				plain := create(ctx, apichannel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.Float32T,
					Virtual:  true,
				})
				Expect(plain.Calculated).To(BeFalse())
				res := MustSucceed(retrieveChannels(
					AuthedCtx(ctx, u),
					apichannel.RetrieveRequest{
						Where: &channel.Where{
							Key: &filter.Set[channel.Key]{Eq: &plain.Key},
						},
					},
				))
				Expect(res.Channels).To(HaveLen(1))
				Expect(res.Channels[0].Calculated).To(BeFalse())
			},
		)

		It("Should hide the expression from a subject that cannot retrieve its input",
			func(ctx SpecContext) {
				r := reader(
					ctx,
					channel.ResourceID(ch.Key),
					resource.ID{Type: resource.TypeCalculation},
				)
				got := retrieveAs(ctx, r)
				Expect(got.Expression).ToNot(ContainSubstring(input.Name))
			},
		)

		It("Should rename a calculated channel for a subject that can update it", func(
			ctx SpecContext,
		) {
			r := reader(
				ctx,
				channel.ResourceID(ch.Key),
				resource.ID{Type: resource.TypeCalculation},
			)
			grantOn(
				ctx,
				r.ResourceID(),
				[]access.Action{access.ActionUpdate},
				channel.ResourceID(ch.Key),
				resource.ID{Type: resource.TypeCalculation},
			)
			name := UniqueChannelName()
			Expect(dispatch(AuthedCtx(ctx, r), apichannel.DispatchRequest{
				Key: ch.Key,
				Actions: []channel.Action{
					channel.NewRenameAction(channel.RenamePayload{Name: name}),
				},
			})).Error().To(Succeed())
			Expect(retrieveChannel(ctx, ch.Key).Name).To(Equal(name))
		})
	})

	Describe("Records", func() {
		retrieveOne := func(ctx SpecContext, key channel.Key) apichannel.Channel {
			GinkgoHelper()
			res := MustSucceed(retrieveChannels(
				AuthedCtx(ctx, u),
				apichannel.RetrieveRequest{
					Where: &channel.Where{
						Key: &filter.Set[channel.Key]{Eq: &key},
					},
					IncludeStatus: true,
				},
			))
			Expect(res.Channels).To(HaveLen(1))
			return res.Channels[0]
		}
		recordOf := func(ctx SpecContext, key channel.Key) calculation.Calculation {
			GinkgoHelper()
			var c calculation.Calculation
			Expect(layer.Calculation.NewRetrieve().
				Where(calculation.MatchOutputs(key)).
				Entry(&c).
				Exec(ctx, db)).To(Succeed())
			return c
		}

		It(
			"Should serve a channel whose calculation is deleted with no expression",
			func(
				ctx SpecContext,
			) {
				input := createChannel(ctx, telem.Float32T)
				reset := createChannel(ctx, telem.BooleanT)
				req := calculated(input)
				req.Operations = []channel.Operation{{
					Type:         channel.OperationTypeMax,
					ResetChannel: reset.Key(),
				}}
				ch := create(ctx, req)
				c := recordOf(ctx, ch.Key)
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return statusSvc.NewWriter(tx).Set(
						ctx,
						calculation.StatusFromError(c.Key, ch.Name, errors.New("broken")),
					)
				})).To(Succeed())
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return layer.Calculation.NewWriter(tx).Delete(ctx, c.Key)
				})).To(Succeed())
				got := retrieveOne(ctx, ch.Key)
				Expect(got.Expression).To(BeEmpty())
				Expect(got.Operations).To(BeEmpty())
				Expect(got.Status).To(BeNil())
			},
		)

		It("Should serve an expression that the calculation API edited", func(
			ctx SpecContext,
		) {
			input := createChannel(ctx, telem.Float32T)
			ch := create(ctx, calculated(input))
			c := recordOf(ctx, ch.Key)
			expression := fmt.Sprintf("return %s * 5", input.Name)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				_, err := layer.Calculation.NewWriter(tx).Apply(
					ctx,
					c.Key,
					calculation.NewSetAction(calculation.SetPayload{
						Path:  []string{"expression"},
						Value: expression,
					}),
				)
				return err
			})).To(Succeed())
			Expect(retrieveOne(ctx, ch.Key).Expression).To(Equal(expression))
		})

		It("Should serve the expression of each calculated channel on every page", func(
			ctx SpecContext,
		) {
			input := createChannel(ctx, telem.Float32T)
			prefix := "brkd" + uuid.New().String()[:8]
			want := map[channel.Key]string{}
			for i := range 5 {
				req := calculated(input)
				req.Name = fmt.Sprintf("%s_%d", prefix, i)
				req.Expression = fmt.Sprintf("return %s * %d", input.Name, i+1)
				ch := create(ctx, req)
				want[ch.Key] = req.Expression
			}
			got := map[channel.Key]string{}
			var after *apichannel.Cursor
			for range 5 {
				res := MustSucceed(retrieveChannels(
					AuthedCtx(ctx, u),
					apichannel.RetrieveRequest{
						Where: &channel.Where{
							Name:    &filter.String{StartsWith: &prefix},
							IsIndex: &filter.Bool{Eq: new(false)},
						},
						Order: &apichannel.Order{Field: apichannel.OrderFieldName},
						After: after,
						Limit: new(uint32(2)),
					},
				))
				if len(res.Channels) == 0 {
					break
				}
				for _, ch := range res.Channels {
					got[ch.Key] = ch.Expression
				}
				last := res.Channels[len(res.Channels)-1]
				after = &apichannel.Cursor{Key: last.Key, Name: &last.Name}
			}
			Expect(got).To(Equal(want))
		})
	})

	Describe("Wire shape", func() {
		// v062 returns ch as the v0.62 translator did, from the stored channel and the
		// expression and operations it stored as sent.
		v062 := func(
			ctx SpecContext,
			ch apichannel.Channel,
			req apichannel.Channel,
		) apichannel.Channel {
			stored := retrieveChannel(ctx, ch.Key)
			return apichannel.Channel{
				Key:         stored.Key(),
				Name:        stored.Name,
				Leaseholder: stored.Leaseholder,
				DataType:    stored.DataType,
				IsIndex:     stored.IsIndex,
				Index:       stored.Index(),
				Virtual:     stored.Virtual,
				Expression:  req.Expression,
				Operations:  req.Operations,
				Group:       stored.Group,
				// Additive since v0.62, which a v0.62 client ignores.
				Calculated: req.Expression != "",
			}
		}
		roundTrip := func(ctx SpecContext, req apichannel.Channel) {
			GinkgoHelper()
			created := create(ctx, req)
			res := MustSucceed(retrieveChannels(
				AuthedCtx(ctx, u),
				apichannel.RetrieveRequest{
					Where: &channel.Where{
						Key: &filter.Set[channel.Key]{Eq: &created.Key},
					},
				},
			))
			Expect(res.Channels).To(HaveLen(1))
			want := v062(ctx, created, req)
			Expect(created).To(Equal(want))
			Expect(res.Channels[0]).To(Equal(want))
			Expect(MustSucceed(json.Marshal(res.Channels[0]))).
				To(MatchJSON(MustSucceed(json.Marshal(want))))
			Expect(MustSucceed(msgpack.Marshal(res.Channels[0]))).
				To(Equal(MustSucceed(msgpack.Marshal(want))))
		}

		It("Should return windowed operations as v0.62 did", func(ctx SpecContext) {
			input := createChannel(ctx, telem.Float32T)
			reset := createChannel(ctx, telem.BooleanT)
			req := calculated(input)
			req.Operations = []channel.Operation{
				{Type: channel.OperationTypeAvg, Duration: telem.Second},
				{Type: channel.OperationTypeMax, ResetChannel: reset.Key()},
			}
			roundTrip(ctx, req)
		})
	})

	Describe("Create batches", func() {
		It("Should return the stored expression when a create finds the channel", func(
			ctx SpecContext,
		) {
			input := createChannel(ctx, telem.Float32T)
			first := create(ctx, calculated(input))
			again := calculated(input)
			again.Name = first.Name
			again.Expression = fmt.Sprintf("return %s * 7", input.Name)
			res := MustSucceed(createChannels(
				AuthedCtx(ctx, u),
				apichannel.CreateRequest{
					Channels:             []apichannel.Channel{again},
					RetrieveIfNameExists: true,
				},
			))
			Expect(res.Channels).To(HaveLen(1))
			Expect(res.Channels[0].Key).To(Equal(first.Key))
			Expect(res.Channels[0].Expression).To(Equal(first.Expression))
		})

		It("Should make no channel of a batch whose later calculation fails", func(
			ctx SpecContext,
		) {
			input := createChannel(ctx, telem.Float32T)
			level := createChannel(ctx, telem.Float32T)
			good := calculated(input)
			bad := calculated(input)
			bad.Operations = []channel.Operation{{
				Type:         channel.OperationTypeMax,
				ResetChannel: level.Key(),
			}}
			Expect(createChannels(AuthedCtx(ctx, u), apichannel.CreateRequest{
				Channels: []apichannel.Channel{good, bad},
			})).Error().To(MatchError(ContainSubstring("reset")))
			Expect(MustSucceed(channelSvc.NewRetrieve().
				Where(channel.MatchNames(
					good.Name,
					good.Name+"_time",
					bad.Name,
					bad.Name+"_time",
				)).
				Exists(ctx, db))).To(BeFalse())
		})
	})

	Describe("Create with a key", func() {
		It("Should refuse a calculated channel whose key is stored", func(
			ctx SpecContext,
		) {
			input := createChannel(ctx, telem.Float32T)
			ch := create(ctx, calculated(input))
			again := ch
			again.Name = UniqueChannelName()
			again.Expression = fmt.Sprintf("return %s * 4", input.Name)
			Expect(createChannels(
				AuthedCtx(ctx, u),
				apichannel.CreateRequest{Channels: []apichannel.Channel{again}},
			)).Error().To(MatchError(validate.ErrValidation))
			Expect(MustSucceed(channelSvc.NewRetrieve().
				Where(channel.MatchNames(again.Name)).
				Exists(ctx, db))).To(BeFalse())
		})
	})

	Describe("Operation types", func() {
		It("Should refuse a dispatch of an operation with an unknown type", func(
			ctx SpecContext,
		) {
			input := createChannel(ctx, telem.Float32T)
			ch := create(ctx, calculated(input))
			var err error
			Expect(func() {
				_, err = dispatch(AuthedCtx(ctx, u), apichannel.DispatchRequest{
					Key: ch.Key,
					Actions: []channel.Action{set("operations", []channel.Operation{{
						Type:     "sum",
						Duration: telem.Second,
					}})},
				})
			}).ToNot(Panic())
			Expect(err).To(MatchError(validate.ErrValidation))
		})

		It("Should refuse a create of an operation with an unknown type", func(
			ctx SpecContext,
		) {
			input := createChannel(ctx, telem.Float32T)
			req := calculated(input)
			req.Operations = []channel.Operation{
				{Type: "sum", Duration: telem.Second},
			}
			var err error
			Expect(func() {
				_, err = createChannels(
					AuthedCtx(ctx, u),
					apichannel.CreateRequest{Channels: []apichannel.Channel{req}},
				)
			}).ToNot(Panic())
			Expect(err).To(MatchError(validate.ErrValidation))
			Expect(err).To(MatchError(ContainSubstring("0.operations.0.type")))
		})
	})
})
