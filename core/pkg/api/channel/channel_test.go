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
	"math/rand/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/gorp"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func createChannel(ctx SpecContext, dt telem.DataType) channel.Channel {
	GinkgoHelper()
	return createNamedChannel(ctx, UniqueChannelName(), dt)
}

func createNamedChannel(
	ctx SpecContext,
	name string,
	dt telem.DataType,
) channel.Channel {
	GinkgoHelper()
	ch := channel.Channel{Name: name, DataType: dt, Virtual: true}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return channelSvc.NewWriter(tx).Create(ctx, &ch)
	})).To(Succeed())
	return ch
}

func createChannelGroup(ctx SpecContext, name string) group.Group {
	GinkgoHelper()
	g := group.Group{Name: name, Scope: resource.TypeChannel}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return groupSvc.NewWriter(tx).Create(ctx, &g)
	})).To(Succeed())
	return g
}

func retrieveChannel(ctx SpecContext, key channel.Key) channel.Channel {
	GinkgoHelper()
	var ch channel.Channel
	Expect(channelSvc.NewRetrieve().
		Where(channel.MatchKeys(key)).
		Entry(&ch).
		Exec(ctx, db)).To(Succeed())
	return ch
}

var _ = Describe("Service", func() {
	Describe("Create", func() {
		var (
			u              user.User
			public, secret channel.Channel
		)
		calculate := func(input string) apichannel.Channel {
			return apichannel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Float32T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s * 2", input),
			}
		}
		create := func(
			ctx SpecContext,
			channels ...apichannel.Channel,
		) (apichannel.CreateResponse, error) {
			return createChannels(
				AuthedCtx(ctx, u),
				apichannel.CreateRequest{Channels: channels},
			)
		}
		BeforeEach(func(ctx SpecContext) {
			u = MustSucceed(rbactestutil.CreateUser(
				ctx,
				db,
				rbacSvc,
				userSvc,
				user.User{Name: "calculations-" + uuid.New().String()},
			))
			public = createChannel(ctx, telem.Float32T)
			secret = createChannel(ctx, telem.Float32T)
			grantOn(
				ctx,
				u.ResourceID(),
				[]access.Action{access.ActionCreate},
				resource.ID{Type: resource.TypeChannel},
			)
			grantOn(
				ctx,
				u.ResourceID(),
				[]access.Action{access.ActionRetrieve},
				framer.ResourceIDs(channel.Keys{public.Key()})...,
			)
		})
		It(
			"Should refuse a calculation that reads a channel the subject cannot read",
			func(ctx SpecContext) {
				Expect(create(ctx, calculate(secret.Name))).
					Error().To(MatchError(access.ErrDenied))
			},
		)
		It(
			"Should refuse a calculation reading an unreadable channel listed with it",
			func(ctx SpecContext) {
				listed := MustSucceed(create(ctx, calculate(public.Name))).Channels[0]
				Expect(create(ctx, listed, calculate(listed.Name))).
					Error().To(MatchError(access.ErrDenied))
			},
		)
		It(
			"Should refuse a calculation reading an unreadable channel through a decoy",
			func(ctx SpecContext) {
				decoy := apichannel.Channel{
					Name:       secret.Name,
					DataType:   telem.Float32T,
					Virtual:    true,
					Expression: "return 1",
				}
				Expect(createChannels(
					AuthedCtx(ctx, u),
					apichannel.CreateRequest{
						Channels: []apichannel.Channel{
							decoy,
							calculate(secret.Name),
						},
						RetrieveIfNameExists: true,
					},
				)).Error().To(MatchError(access.ErrDenied))
			},
		)
		It(
			"Should refuse an edit that reads an unreadable channel through a stored one",
			func(ctx SpecContext) {
				stored := channel.Channel{
					Name:       UniqueChannelName(),
					DataType:   telem.Float32T,
					Expression: fmt.Sprintf("return %s * 2", secret.Name),
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return channelSvc.NewWriter(tx).Create(ctx, &stored)
				})).To(Succeed())
				grantOn(
					ctx,
					u.ResourceID(),
					[]access.Action{access.ActionRetrieve},
					framer.ResourceIDs(channel.Keys{stored.Key()})...,
				)
				calc := MustSucceed(create(ctx, calculate(public.Name))).Channels[0]
				plain := apichannel.Channel{
					Key:         stored.Key(),
					Name:        stored.Name,
					Leaseholder: stored.Leaseholder,
					DataType:    telem.Float32T,
					Virtual:     true,
				}
				calc.Expression = fmt.Sprintf("return %s * 2", stored.Name)
				Expect(create(ctx, plain, calc)).
					Error().To(MatchError(access.ErrDenied))
				Expect(retrieveChannel(ctx, calc.Key).Expression).
					To(Equal(fmt.Sprintf("return %s * 2", public.Name)))
			},
		)
		It(
			"Should refuse an edit that makes a calculation read an unreadable channel",
			func(ctx SpecContext) {
				calc := MustSucceed(create(ctx, calculate(public.Name))).Channels[0]
				calc.Expression = fmt.Sprintf("return %s * 2", secret.Name)
				Expect(create(ctx, calc)).Error().To(MatchError(access.ErrDenied))
				Expect(retrieveChannel(ctx, calc.Key).Expression).
					To(Equal(fmt.Sprintf("return %s * 2", public.Name)))
			},
		)
	})

	Describe("Group", func() {
		It(
			"Should create a channel in a group and return the group on retrieve",
			func(ctx SpecContext) {
				grantOn(
					ctx,
					author.ResourceID(),
					[]access.Action{access.ActionCreate, access.ActionRetrieve},
					resource.ID{Type: resource.TypeChannel},
				)
				g := group.Group{Name: "API Channels", Scope: resource.TypeChannel}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return groupSvc.NewWriter(tx).Create(ctx, &g)
				})).To(Succeed())
				grantOn(
					ctx,
					author.ResourceID(),
					[]access.Action{access.ActionUpdate},
					group.ResourceID(g.Key),
				)
				created := MustSucceed(createChannels(
					AuthedCtx(ctx, author),
					apichannel.CreateRequest{Channels: []apichannel.Channel{{
						Name:     UniqueChannelName(),
						DataType: telem.TimestampT,
						IsIndex:  true,
						Group:    &g.Key,
					}}},
				)).Channels
				Expect(created[0].Group).To(Equal(&g.Key))
				retrieved := MustSucceed(retrieveChannels(
					AuthedCtx(ctx, author),
					apichannel.RetrieveRequest{
						Where: &channel.Where{
							Key: &filter.Set[channel.Key]{Eq: &created[0].Key},
						},
					},
				)).Channels
				Expect(retrieved[0].Group).To(Equal(&g.Key))
			},
		)
		It(
			"Should refuse a created channel without update on its group",
			func(ctx SpecContext) {
				grantOn(
					ctx,
					author.ResourceID(),
					[]access.Action{access.ActionCreate},
					resource.ID{Type: resource.TypeChannel},
				)
				g := createChannelGroup(ctx, "Create denied")
				Expect(createChannels(
					AuthedCtx(ctx, author),
					apichannel.CreateRequest{Channels: []apichannel.Channel{{
						Name:     UniqueChannelName(),
						DataType: telem.TimestampT,
						IsIndex:  true,
						Group:    &g.Key,
					}}},
				)).Error().To(MatchError(access.ErrDenied))
			},
		)
	})
	Describe("RetrieveIfNameExists", func() {
		var (
			existing channel.Channel
			u        user.User
			create   = func(ctx SpecContext) (apichannel.CreateResponse, error) {
				return createChannels(AuthedCtx(ctx, u), apichannel.CreateRequest{
					Channels: []apichannel.Channel{{
						Name:     existing.Name,
						DataType: existing.DataType,
						Virtual:  true,
					}},
					RetrieveIfNameExists: true,
				})
			}
		)
		BeforeEach(func(ctx SpecContext) {
			existing = createChannel(ctx, telem.Float32T)
			u = MustSucceed(rbactestutil.CreateUser(
				ctx, db, rbacSvc, userSvc, user.User{Name: uuid.New().String()},
			))
		})

		It(
			"Should deny an existing channel that the subject cannot retrieve",
			func(ctx SpecContext) {
				Expect(create(ctx)).Error().To(SatisfyAll(
					MatchError(access.ErrDenied),
					MatchError(ContainSubstring(
						"%s cannot %s %s",
						u.ResourceID(),
						access.ActionRetrieve,
						existing.ResourceID(),
					)),
				))
			},
		)
		It(
			"Should return an existing channel that the subject can retrieve",
			func(ctx SpecContext) {
				grantOn(
					ctx,
					u.ResourceID(),
					[]access.Action{access.ActionRetrieve},
					existing.ResourceID(),
				)
				res := MustSucceed(create(ctx))
				Expect(res.Channels).To(HaveLen(1))
				Expect(res.Channels[0].Key).To(Equal(existing.Key()))
			},
		)
	})

	Describe("Storage", func() {
		var (
			ch       channel.Channel
			outsider user.User
			start    = 10 * telem.SecondTS
		)
		BeforeEach(func(ctx SpecContext) {
			ch = channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.TimestampT,
				IsIndex:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			Expect(dist.Storage.TS.WriteSeries(
				ctx,
				ch.Key().StorageKey(),
				start,
				telem.NewSeriesSecondsTSV(10, 11, 12),
			)).To(Succeed())
			outsider = MustSucceed(rbactestutil.CreateUser(
				ctx, db, rbacSvc, userSvc, user.User{Name: uuid.New().String()},
			))
		})
		stored := func(ctx SpecContext) ts.Channel {
			GinkgoHelper()
			return MustSucceed(
				dist.Storage.TS.RetrieveChannel(ctx, ch.Key().StorageKey()),
			)
		}

		It(
			"Should keep the channel and its data when the subject cannot delete it",
			func(ctx SpecContext) {
				Expect(deleteChannels(
					AuthedCtx(ctx, outsider),
					apichannel.DeleteRequest{Keys: channel.Keys{ch.Key()}},
				)).Error().To(MatchError(access.ErrDenied))
				Expect(retrieveChannel(ctx, ch.Key()).Name).To(Equal(ch.Name))
				frame := MustSucceed(dist.Storage.TS.Read(
					ctx,
					telem.TimeRangeMax,
					ch.Key().StorageKey(),
				))
				Expect(frame.Len()).To(Equal(int64(3)))
			},
		)
		It(
			"Should delete the channel and its storage when the subject can delete it",
			func(ctx SpecContext) {
				grantOn(
					ctx,
					outsider.ResourceID(),
					[]access.Action{access.ActionDelete},
					channel.ResourceID(ch.Key()),
				)
				MustSucceed(deleteChannels(
					AuthedCtx(ctx, outsider),
					apichannel.DeleteRequest{Names: []string{ch.Name}},
				))
				Expect(channelSvc.NewRetrieve().
					Where(channel.MatchKeys(ch.Key())).
					Exists(ctx, db)).To(BeFalse())
				Expect(dist.Storage.TS.RetrieveChannel(ctx, ch.Key().StorageKey())).
					Error().To(MatchError(ts.ErrChannelNotFound))
			},
		)
		Describe("Dispatch", func() {
			renameTo := func(name string) apichannel.DispatchRequest {
				return apichannel.DispatchRequest{
					Key: ch.Key(),
					Actions: []channel.Action{
						channel.NewRenameAction(channel.RenamePayload{Name: name}),
					},
				}
			}
			It("Should refuse the actions without update on the channel", func(
				ctx SpecContext,
			) {
				Expect(
					dispatch(AuthedCtx(ctx, outsider), renameTo(UniqueChannelName())),
				).
					Error().
					To(MatchError(access.ErrDenied))
				Expect(retrieveChannel(ctx, ch.Key()).Name).To(Equal(ch.Name))
				Expect(stored(ctx).Name).To(Equal(ch.Name))
			})
			It("Should rename the channel in both stores with update on it", func(
				ctx SpecContext,
			) {
				grantOn(
					ctx,
					outsider.ResourceID(),
					[]access.Action{access.ActionUpdate},
					channel.ResourceID(ch.Key()),
				)
				name := UniqueChannelName()
				Expect(dispatch(AuthedCtx(ctx, outsider), renameTo(name))).
					To(Equal(apichannel.DispatchResponse{}))
				Expect(retrieveChannel(ctx, ch.Key()).Name).To(Equal(name))
				Expect(stored(ctx).Name).To(Equal(name))
			})
		})
	})
	Describe("Dispatch group move", func() {
		// denied matches an error that denies the subject the action on object.
		denied := func(
			subject resource.ID,
			action access.Action,
			object resource.ID,
		) types.GomegaMatcher {
			return SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(
					ContainSubstring("%s cannot %s %s", subject, action, object),
				),
			)
		}
		move := func(
			ctx SpecContext,
			g group.Group,
			ch channel.Channel,
		) (apichannel.DispatchResponse, error) {
			return dispatch(AuthedCtx(ctx, author), apichannel.DispatchRequest{
				Key: ch.Key(),
				Actions: []channel.Action{channel.NewSetAction(channel.SetPayload{
					Path:  []string{"group"},
					Value: g.Key.String(),
				})},
			})
		}
		It(
			"Should refuse a move without update on the destination group",
			func(ctx SpecContext) {
				ch := createChannel(ctx, telem.Float32T)
				g := createChannelGroup(ctx, "Set group denied")
				grantOn(
					ctx,
					author.ResourceID(),
					[]access.Action{access.ActionUpdate},
					channel.ResourceID(ch.Key()),
				)
				Expect(move(ctx, g, ch)).
					Error().
					To(denied(author.ResourceID(), access.ActionUpdate, g.ResourceID()))
				Expect(retrieveChannel(ctx, ch.Key()).Group).To(BeNil())
			},
		)
		It(
			"Should refuse a move without update on the source group",
			func(ctx SpecContext) {
				ch := createChannel(ctx, telem.Float32T)
				from := createChannelGroup(ctx, "Set group source denied")
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return channelSvc.NewWriter(tx).SetGroup(ctx, &from.Key, ch.Key())
				})).
					To(Succeed())
				to := createChannelGroup(ctx, "Set group destination granted")
				grantOn(
					ctx,
					author.ResourceID(),
					[]access.Action{access.ActionUpdate},
					channel.ResourceID(ch.Key()),
					to.ResourceID(),
				)
				Expect(move(ctx, to, ch)).Error().To(denied(
					author.ResourceID(),
					access.ActionUpdate,
					from.ResourceID(),
				))
				Expect(retrieveChannel(ctx, ch.Key()).Group).To(Equal(&from.Key))
			},
		)
		It(
			"Should refuse a move without update on the channel",
			func(ctx SpecContext) {
				ch := createChannel(ctx, telem.Float32T)
				g := createChannelGroup(ctx, "Set group channel denied")
				grantOn(
					ctx,
					author.ResourceID(),
					[]access.Action{access.ActionUpdate},
					g.ResourceID(),
				)
				Expect(move(ctx, g, ch)).Error().To(denied(
					author.ResourceID(),
					access.ActionUpdate,
					channel.ResourceID(ch.Key()),
				))
				Expect(retrieveChannel(ctx, ch.Key()).Group).To(BeNil())
			},
		)
		It(
			"Should move a channel with update on the channel and both groups",
			func(ctx SpecContext) {
				ch := createChannel(ctx, telem.Float32T)
				from := createChannelGroup(ctx, "Set group source granted")
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return channelSvc.NewWriter(tx).SetGroup(ctx, &from.Key, ch.Key())
				})).
					To(Succeed())
				to := createChannelGroup(ctx, "Set group granted")
				grantOn(
					ctx,
					author.ResourceID(),
					[]access.Action{access.ActionUpdate},
					channel.ResourceID(ch.Key()),
					from.ResourceID(),
					to.ResourceID(),
				)
				MustSucceed(move(ctx, to, ch))
				Expect(retrieveChannel(ctx, ch.Key()).Group).To(Equal(&to.Key))
			},
		)
	})
	Describe("CountLimited", func() {
		var u user.User
		BeforeEach(func(ctx SpecContext) {
			u = MustSucceed(rbactestutil.CreateUser(
				ctx, db, rbacSvc, userSvc, user.User{Name: uuid.New().String()},
			))
		})
		It("Should refuse a subject that cannot retrieve every channel", func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx, telem.Float32T)
			grantOn(
				ctx,
				u.ResourceID(),
				[]access.Action{access.ActionRetrieve},
				channel.ResourceID(ch.Key()),
			)
			Expect(countLimited(AuthedCtx(ctx, u), apichannel.CountLimitedRequest{})).
				Error().To(MatchError(access.ErrDenied))
		})
		It("Should count a stored channel and leave out a virtual one", func(
			ctx SpecContext,
		) {
			grantOn(
				ctx,
				u.ResourceID(),
				[]access.Action{access.ActionRetrieve},
				resource.ID{Type: resource.TypeChannel},
			)
			before := MustSucceed(
				countLimited(AuthedCtx(ctx, u), apichannel.CountLimitedRequest{}),
			)
			Expect(before.Count).To(Equal(channelSvc.CountLimited()))
			createChannel(ctx, telem.Float32T)
			idx := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.TimestampT,
				IsIndex:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, &idx)
			})).To(Succeed())
			Expect(countLimited(AuthedCtx(ctx, u), apichannel.CountLimitedRequest{})).
				To(Equal(apichannel.CountLimitedResponse{Count: before.Count + 1}))
		})
	})
	Describe("RetrieveDataTypes", func() {
		It(
			"Should resolve the data types of the channels in key order",
			func(ctx SpecContext) {
				ch1 := createChannel(ctx, telem.Float32T)
				ch2 := createChannel(ctx, telem.Int64T)
				ch3 := createChannel(ctx, telem.Uint8T)
				Expect(apiSvc.RetrieveDataTypes(
					ctx, channel.Keys{ch1.Key(), ch2.Key(), ch3.Key()},
				)).To(Equal([]telem.DataType{telem.Float32T, telem.Int64T, telem.Uint8T}))
			},
		)

		It(
			"Should resolve a data type for each occurrence of a repeated key",
			func(ctx SpecContext) {
				ch1 := createChannel(ctx, telem.Float32T)
				ch2 := createChannel(ctx, telem.Int64T)
				Expect(apiSvc.RetrieveDataTypes(
					ctx, channel.Keys{ch1.Key(), ch2.Key(), ch1.Key()},
				)).To(Equal([]telem.DataType{
					telem.Float32T, telem.Int64T, telem.Float32T,
				}))
			},
		)

		It("Should resolve the data type of a single channel", func(ctx SpecContext) {
			ch := createChannel(ctx, telem.Float64T)
			Expect(apiSvc.RetrieveDataTypes(ctx, channel.Keys{ch.Key()})).
				To(Equal([]telem.DataType{telem.Float64T}))
		})

		It(
			"Should return an empty slice when no keys are provided",
			func(ctx SpecContext) {
				Expect(apiSvc.RetrieveDataTypes(ctx, channel.Keys{})).To(BeEmpty())
			},
		)

		It(
			"Should return a not found error when a key does not resolve to a channel",
			func(ctx SpecContext) {
				ch := createChannel(ctx, telem.Int32T)
				Expect(apiSvc.RetrieveDataTypes(
					ctx, channel.Keys{ch.Key(), channel.Key(999999)},
				)).Error().To(MatchError(query.ErrNotFound))
			},
		)
	})

	Describe("RetrieveName", func() {
		It("Should resolve the name of an existing channel", func(ctx SpecContext) {
			ch := createChannel(ctx, telem.Float32T)
			Expect(apiSvc.RetrieveName(ctx, ch.Key())).To(Equal(ch.Name))
		})

		It(
			"Should return an empty string when no channel has the key",
			func(ctx SpecContext) {
				Expect(apiSvc.RetrieveName(ctx, channel.Key(999999))).To(BeEmpty())
			},
		)
	})

	Describe("Retrieve", func() {
		Describe("IncludeStatus", func() {
			BeforeEach(func(ctx SpecContext) {
				grantOn(
					ctx,
					author.ResourceID(),
					[]access.Action{access.ActionRetrieve},
					resource.ID{Type: resource.TypeChannel},
				)
			})

			setStatus := func(ctx SpecContext, ch channel.Channel) {
				GinkgoHelper()
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return statusSvc.NewWriter(tx).Set(ctx, &apichannel.Status{
						Owner:   new(channel.ResourceID(ch.Key())),
						Name:    ch.Name,
						Variant: status.VariantSuccess,
						Message: "ok",
						Time:    telem.Now(),
					})
				})).To(Succeed())
			}

			retrieve := func(ctx SpecContext, keys ...channel.Key) []apichannel.Channel {
				GinkgoHelper()
				return MustSucceed(retrieveChannels(
					AuthedCtx(ctx, author),
					apichannel.RetrieveRequest{
						Where: &channel.Where{
							Key: &filter.Set[channel.Key]{In: keys},
						},
						IncludeStatus: true,
					},
				)).Channels
			}

			It(
				"Should attach the status of each channel that has one",
				func(ctx SpecContext) {
					ch := createChannel(ctx, telem.Float32T)
					setStatus(ctx, ch)
					res := retrieve(ctx, ch.Key())
					Expect(res).To(HaveLen(1))
					Expect(res[0].Status).ToNot(BeNil())
					Expect(res[0].Status.Message).To(Equal("ok"))
				},
			)

			It(
				"Should leave the status empty for a channel that has none",
				func(ctx SpecContext) {
					withStatus := createChannel(ctx, telem.Float32T)
					setStatus(ctx, withStatus)
					withoutStatus := createChannel(ctx, telem.Int64T)
					res := retrieve(ctx, withStatus.Key(), withoutStatus.Key())
					Expect(res).To(HaveLen(2))
					Expect(res[0].Status).ToNot(BeNil())
					Expect(res[1].Status).To(BeNil())
				},
			)
		})

		It("Should treat a missing key as a non-match", func(ctx SpecContext) {
			grantOn(
				ctx,
				author.ResourceID(),
				[]access.Action{access.ActionRetrieve},
				resource.ID{Type: resource.TypeChannel},
			)
			ch := createChannel(ctx, telem.Float32T)
			res := MustSucceed(retrieveChannels(
				AuthedCtx(ctx, author),
				apichannel.RetrieveRequest{
					Where: &channel.Where{
						Key: &filter.Set[channel.Key]{
							In: []channel.Key{ch.Key(), 999999},
						},
					},
				},
			))
			Expect(res.Channels).To(HaveLen(1))
			Expect(res.Channels[0].Key).To(Equal(ch.Key()))
		})

		Describe("Access", func() {
			var (
				prefix           string
				reader           user.User
				allowed, blocked channel.Channel
			)
			BeforeEach(func(ctx SpecContext) {
				prefix = fmt.Sprintf("access%09d", rand.IntN(999999999))
				allowed = createNamedChannel(ctx, prefix+"_allowed", telem.Float32T)
				blocked = createNamedChannel(ctx, prefix+"_blocked", telem.Float32T)
				reader = MustSucceed(rbactestutil.CreateUser(
					ctx,
					db,
					rbacSvc,
					userSvc,
					user.User{Name: "reader-" + uuid.New().String()},
				))
				grantOn(
					ctx,
					reader.ResourceID(),
					[]access.Action{access.ActionRetrieve},
					allowed.ResourceID(),
				)
			})

			It("Should leave out the matches the subject cannot retrieve", func(
				ctx SpecContext,
			) {
				res := MustSucceed(retrieveChannels(
					AuthedCtx(ctx, reader),
					apichannel.RetrieveRequest{Where: &channel.Where{
						Name: &filter.String{StartsWith: new(prefix)},
					}},
				))
				Expect(res.Channels).To(ConsistOf(HaveField("Key", allowed.Key())))
			})

			It("Should deny a named channel the subject cannot retrieve", func(
				ctx SpecContext,
			) {
				Expect(retrieveChannels(
					AuthedCtx(ctx, reader),
					apichannel.RetrieveRequest{Where: &channel.Where{
						Key: &filter.Set[channel.Key]{
							In: []channel.Key{allowed.Key(), blocked.Key()},
						},
					}},
				)).Error().To(SatisfyAll(
					MatchError(access.ErrDenied),
					MatchError(ContainSubstring(blocked.ResourceID().String())),
				))
			})

			It(
				"Should deny a channel named by name that the subject cannot retrieve",
				func(
					ctx SpecContext,
				) {
					Expect(retrieveChannels(
						AuthedCtx(ctx, reader),
						apichannel.RetrieveRequest{Where: &channel.Where{
							Name: &filter.String{
								In: []string{allowed.Name, blocked.Name},
							},
						}},
					)).Error().To(MatchError(access.Denied(
						reader.ResourceID(),
						access.ActionRetrieve,
						blocked.ResourceID(),
					).Error()))
				},
			)

			It(
				"Should return a channel named by name that the subject can retrieve",
				func(
					ctx SpecContext,
				) {
					res := MustSucceed(retrieveChannels(
						AuthedCtx(ctx, reader),
						apichannel.RetrieveRequest{Where: &channel.Where{
							Name: &filter.String{Eq: new(allowed.Name)},
						}},
					))
					Expect(res.Channels).To(ConsistOf(HaveField("Key", allowed.Key())))
				},
			)
		})

		It("Should return a path error for an invalid regex", func(ctx SpecContext) {
			Expect(retrieveChannels(
				AuthedCtx(ctx, author),
				apichannel.RetrieveRequest{
					Where: &channel.Where{Name: &filter.String{Regex: new("(a")}},
				},
			)).Error().To(SatisfyAll(
				BeAssignableToTypeOf(validate.PathError{}),
				MatchError(
					"where.name.regex: error parsing regexp: missing closing ): `(a`: validation error",
				),
			))
		})
	})
})

var _ = Describe("RetrieveRequest", func() {
	It("Should reject a request member that it does not declare", func() {
		var req apichannel.RetrieveRequest
		Expect(json.Unmarshal([]byte(`{"keys":[1]}`), &req)).To(SatisfyAll(
			BeAssignableToTypeOf(&json.SemanticError{}),
			MatchError(ContainSubstring(`unknown object member name "keys"`)),
		))
	})
})
