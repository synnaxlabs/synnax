// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channelv1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Clash upgrade breaker", func() {
	It("Should write the new name into text that v0.62 lexed with dashed names off",
		func(ctx SpecContext) {
			n := testutil.NewNode(ctx)
			stored := storeV062(
				ctx,
				n,
				[]string{groupv1.MetricsFolder, "less"},
				func(cs []channelv1.Channel) { cs[1].Expression = "return Metrics-1" },
				folder(groupv1.MetricsFolder),
			)
			arcs := storeArcs(ctx, n, []string{"x := Metrics-1"}, nil)
			// The layer opens with name validation on, the default, so v0.62 read
			// "Metrics-1" as Metrics minus 1.
			MustOpen(svctestutil.OpenLayer(ctx, n))
			Expect(namesOf(ctx, n, stored)[stored[0].Key()]).To(Equal("Metrics_2"))
			Expect(
				expressionOf(ctx, n, stored[1].Key()),
			).To(Equal("return Metrics_2-1"))
			Expect(sourceOf(ctx, n, arcs[0])).To(Equal("x := Metrics_2-1"))
		})

	It("Should keep a calculation on the channel that v0.62 bound when a channel in "+
		"another folder holds the old name", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		plant := folder("Plant")
		stored := storeV062(
			ctx,
			n,
			[]string{groupv1.MetricsFolder, groupv1.MetricsFolder, "doubled"},
			func(cs []channelv1.Channel) {
				cs[0].Group = &plant.Key
				cs[2].Expression = "return Metrics * 2"
			},
			plant,
			folder(groupv1.MetricsFolder),
		)
		// A v0.62 Core bound "Metrics" to its last channel in key order: the
		// top-level one.
		bound, other := stored[1].Key(), stored[0].Key()
		Expect(bound).To(BeNumerically(">", other))
		l := MustOpen(svctestutil.OpenLayer(ctx, n))
		names := namesOf(ctx, n, stored)
		Expect(names[bound]).To(Equal("Metrics_2"))
		Expect(names[other]).To(Equal(groupv1.MetricsFolder))
		var calc channel.Channel
		Expect(gorp.NewRetrieve[channel.Key, channel.Channel]().
			Where(gorp.MatchKeys[channel.Key, channel.Channel](stored[2].Key())).
			Entry(&calc).
			Exec(ctx, n.DB)).To(Succeed())
		Expect(l.Channel.CalculationInputs(ctx, n.DB, []channel.Channel{calc})).
			To(ConsistOf(bound))
	})

	It("Should not report a folder clash for a Metrics channel when the store held no "+
		"folder of the name", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		storeV062(ctx, n, []string{groupv1.MetricsFolder}, func([]channelv1.Channel) {})
		MustOpen(svctestutil.OpenLayer(ctx, n))
		Expect(renameStatuses(ctx, n)["Metrics_2"].Description).
			ToNot(ContainSubstring("because a folder beside it held the same name"))
	})

	It("Should report a repeat for a second top-level Metrics folder",
		func(ctx SpecContext) {
			n := testutil.NewNode(ctx)
			first, second := folder(
				groupv1.MetricsFolder,
			), folder(
				groupv1.MetricsFolder,
			)
			first.Key = uuid.MustParse("00000000-0000-4000-8000-000000000001")
			second.Key = uuid.MustParse("00000000-0000-4000-8000-000000000002")
			storeV062(ctx, n, nil, func([]channelv1.Channel) {}, first, second)
			MustOpen(svctestutil.OpenLayer(ctx, n))
			Expect(folderName(ctx, n, second.Key)).To(Equal("Metrics_2"))
			Expect(renameStatuses(ctx, n)["Metrics_2"].Description).To(Equal(
				`The upgrade renamed folder "Metrics" to "Metrics_2", because ` +
					"another folder beside it held the same name. Update each " +
					"reference to the old name outside Synnax.",
			))
		})
})
