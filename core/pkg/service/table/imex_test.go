// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ImEx", func() {
	Describe("Export", func() {
		It("Should export a table as a versioned envelope", func(ctx SpecContext) {
			t := table.Table{Name: "exported"}
			Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &t)).To(Succeed())
			env := MustSucceed(svc.Export(ctx, table.OntologyID(t.Key)))
			Expect(env.Version).To(Equal(table.Version))
			Expect(env.Type).To(Equal("table"))
			Expect(env.Name).To(Equal("exported"))

			decoded := MustSucceed(imex.Decode[table.Table](ctx, WireRoundTrip(env)))
			Expect(decoded.Name).To(Equal("exported"))
		})

		It("Should return not found for a missing key", func(ctx SpecContext) {
			id := ontology.ID{Type: ontology.ResourceTypeTable, Key: uuid.NewString()}
			Expect(svc.Export(ctx, id)).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should error on an invalid UUID key", func(ctx SpecContext) {
			id := ontology.ID{Type: ontology.ResourceTypeTable, Key: "not-a-uuid"}
			Expect(svc.Export(ctx, id)).Error().To(MatchError(ContainSubstring("UUID")))
		})
	})
})
