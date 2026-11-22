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
	"github.com/synnaxlabs/gorp"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Calculation status checks", func() {
	// A1: a subject that cannot read a calculation sees its output as a plain virtual
	// channel, so the status must not name what the calculation reads.
	It("Should join no status for a calculation hidden from the subject", func(
		ctx SpecContext,
	) {
		newUser := func(prefix string, objects ...resource.ID) user.User {
			r := MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc,
				user.User{Name: prefix + uuid.New().String()}))
			grantOn(ctx, r.ResourceID(), access.AllActions, objects...)
			return r
		}
		owner := newUser(
			"status-owner-",
			resource.ID{Type: resource.TypeChannel},
			resource.ID{Type: resource.TypeCalculation},
			resource.ID{Type: resource.TypeFramer},
			resource.ID{Type: resource.TypeGroup},
		)
		input := createChannel(ctx, telem.Float32T)
		ch := MustSucceed(createChannels(
			AuthedCtx(ctx, owner),
			apichannel.CreateRequest{Channels: []apichannel.Channel{{
				Name:       UniqueChannelName(),
				DataType:   telem.Float32T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s * 2", input.Name),
			}}},
		)).Channels[0]
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Delete(ctx, input.Key(), false)
		})).To(Succeed())
		retrieve := func(subject user.User) (apichannel.Channel, error) {
			res, err := retrieveChannels(AuthedCtx(ctx, subject),
				apichannel.RetrieveRequest{
					Where: &channel.Where{
						Key: &filter.Set[channel.Key]{Eq: &ch.Key},
					},
					IncludeStatus: true,
				})
			if err != nil || len(res.Channels) != 1 {
				return apichannel.Channel{}, err
			}
			return res.Channels[0], nil
		}
		Eventually(func(g Gomega) {
			got, err := retrieve(owner)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(got.Status).ToNot(BeNil())
		}).Should(Succeed())
		reader := newUser("status-reader-", resource.ID{Type: resource.TypeChannel})
		got := MustSucceed(retrieve(reader))
		Expect(got.Expression).To(BeEmpty())
		Expect(got.Status).To(BeNil())
	})

	It("Should join no status for a calculation whose input the subject cannot read",
		func(ctx SpecContext) {
			owner := MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc,
				user.User{Name: "status-input-owner-" + uuid.New().String()}))
			grantOn(ctx, owner.ResourceID(), access.AllActions,
				resource.ID{Type: resource.TypeChannel},
				resource.ID{Type: resource.TypeCalculation},
				resource.ID{Type: resource.TypeFramer},
			)
			input := createChannel(ctx, telem.Float32T)
			ch := MustSucceed(createChannels(
				AuthedCtx(ctx, owner),
				apichannel.CreateRequest{Channels: []apichannel.Channel{{
					Name:       UniqueChannelName(),
					DataType:   telem.Float32T,
					Virtual:    true,
					Expression: fmt.Sprintf("return %s * 2", input.Name),
				}}},
			)).Channels[0]
			calcs := MustSucceed(layer.Calculation.Outputs(ctx, db, ch.Key))
			// A runtime error names what the calculation reads.
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return statusSvc.NewWriter(tx).Set(ctx, &apichannel.Status{
					Owner:       new(calculation.ResourceID(calcs[ch.Key].Key)),
					Name:        ch.Name,
					Variant:     status.VariantError,
					Message:     "Failed to run calculation",
					Description: "cannot read " + input.Name,
					Time:        telem.Now(),
				})
			})).To(Succeed())
			reader := MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc,
				user.User{Name: "status-input-reader-" + uuid.New().String()}))
			grantOn(ctx, reader.ResourceID(), []access.Action{access.ActionRetrieve},
				channel.ResourceID(ch.Key),
				resource.ID{Type: resource.TypeCalculation},
			)
			res := MustSucceed(retrieveChannels(AuthedCtx(ctx, reader),
				apichannel.RetrieveRequest{
					Where: &channel.Where{
						Key: &filter.Set[channel.Key]{Eq: &ch.Key},
					},
					IncludeStatus: true,
				}))
			Expect(res.Channels).To(HaveLen(1))
			Expect(res.Channels[0].Expression).To(BeEmpty())
			Expect(res.Channels[0].Status).To(BeNil())
		},
	)

	// The old API calls the output of a hidden calculation calculated, so its virtual
	// filter must leave that output out for the same subject.
	It("Should leave a hidden calculation's output out of a virtual retrieve", func(
		ctx SpecContext,
	) {
		owner := MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc,
			user.User{Name: "virtual-owner-" + uuid.New().String()}))
		grantOn(ctx, owner.ResourceID(), access.AllActions,
			resource.ID{Type: resource.TypeChannel},
			resource.ID{Type: resource.TypeCalculation},
			resource.ID{Type: resource.TypeFramer},
			resource.ID{Type: resource.TypeGroup},
		)
		input := createChannel(ctx, telem.Float32T)
		ch := MustSucceed(createChannels(
			AuthedCtx(ctx, owner),
			apichannel.CreateRequest{Channels: []apichannel.Channel{{
				Name:       UniqueChannelName(),
				DataType:   telem.Float32T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s * 2", input.Name),
			}}},
		)).Channels[0]
		reader := MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc,
			user.User{Name: "virtual-reader-" + uuid.New().String()}))
		grantOn(ctx, reader.ResourceID(), []access.Action{access.ActionRetrieve},
			resource.ID{Type: resource.TypeChannel})
		virtual := true
		res := MustSucceed(retrieveChannels(AuthedCtx(ctx, reader),
			apichannel.RetrieveRequest{Where: &channel.Where{
				Key:     &filter.Set[channel.Key]{Eq: &ch.Key},
				Virtual: &virtual,
			}}))
		Expect(res.Channels).To(BeEmpty())
		virtual = false
		res = MustSucceed(retrieveChannels(AuthedCtx(ctx, reader),
			apichannel.RetrieveRequest{Where: &channel.Where{
				Key:     &filter.Set[channel.Key]{Eq: &ch.Key},
				Virtual: &virtual,
			}}))
		Expect(res.Channels).To(ConsistOf(SatisfyAll(
			HaveField("Key", ch.Key),
			HaveField("Expression", BeEmpty()),
			HaveField("Calculated", BeTrue()),
		)))
	})
})
