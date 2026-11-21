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
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
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
})
