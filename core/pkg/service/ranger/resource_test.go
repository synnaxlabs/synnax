// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resources", func() {
	It("Should name each range by its name", func(ctx SpecContext) {
		openTx()
		r := ranger.Range{
			Name:      "Burn",
			TimeRange: telem.SecondTS.SpanRange(telem.Second),
		}
		Expect(w.Create(ctx, &r)).To(Succeed())
		Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{r.ResourceID()})).
			To(Equal([]resource.Resource{{ID: r.ResourceID(), Name: "Burn"}}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(svc.Resources().Retrieve(
			ctx,
			nil,
			[]resource.ID{{Type: resource.TypeRange, Key: "pump"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid range key "pump"`)),
		))
	})

	It("Should leave out an ID that names no range", func(ctx SpecContext) {
		openTx()
		r := ranger.Range{
			Name:      "Listed burn",
			TimeRange: telem.SecondTS.SpanRange(telem.Second),
		}
		Expect(w.Create(ctx, &r)).To(Succeed())
		missing := resource.ID{Type: resource.TypeRange, Key: uuid.New().String()}
		Expect(svc.Resources().Retrieve(
			ctx,
			tx,
			[]resource.ID{missing, r.ResourceID()},
		)).To(Equal([]resource.Resource{{ID: r.ResourceID(), Name: r.Name}}))
	})

	It("Should list each range by its name", func(ctx SpecContext) {
		openTx()
		r := ranger.Range{
			Name:      "Listed burn",
			TimeRange: telem.SecondTS.SpanRange(telem.Second),
		}
		Expect(w.Create(ctx, &r)).To(Succeed())
		Expect(svc.Resources().List(ctx, tx, 10000, nil)).To(ContainElement(
			resource.Resource{ID: r.ResourceID(), Name: r.Name},
		))
	})
})
