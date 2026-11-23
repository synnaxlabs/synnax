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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	channelv1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Arc list check", func() {
	It("Should fill the channel lists of a stored Arc when the Core starts", func(
		ctx SpecContext,
	) {
		n := testutil.NewNode(ctx)
		stored := storeV062(
			ctx,
			n,
			[]string{"lc_in", "lc_out"},
			func([]channelv1.Channel) {},
		)
		keys := storeArcs(ctx, n, []string{
			"func pass() {v f32} f32 { return v }\nlc_in -> pass() -> lc_out",
		}, nil)
		MustOpen(svctestutil.OpenLayer(ctx, n))
		Eventually(func(g Gomega) {
			var a arc.Arc
			g.Expect(gorp.NewRetrieve[arc.Key, arc.Arc]().
				Where(gorp.MatchKeys[arc.Key, arc.Arc](keys[0])).
				Entry(&a).
				Exec(ctx, n.DB)).To(Succeed())
			g.Expect(a.Base().Reads).To(ConsistOf(stored[0].Key()))
			g.Expect(a.Base().Writes).To(ConsistOf(stored[1].Key()))
		}).WithTimeout(10 * time.Second).Should(Succeed())
	})
})
