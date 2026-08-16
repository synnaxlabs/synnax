// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/query"
)

var _ = Describe("Builtin", func() {
	It("Should retrieve the builtin root resource", func(ctx SpecContext) {
		var res ontology.Resource
		Expect(otg.NewRetrieve().WhereIDs(ontology.RootID).Entry(&res).Exec(ctx, tx)).
			To(Succeed())
		Expect(res.ID).To(Equal(ontology.RootID))
		Expect(res.Name).To(Equal("root"))
	})
	It(
		"Should return query.ErrNotFound for a nonexistent builtin resource",
		func(ctx SpecContext) {
			id := ontology.ID{Type: ontology.ResourceTypeBuiltin, Key: "nonexistent"}
			Expect(otg.NewWriter(tx).DefineResources(ctx, id)).To(Succeed())
			var res ontology.Resource
			Expect(otg.NewRetrieve().WhereIDs(id).Entry(&res).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
		},
	)
})
