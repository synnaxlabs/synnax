// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package imex_test

import (
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apiimex "github.com/synnaxlabs/synnax/pkg/api/imex"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/notation"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Channel links access check", func() {
	It("Should not name a channel that the exporter cannot retrieve", func(
		ctx SpecContext,
	) {
		l := defaults.Fill(log.Log{
			Name:    "Hidden links",
			Project: new(proj.Key),
			Channels: []log.ChannelEntry{defaults.Fill(log.ChannelEntry{
				Channel:  ch,
				Notation: notation.NotationStandard,
			})},
		})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return logSvc.NewWriter(tx).Create(ctx, &l)
		})).To(Succeed())
		u := MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc, user.User{
			Name: "exporter-" + uuid.New().String(),
		}))
		MustSucceed(rbactestutil.Grant(
			ctx,
			db,
			rbacSvc,
			userSvc,
			u.Key,
			[]access.Action{access.ActionRetrieve},
			log.ResourceID(l.Key),
		))
		env := MustSucceed(
			export(subjectCtx(ctx, u), apiimex.ExportRequest(log.ResourceID(l.Key))),
		)
		file := string(MustSucceed(json.Marshal(env)))
		Expect(file).ToNot(ContainSubstring(chName))
	})
})
