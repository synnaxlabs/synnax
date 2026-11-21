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
	// A8: the old API shows the output of a hidden calculation as a plain virtual
	// channel, so its virtual filter must match that output for the same subject.
	It("Should match the output of a hidden calculation as virtual", func(
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
		Expect(res.Channels).To(ConsistOf(SatisfyAll(
			HaveField("Key", ch.Key),
			HaveField("Expression", BeEmpty()),
		)))
	})
})
