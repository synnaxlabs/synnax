// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apiimex "github.com/synnaxlabs/synnax/pkg/api/imex"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Graph Arc channel links access check", func() {
	var (
		ch         channel.Channel
		a          arc.Arc
		u          user.User
		exportFile func(ctx SpecContext) string
	)
	BeforeEach(func(ctx SpecContext) {
		ch = channel.Channel{
			Name:     "graph_link_" + uuid.New().String()[:8],
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		a = arc.Arc{Variant: arc.AutomationArc{Base: arc.Base{
			Name: "graph_link_" + uuid.New().String()[:8],
			Source: arc.Source{Variant: arc.GraphSource{Graph: graph.Graph{
				Nodes: graph.Nodes{{Key: "source"}},
				Inputs: map[string]msgpack.EncodedJSON{
					"source": {"type": "on", "channel": ch.Key()},
				},
			}}},
		}}}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return arcSvc.NewWriter(tx).Create(ctx, &a)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return arcSvc.NewWriter(tx).Delete(ctx, a.Base().Key)
			})).To(Succeed())
		})
		u = MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc, user.User{
			Name: "graph-exporter-" + uuid.New().String()[:8],
		}))
		grantOn(ctx, u.ResourceID(), access.ActionRetrieve, a.ResourceID())
		imexSvc := MustSucceed(apiimex.NewService(apiimex.ServiceConfig{
			ImEx:    svc.ImEx,
			Channel: channelSvc,
		}))
		export := bind.Read(binder, imexSvc.Export)
		exportFile = func(ctx SpecContext) string {
			GinkgoHelper()
			env := MustSucceed(export(
				AuthedCtx(ctx, u),
				apiimex.ExportRequest(a.ResourceID()),
			))
			return string(MustSucceed(json.Marshal(env)))
		}
	})

	It("Should name the channel when the exporter can retrieve it", func(
		ctx SpecContext,
	) {
		grantOn(
			ctx,
			u.ResourceID(),
			access.ActionRetrieve,
			channel.ResourceID(ch.Key()),
		)
		Expect(exportFile(ctx)).To(ContainSubstring(ch.Name))
	})

	It("Should not name a channel that the exporter cannot retrieve", func(
		ctx SpecContext,
	) {
		file := exportFile(ctx)
		Expect(file).ToNot(ContainSubstring(ch.Name))
		Expect(file).To(ContainSubstring(`"channel":%d`, ch.Key()))
	})
})
