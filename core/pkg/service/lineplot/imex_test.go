// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ImEx", func() {
	Describe("Export", func() {
		It("Should export a line plot as a versioned envelope", func(ctx SpecContext) {
			lp := lineplot.LinePlot{Name: "exported"}
			Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &lp)).To(Succeed())
			env := MustSucceed(svc.Export(ctx, lineplot.OntologyID(lp.Key)))
			Expect(env.Version).To(Equal(lineplot.Version))
			Expect(env.Type).To(Equal("lineplot"))
			Expect(env.Name).To(Equal("exported"))

			decoded := MustSucceed(imex.Decode[lineplot.LinePlot](ctx, WireRoundTrip(env)))
			Expect(decoded.Name).To(Equal("exported"))
		})

		It("Should return not found for a missing key", func(ctx SpecContext) {
			id := ontology.ID{Type: ontology.ResourceTypeLineplot, Key: uuid.NewString()}
			Expect(svc.Export(ctx, id)).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should error on an invalid UUID key", func(ctx SpecContext) {
			id := ontology.ID{Type: ontology.ResourceTypeLineplot, Key: "not-a-uuid"}
			Expect(svc.Export(ctx, id)).Error().To(MatchError(ContainSubstring("UUID")))
		})
	})
})
