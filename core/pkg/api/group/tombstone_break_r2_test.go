// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apigroup "github.com/synnaxlabs/synnax/pkg/api/group"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Tombstone break r2", Ordered, ContinueOnFailure, func() {
	var proj project.Project
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		proj = project.Project{Name: "tombstone-break-r2"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Project.NewWriter(tx).Create(ctx, &proj)
		})).To(Succeed())
	})
	tombstone := func(ctx SpecContext, g group.Key) channel.Key {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
			Group:    &g,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		var value schematic.ValueNodeConfig
		value.SetDefaults()
		value.Channel = ch.Key()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Schematic.NewWriter(tx).Create(ctx, &schematic.Schematic{
				Name:    "pid",
				Project: &proj.Key,
				Nodes: []schematic.Node{{
					Key:    "value",
					Config: schematic.NodeConfig{Variant: value},
				}},
			})
		})).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Channel.NewWriter(tx).Delete(ctx, ch.Key(), false)
		})).To(Succeed())
		return ch.Key()
	}
	tombstoned := func(ctx SpecContext, k channel.Key) bool {
		GinkgoHelper()
		var ts []channel.Tombstone
		Expect(gorp.NewRetrieve[channel.Key, channel.Tombstone]().
			Entries(&ts).
			Exec(ctx, db)).To(Succeed())
		for _, t := range ts {
			if t.Key == k {
				return true
			}
		}
		return false
	}

	It("Should delete a nested folder that holds a tombstone with the grants of an "+
		"empty one", func(ctx SpecContext) {
		parent := newGroup(ctx, resource.TypeChannel, nil)
		g := newGroup(ctx, resource.TypeChannel, &parent)
		dead := tombstone(ctx, g.Key)
		u := newUser(
			ctx,
			grant(remove, g.ResourceID()),
			grant(update, parent.ResourceID()),
		)
		MustSucceed(
			del(AuthedCtx(ctx, u), apigroup.DeleteRequest{Keys: []group.Key{g.Key}}),
		)
		Expect(exists(ctx, g.Key)).To(BeFalse())
		Expect(tombstoned(ctx, dead)).To(BeFalse())
	})

	It("Should delete a top folder that holds a tombstone with delete on it alone",
		func(ctx SpecContext) {
			g := newGroup(ctx, resource.TypeChannel, nil)
			dead := tombstone(ctx, g.Key)
			u := newUser(ctx, grant(remove, g.ResourceID()))
			MustSucceed(
				del(
					AuthedCtx(ctx, u),
					apigroup.DeleteRequest{Keys: []group.Key{g.Key}},
				),
			)
			Expect(tombstoned(ctx, dead)).To(BeFalse())
		})
})
