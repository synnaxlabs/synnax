// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apisymbol "github.com/synnaxlabs/synnax/pkg/api/schematic/symbol"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("DeleteGroup of a channel folder", func() {
	It("Should remove the tombstones of the folder first", func(ctx SpecContext) {
		folder := group.Group{Name: "measurements", Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &folder)
		})).To(Succeed())
		ch := channel.Channel{
			Name:     "tombstoned",
			DataType: telem.Float32T,
			Virtual:  true,
			Group:    &folder.Key,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		proj := project.Project{Name: "tombstones"}
		var value schematic.ValueNodeConfig
		value.SetDefaults()
		value.Channel = ch.Key()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := projectSvc.NewWriter(tx).Create(ctx, &proj); err != nil {
				return err
			}
			return schematicSvc.NewWriter(tx).Create(ctx, &schematic.Schematic{
				Name:    "pid",
				Project: &proj.Key,
				Nodes: []schematic.Node{{
					Key:    "value",
					Config: schematic.NodeConfig{Variant: value},
				}},
			})
		})).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Delete(ctx, ch.Key(), false)
		})).To(Succeed())
		u := createUser(ctx)
		grantDeleteOn(ctx, u.ResourceID(), folder.ResourceID())
		grantOn(ctx, u.ResourceID(), access.ActionUpdate, folder.ResourceID())
		MustSucceed(deleteGroup(
			AuthedCtx(ctx, u),
			apisymbol.DeleteGroupRequest{Key: folder.Key},
		))
		Expect(groupSvc.NewRetrieve().
			Where(group.MatchKeys(folder.Key)).
			Exists(ctx, db)).To(BeFalse())
	})
})
