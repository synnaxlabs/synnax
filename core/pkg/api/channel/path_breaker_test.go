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
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Path breaker", func() {
	It("Should deny a channel named by path that the subject cannot retrieve, as by "+
		"name", func(ctx SpecContext) {
		dir := createChannelGroup(ctx, UniqueChannelName())
		allowed := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
			Group:    &dir.Key,
		}
		blocked := allowed
		blocked.Name = UniqueChannelName()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := channelSvc.NewWriter(tx)
			if err := w.Create(ctx, &allowed); err != nil {
				return err
			}
			return w.Create(ctx, &blocked)
		})).To(Succeed())
		reader := MustSucceed(rbactestutil.CreateUser(
			ctx,
			db,
			rbacSvc,
			userSvc,
			user.User{Name: "path-reader-" + uuid.New().String()},
		))
		// The folder is visible, so the path reaches both channels.
		grantOn(
			ctx,
			reader.ResourceID(),
			[]access.Action{access.ActionRetrieve},
			allowed.ResourceID(),
			dir.ResourceID(),
		)
		denied := access.Denied(
			reader.ResourceID(),
			access.ActionRetrieve,
			blocked.ResourceID(),
		).Error()
		Expect(retrieveChannels(
			AuthedCtx(ctx, reader),
			apichannel.RetrieveRequest{Where: &channel.Where{
				Name: &filter.String{In: []string{allowed.Name, blocked.Name}},
			}},
		)).Error().To(MatchError(denied))
		Expect(retrieveChannels(
			AuthedCtx(ctx, reader),
			apichannel.RetrieveRequest{Where: &channel.Where{
				Path: &filter.Set[string]{In: []string{
					dir.Name + "." + allowed.Name,
					dir.Name + "." + blocked.Name,
				}},
			}},
		)).Error().To(MatchError(denied))
	})
})
