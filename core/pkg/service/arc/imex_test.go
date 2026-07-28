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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ImEx", func() {
	Describe("Export", func() {
		It("Should export an arc as a versioned envelope", func(ctx SpecContext) {
			a := arc.Arc{Name: "exported", Mode: arc.ModeText}
			Expect(svc.NewWriter(nil).Create(ctx, &a)).To(Succeed())
			env := MustSucceed(svc.Export(ctx, arc.OntologyID(a.Key)))
			Expect(env.Version).To(Equal(arc.Version))
			Expect(env.Type).To(Equal("arc"))
			Expect(env.Name).To(Equal("exported"))

			decoded := MustSucceed(imex.Decode[arc.Arc](ctx, WireRoundTrip(env)))
			Expect(decoded.Name).To(Equal("exported"))
			Expect(decoded.Mode).To(Equal(arc.ModeText))
		})

		It("Should return not found for a missing key", func(ctx SpecContext) {
			id := ontology.ID{Type: ontology.ResourceTypeArc, Key: uuid.NewString()}
			Expect(svc.Export(ctx, id)).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should error on an invalid UUID key", func(ctx SpecContext) {
			id := ontology.ID{Type: ontology.ResourceTypeArc, Key: "not-a-uuid"}
			Expect(svc.Export(ctx, id)).Error().To(MatchError(ContainSubstring("UUID")))
		})
	})
})
