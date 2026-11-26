// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	v1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	resource "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("NameRuleMigration breaker", func() {
	var db *gorp.DB
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })
	channel := func(local v1.LocalKey, name string, internal bool) v1.Channel {
		return v1.Channel{
			Name:        name,
			Leaseholder: node.KeyFree,
			LocalKey:    local,
			LocalIndex:  local,
			IsIndex:     true,
			Virtual:     true,
			Internal:    internal,
		}
	}
	// run runs the migration once and returns the name of each channel by local key.
	run := func(ctx context.Context) map[v1.LocalKey]string {
		GinkgoHelper()
		tx := db.OpenTx()
		Expect(
			v1.NameRuleMigration.Run(ctx, tx, alamos.Instrumentation{}),
		).To(Succeed())
		Expect(tx.Commit(ctx)).To(Succeed())
		Expect(tx.Close()).To(Succeed())
		var stored []v1.Channel
		Expect(gorp.NewRetrieve[v1.Key, v1.Channel]().
			Entries(&stored).
			Exec(ctx, db)).To(Succeed())
		names := make(map[v1.LocalKey]string, len(stored))
		for _, c := range stored {
			names[c.LocalKey] = c.Name
		}
		return names
	}
	store := func(ctx context.Context, channels ...v1.Channel) {
		GinkgoHelper()
		Expect(gorp.WrapWriter[v1.Key, v1.Channel](db).Set(ctx, channels...)).
			To(Succeed())
	}

	It("Should move a user channel off the NFC form of a built-in name that is not "+
		"in NFC", func(ctx SpecContext) {
		store(
			ctx,
			channel(1, "caf\u00E9", false),
			channel(9, "cafe\u0301", true),
		)
		names := run(ctx)
		Expect(names[9]).To(Equal("cafe\u0301"))
		Expect(MustSucceed(resource.StagedRenames(db))).To(Equal([]resource.Rename{
			{ID: channel(1, "", false).ResourceID(), From: "caf\u00E9"},
		}))
		Expect(names[1]).To(
			Equal("caf\u00E9_2"),
			"user channel %+q, built-in %+q",
			names[1],
			names[9],
		)
	})
})
