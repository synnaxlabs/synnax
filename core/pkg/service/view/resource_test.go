// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package view_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/view"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resources", func() {
	var svc *view.Service
	BeforeEach(func(ctx SpecContext) {
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		svc = MustOpen(view.OpenService(ctx, view.ServiceConfig{DB: db, Ontology: otg}))
	})

	It("Should name each view by its base name", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		v := rangeView("Burns")
		Expect(svc.NewWriter(tx).Create(ctx, &v)).To(Succeed())
		Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{v.ResourceID()})).
			To(Equal([]resource.Resource{{ID: v.ResourceID(), Name: "Burns"}}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(svc.Resources().Retrieve(
			ctx,
			nil,
			[]resource.ID{{Type: resource.TypeView, Key: "pump"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid view key "pump"`)),
		))
	})
})
