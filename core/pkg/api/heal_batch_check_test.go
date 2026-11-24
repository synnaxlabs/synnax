// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/batch"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Heal batch check", func() {
	It("Should name no hidden record in the denial of a heal that a batch rename makes",
		func(ctx SpecContext) {
			create := func(ch *channel.Channel) {
				GinkgoHelper()
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.Channel.NewWriter(tx).Create(ctx, ch)
				})).To(Succeed())
			}
			dead := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			create(&dead)
			proj := project.Project{Name: "heal-batch-check"}
			var value schematic.ValueNodeConfig
			value.SetDefaults()
			value.Channel = dead.Key()
			sch := schematic.Schematic{
				Name: "hidden",
				Nodes: []schematic.Node{{
					Key:    "value",
					Config: schematic.NodeConfig{Variant: value},
				}},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := svc.Project.NewWriter(tx).Create(ctx, &proj); err != nil {
					return err
				}
				sch.Project = &proj.Key
				return svc.Schematic.NewWriter(tx).Create(ctx, &sch)
			})).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Channel.NewWriter(tx).Delete(ctx, dead.Key(), false)
			})).To(Succeed())
			ch := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			create(&ch)
			// The subject may rename ch, and can see no schematic.
			u := newUser(ctx)
			MustSucceed(rbactestutil.Grant(
				ctx,
				db,
				svc.RBAC,
				svc.User,
				u.Key,
				[]access.Action{access.ActionUpdate, access.ActionRetrieve},
				channel.ResourceID(ch.Key()),
			))
			var req batch.ExecRequest
			Expect(json.Codec.Decode([]byte(fmt.Sprintf(`{"requests": [
				{"endpoint": "/channel/dispatch", "payload": {"key": %d, "actions": [
					{"type": "rename", "rename": {"name": %q}}
				]}}
			]}`, ch.Key(), dead.Name)), &req)).To(Succeed())
			_, err := exec(AuthedCtx(ctx, u), req)
			Expect(err).To(MatchError(access.ErrDenied))
			Expect(err).To(MatchError(fmt.Sprintf(
				"%s cannot change 1 record that this request changes and that it "+
					"cannot retrieve: access denied: auth error",
				u.ResourceID(),
			)))
			Expect(err.Error()).ToNot(ContainSubstring(sch.Key.String()))
		},
	)
})
