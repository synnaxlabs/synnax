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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	v1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/resource/rename"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("NameRuleMigration", func() {
	var db *gorp.DB
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })
	channel := func(local v1.LocalKey, name string, group *uuid.UUID) v1.Channel {
		return v1.Channel{
			Name:        name,
			Leaseholder: node.KeyFree,
			LocalKey:    local,
			Virtual:     true,
			Group:       group,
		}
	}
	// names creates channels, runs the migration, and returns the name of each channel
	// in the order given.
	names := func(ctx SpecContext, channels ...v1.Channel) []string {
		GinkgoHelper()
		Expect(gorp.NewCreate[v1.Key, v1.Channel]().
			Entries(&channels).
			Exec(ctx, db)).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "channel",
			Migrations: []gorp.Migration{v1.NameRuleMigration},
		})).To(Succeed())
		out := make([]string, len(channels))
		for i, c := range channels {
			var stored v1.Channel
			Expect(gorp.NewRetrieve[v1.Key, v1.Channel]().
				Where(gorp.MatchKeys[v1.Key, v1.Channel](c.Key())).
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			out[i] = stored.Name
		}
		return out
	}

	It("Should never move a legal name", func(ctx SpecContext) {
		Expect(names(
			ctx,
			channel(1, "PT-FU-01", nil),
			channel(2, "Stand A", nil),
			channel(3, "pt_01", nil),
		)).To(Equal([]string{"PT-FU-01", "Stand A", "pt_01"}))
		Expect(rename.Staged(db)).To(BeEmpty())
	})

	It("Should rewrite each name that breaks the rule", func(ctx SpecContext) {
		empty := channel(5, "\u200B", nil)
		Expect(names(
			ctx,
			channel(1, "a.b", nil),
			channel(2, "a\u200Bb\u00A0c", nil),
			channel(3, "p\xfft", nil),
			channel(4, "e\u0301", nil),
			empty,
		)).To(Equal([]string{
			"a_b",
			"abc",
			"p\uFFFDt",
			"\u00E9",
			"channel_" + empty.Key().String(),
		}))
	})

	It("Should give a rewritten name the first suffix free in the Core", func(
		ctx SpecContext,
	) {
		g := uuid.New()
		Expect(names(
			ctx,
			channel(1, "a_b", &g),
			channel(2, "a_b_2", nil),
			channel(3, "a.b", nil),
		)).To(Equal([]string{"a_b", "a_b_2", "a_b_3"}))
	})

	It("Should keep a name for the lowest key in a group", func(ctx SpecContext) {
		g := uuid.New()
		Expect(names(
			ctx,
			channel(1, "pt", &g),
			channel(2, "pt", &g),
			channel(3, "pt", nil),
		)).To(Equal([]string{"pt", "pt_2", "pt"}))
	})

	It("Should compare names in one group after NFC", func(ctx SpecContext) {
		Expect(names(
			ctx,
			channel(1, "\u00E9", nil),
			channel(2, "e\u0301", nil),
		)).To(Equal([]string{"\u00E9", "\u00E9_2"}))
	})

	It("Should keep the name of a built-in over a lower key", func(ctx SpecContext) {
		builtIn := channel(2, "sy_task_cmd", nil)
		builtIn.Internal = true
		Expect(names(ctx, channel(1, "sy_task_cmd", nil), builtIn)).
			To(Equal([]string{"sy_task_cmd_2", "sy_task_cmd"}))
	})

	It("Should stage each rename", func(ctx SpecContext) {
		renamed := channel(2, "a.b", nil)
		names(ctx, channel(1, "pt", nil), renamed)
		Expect(rename.Staged(db)).To(Equal([]rename.Rename{{
			Resource: renamed.ResourceID(),
			From:     "a.b",
			To:       "a_b",
		}}))
	})
})
