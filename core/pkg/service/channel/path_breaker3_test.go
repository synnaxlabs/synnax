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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	dtestutil "github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Sibling breaker 3", func() {
	It("Should name the position of a calculated channel whose adopted index a "+
		"folder's name refuses", func(ctx SpecContext) {
		dist := dtestutil.NewNode(ctx)
		cfg, groups := serviceConfig(ctx, dist, dist.DB)
		svc := MustOpen(channel.OpenService(ctx, cfg))
		db := dist.DB
		name := UniqueChannelName()
		orphan := channel.Channel{
			Name:        name + "_time",
			DataType:    telem.TimestampT,
			IsIndex:     true,
			Virtual:     true,
			Leaseholder: node.KeyFree,
		}
		parent := group.Group{Name: UniqueChannelName(), Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := svc.NewWriter(tx).Create(ctx, &orphan); err != nil {
				return err
			}
			gw := groups.NewWriter(tx)
			if err := gw.Create(ctx, &parent); err != nil {
				return err
			}
			taken := group.Group{
				Name:   orphan.Name,
				Scope:  resource.TypeChannel,
				Parent: new(group.ResourceID(parent.Key)),
			}
			return gw.Create(ctx, &taken)
		})).To(Succeed())
		chs := []channel.Channel{
			{Name: UniqueChannelName(), DataType: telem.Float32T, Virtual: true},
			{
				Name:       name,
				DataType:   telem.Float64T,
				Expression: "return 1.0",
				Group:      &parent.Key,
			},
		}
		err := db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).CreateMany(ctx, &chs)
		})
		Expect(err).To(MatchError(validate.ErrValidation))
		pe, ok := errors.AsType[validate.PathError](err)
		Expect(ok).To(BeTrue(), "not a path error: %v", err)
		Expect(pe.Path[0]).To(Equal("1"), err.Error())
	})
})
