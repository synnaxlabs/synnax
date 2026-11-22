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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Ambiguity wire check", func() {
	It(
		"Should send a name that several channels hold as a multiple results error",
		func(
			ctx SpecContext,
		) {
			fresh := testutil.NewNode(ctx)
			cfg, groupSvc := serviceConfig(ctx, fresh, fresh.DB)
			stand := group.Group{Name: "stand", Scope: resource.TypeChannel}
			rig := group.Group{Name: "rig", Scope: resource.TypeChannel}
			Expect(fresh.DB.WithTx(ctx, func(tx gorp.Tx) error {
				for _, g := range []*group.Group{&stand, &rig} {
					if err := groupSvc.NewWriter(tx).Create(ctx, g); err != nil {
						return err
					}
				}
				return nil
			})).To(Succeed())
			stored := func(key channel.LocalKey, g *group.Key) channel.Channel {
				return channel.Channel{
					Name:        "pt",
					Leaseholder: node.KeyFree,
					LocalKey:    key,
					DataType:    telem.Float32T,
					Virtual:     true,
					Group:       g,
				}
			}
			createStored(ctx, fresh.DB, stored(7, &stand.Key), stored(8, &rig.Key))
			svc := MustOpen(channel.OpenService(ctx, cfg))
			codec := errors.NewCodec(query.ErrorProvider())
			_, err := svc.KeysByNames(ctx, fresh.DB, []string{"pt"})
			Expect(codec.Encode(err, false).Type).To(Equal("sy.query.multiple_results"))
			described := svc.DescribeAmbiguity(ctx, fresh.DB, err, func(
				context.Context,
			) (func(resource.ID) bool, error) {
				return func(resource.ID) bool { return true }, nil
			})
			Expect(
				codec.Encode(described, false).Type,
			).To(Equal("sy.query.multiple_results"))
		},
	)
})
