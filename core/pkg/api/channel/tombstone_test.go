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
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Tombstone", func() {
	It("Should delete a channel that a schematic shows and keep its path", func(
		ctx SpecContext,
	) {
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
		}
		proj := project.Project{Name: "tombstone"}
		var value schematic.ValueNodeConfig
		value.SetDefaults()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := channelSvc.NewWriter(tx).Create(ctx, &ch); err != nil {
				return err
			}
			if err := layer.Project.NewWriter(tx).Create(ctx, &proj); err != nil {
				return err
			}
			value.Channel = ch.Key()
			return layer.Schematic.NewWriter(tx).Create(ctx, &schematic.Schematic{
				Name:    "pid",
				Project: &proj.Key,
				Nodes: []schematic.Node{{
					Key:    "value",
					Config: schematic.NodeConfig{Variant: value},
				}},
			})
		})).To(Succeed())
		grantOn(
			ctx,
			author.ResourceID(),
			[]access.Action{access.ActionDelete},
			channel.ResourceID(ch.Key()),
		)
		MustSucceed(deleteChannels(
			AuthedCtx(ctx, author),
			apichannel.DeleteRequest{Keys: channel.Keys{ch.Key()}},
		))
		var tombstones []channel.Tombstone
		Expect(gorp.NewRetrieve[channel.Key, channel.Tombstone]().
			Where(gorp.MatchKeys[channel.Key, channel.Tombstone](ch.Key())).
			Entries(&tombstones).
			Exec(ctx, db)).To(Succeed())
		Expect(tombstones).To(ConsistOf(HaveField("Name", ch.Name)))
	})
})
