// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package impact_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apiimpact "github.com/synnaxlabs/synnax/pkg/api/impact"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/impact"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Mixed denial check", func() {
	It(
		"Should name the caller's own channel and not the hidden channel that a pick adds",
		func(ctx SpecContext) {
			node := testutil.NewNode(ctx)
			layer := MustOpen(svctestutil.OpenLayer(ctx, node))
			svc := MustSucceed(apiimpact.NewService(apiimpact.ServiceConfig{
				RBAC:   layer.RBAC,
				Impact: layer.Impact,
			}))
			binder := MustSucceed(bind.NewService(bind.ServiceConfig{
				DB:      node.DB,
				RBAC:    layer.RBAC,
				Held:    apiauth.HeldTables(),
				Mirrors: channel.MirrorTables(),
			}))
			apply := svc.ApplyHandler(binder)
			index := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.TimestampT,
				IsIndex:  true,
			}
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Channel.NewWriter(tx).Create(ctx, &index)
			})).To(Succeed())
			data := channel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Float32T,
				LocalIndex: index.LocalKey,
			}
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Channel.NewWriter(tx).Create(ctx, &data)
			})).To(Succeed())
			u := MustSucceed(rbactestutil.CreateUser(
				ctx,
				node.DB,
				layer.RBAC,
				layer.User,
				user.User{Name: uuid.New().String()},
			))
			// The caller holds no grant: it names its own channel and sees neither.
			_, err := apply(AuthedCtx(ctx, u), apiimpact.ApplyRequest{
				Change: impact.Change{Variant: impact.DeleteChannelsChange{
					Keys: []channel.Key{index.Key()},
				}},
				Hidden: new(impact.ChoiceDelete),
			})
			Expect(err).To(MatchError(access.ErrDenied))
			Expect(err).To(MatchError(ContainSubstring(
				"cannot delete %s",
				channel.ResourceID(index.Key()),
			)))
			Expect(err.Error()).ToNot(ContainSubstring(data.Key().String()))
		},
	)
})
