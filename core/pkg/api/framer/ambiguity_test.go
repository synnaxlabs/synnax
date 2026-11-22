// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apiframer "github.com/synnaxlabs/synnax/pkg/api/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// A store from before the sibling rule can hold a name in several folders.
var _ = Describe("Delete by an ambiguous name", func() {
	It("Should refuse a name that several channels hold", func(ctx SpecContext) {
		folders := []*group.Group{
			{Name: UniqueChannelName(), Scope: resource.TypeChannel},
			{Name: UniqueChannelName(), Scope: resource.TypeChannel},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			for _, g := range folders {
				if err := svc.Group.NewWriter(tx).Create(ctx, g); err != nil {
					return err
				}
			}
			return nil
		})).To(Succeed())
		name := UniqueChannelName()
		holders := make([]channel.Channel, len(folders))
		for i, g := range folders {
			holders[i] = channel.Channel{
				Name:        name,
				Leaseholder: node.KeyFree,
				LocalKey:    channel.LocalKey(1_000_401 + i),
				DataType:    telem.Float32T,
				Virtual:     true,
				Group:       &g.Key,
			}
		}
		Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).Set(ctx, holders...)).
			To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).
				Delete(ctx, channel.KeysFromChannels(holders)...)).To(Succeed())
		})
		Expect(apiSvc.Delete(rootCtx(ctx), db, apiframer.DeleteRequest{
			Names:  []string{name},
			Bounds: telem.TimeRangeMax,
		})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("2 channels are named '"+name+"': '")),
		))
	})
})
