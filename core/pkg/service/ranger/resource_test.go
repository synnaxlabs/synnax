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
})
