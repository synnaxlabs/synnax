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
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Named line breaker", Ordered, ContinueOnFailure, func() {
	var (
		reader user.User
		// readPlain and readOutput are readable. The rest are not.
		readPlain, readOutput           channel.Channel
		hidePlain, hideOutput, hideData channel.Channel
		// upper is unreadable and has the name of readPlain in upper case.
		upper channel.Channel
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		base := "named_" + strings.TrimPrefix(UniqueChannelName(), "test_ch_")
		virtual := func(name string) channel.Channel {
			return channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
		}
		readPlain, upper = virtual(base+"_rp"), virtual(strings.ToUpper(base+"_rp"))
		readOutput = virtual(base + "_ro")
		hidePlain, hideOutput = virtual(base+"_hp"), virtual(base+"_ho")
		index := channel.Channel{
			Name:     base + "_idx",
			DataType: telem.TimestampT,
			IsIndex:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := channelSvc.NewWriter(tx)
			for _, ch := range []*channel.Channel{
				&readPlain, &upper, &readOutput, &hidePlain, &hideOutput, &index,
			} {
				if err := w.Create(ctx, ch); err != nil {
					return err
				}
			}
			hideData = channel.Channel{
				Name:       base + "_hd",
				DataType:   telem.Float32T,
				LocalIndex: index.LocalKey,
			}
			if err := w.Create(ctx, &hideData); err != nil {
				return err
			}
			for _, out := range []channel.Channel{readOutput, hideOutput} {
				c := calculation.Calculation{Expression: "return 2", Output: out.Key()}
				if err := layer.Calculation.NewWriter(tx).Create(ctx, &c); err != nil {
					return err
				}
			}
			return nil
		})).To(Succeed())
		reader = MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc,
			user.User{Name: "named-reader-" + uuid.New().String()}))
		readable := channel.Keys{readPlain.Key(), readOutput.Key()}
		grantOn(
			ctx,
			reader.ResourceID(),
			[]access.Action{access.ActionRetrieve},
			append(
				framer.ResourceIDs(readable),
				channel.ResourceIDsFromKeys(readable)...,
			)...,
		)
	})
	ask := func(ctx SpecContext, w channel.Where) ([]channel.Key, error) {
		res, err := retrieveChannels(
			AuthedCtx(ctx, reader),
			apichannel.RetrieveRequest{Where: &w},
		)
		if err != nil {
			return nil, err
		}
		return lo.Map(res.Channels, func(ch apichannel.Channel, _ int) channel.Key {
			return ch.Key
		}), err
	}
	eq := func(name string) *filter.String { return &filter.String{Eq: new(name)} }

	Describe("Top-level virtual line", func() {
		It("Should deny the name alone and the name with the calculated line", func(
			ctx SpecContext,
		) {
			Expect(ask(ctx, channel.Where{Name: eq(hidePlain.Name)})).Error().
				To(MatchError(access.ErrDenied))
			Expect(ask(ctx, channel.Where{
				Name:       eq(hidePlain.Name),
				Calculated: new(false),
			})).Error().To(MatchError(access.ErrDenied))
		})

		It("Should deny an unreadable channel that the name and virtual line match",
			func(ctx SpecContext) {
				// main enforces each match of a named retrieve, so this is denied.
				Expect(ask(ctx, channel.Where{
					Name:    eq(hidePlain.Name),
					Virtual: new(true),
				})).Error().To(MatchError(access.ErrDenied))
			},
		)

		It("Should deny a stored channel named with virtual false and calculated false",
			func(ctx SpecContext) {
				// D-338: a named retrieve with the calculated line enforces on every
				// channel the name alone matches.
				Expect(ask(ctx, channel.Where{
					Name:       eq(hideData.Name),
					Virtual:    new(false),
					Calculated: new(false),
				})).Error().To(MatchError(access.ErrDenied))
			},
		)

		It("Should deny an output named with virtual true and calculated true",
			func(ctx SpecContext) {
				Expect(ask(ctx, channel.Where{
					Name:       eq(hideOutput.Name),
					Virtual:    new(true),
					Calculated: new(true),
				})).Error().To(MatchError(access.ErrDenied))
			},
		)
	})

	Describe("Named retrieves with the calculated line", func() {
		It("Should deny a name in a list that an unreadable channel holds", func(
			ctx SpecContext,
		) {
			Expect(ask(ctx, channel.Where{
				Name: &filter.String{
					In: []string{readPlain.Name, hidePlain.Name},
				},
				Calculated: new(true),
			})).Error().To(MatchError(access.ErrDenied))
		})

		It("Should deny a case-insensitive name that an unreadable channel holds",
			func(ctx SpecContext) {
				Expect(ask(ctx, channel.Where{
					Name: &filter.String{
						Eq:              new(readPlain.Name),
						CaseInsensitive: true,
					},
					Calculated: new(true),
				})).Error().To(MatchError(access.ErrDenied))
			},
		)

		It("Should return nothing for a name that no channel holds", func(
			ctx SpecContext,
		) {
			Expect(ask(ctx, channel.Where{
				Name:       eq(readPlain.Name + "_none"),
				Calculated: new(true),
			})).To(BeEmpty())
		})

		It("Should deny the line nested under not and or", func(ctx SpecContext) {
			Expect(ask(ctx, channel.Where{
				Name: eq(hidePlain.Name),
				Not:  &channel.Where{Calculated: new(true)},
			})).Error().To(MatchError(access.ErrDenied))
			Expect(ask(ctx, channel.Where{
				Name: eq(hideOutput.Name),
				Or: []channel.Where{
					{Calculated: new(false)},
					{Key: &filter.Set[channel.Key]{In: []channel.Key{readPlain.Key()}}},
				},
			})).Error().To(MatchError(access.ErrDenied))
		})

		It("Should give a reader the channels it can read", func(ctx SpecContext) {
			Expect(ask(ctx, channel.Where{
				Name:       eq(readOutput.Name),
				Calculated: new(true),
			})).To(ConsistOf(readOutput.Key()))
			Expect(ask(ctx, channel.Where{
				Name:       eq(readPlain.Name),
				Calculated: new(true),
			})).To(BeEmpty())
			Expect(ask(ctx, channel.Where{
				Name:       eq(readOutput.Name),
				Not:        &channel.Where{Calculated: new(false)},
				Calculated: new(true),
			})).To(ConsistOf(readOutput.Key()))
		})

		It("Should keep the old meaning of virtual for a reader", func(
			ctx SpecContext,
		) {
			Expect(ask(ctx, channel.Where{
				Name:    eq(readOutput.Name),
				Virtual: new(true),
			})).To(BeEmpty())
			Expect(ask(ctx, channel.Where{
				Name:    eq(readOutput.Name),
				Virtual: new(false),
			})).To(ConsistOf(readOutput.Key()))
			Expect(ask(ctx, channel.Where{
				Name:    eq(readPlain.Name),
				Virtual: new(true),
			})).To(ConsistOf(readPlain.Key()))
		})
	})
})
