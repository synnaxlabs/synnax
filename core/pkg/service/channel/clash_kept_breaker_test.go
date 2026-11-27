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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Kept name clash breaker", func() {
	var (
		db  *gorp.DB
		svc *channel.Service
	)
	BeforeEach(func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		db = node.DB
		cfg, _ := serviceConfig(ctx, node, node.DB)
		cfg.ValidateNames = new(false)
		svc = MustOpen(channel.OpenService(ctx, cfg))
	})
	virtual := func(name string) channel.Channel {
		return channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
	}
	createMany := func(ctx SpecContext, chs []channel.Channel) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).CreateMany(ctx, &chs)
		})
	}

	It("Should use no channel key on a create into a name that a kept channel holds "+
		"in another normal form", func(ctx SpecContext) {
		base := UniqueChannelName()
		composed, decomposed := base+"_caf\u00e9", base+"_cafe\u0301"
		kept := virtual(composed)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &kept)
		})).To(Succeed())
		// A v0.62 Core stored a name as the client sent it, and the upgrade keeps it.
		kept.Name = decomposed
		Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).Set(ctx, kept)).
			To(Succeed())
		err := createMany(ctx, []channel.Channel{
			virtual(UniqueChannelName()),
			virtual(decomposed),
		})
		Expect(err).To(MatchError(ContainSubstring(
			fmt.Sprintf("channel with name '%s' already exists", composed),
		)))
		pathErr, ok := errors.AsType[validate.PathError](err)
		Expect(ok).To(BeTrue())
		Expect(pathErr.Path).To(Equal([]string{"1", "name"}))
		next := []channel.Channel{virtual(UniqueChannelName())}
		Expect(createMany(ctx, next)).To(Succeed())
		var stored channel.Channel
		Expect(svc.NewRetrieve().
			Where(channel.MatchNames(next[0].Name)).
			Entry(&stored).
			Exec(ctx, db)).To(Succeed())
		Expect(stored.LocalKey).To(Equal(kept.LocalKey + 1))
	})
})
