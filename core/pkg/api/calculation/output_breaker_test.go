// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apicalculation "github.com/synnaxlabs/synnax/pkg/api/calculation"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Calculation output breaker", func() {
	It("Should refuse the index of a calculated channel as an output", func(
		ctx SpecContext,
	) {
		input := createVirtual(ctx)
		calc := channel.Channel{
			Name:       "calc_" + uuid.New().String()[:8],
			DataType:   telem.Float32T,
			Expression: "return " + input.Name + " * 2",
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Channel.NewWriter(tx).Create(ctx, &calc)
		})).To(Succeed())
		// The calculation of calc writes this index with each sample.
		Expect(calc.Index()).ToNot(BeZero())
		u := createUser(ctx)
		grant(ctx, access.AllActions, u.ResourceID(), anyCalculation, anyChannel)
		Expect(create(AuthedCtx(ctx, u), apicalculation.CreateRequest{
			Calculations: []calculation.Calculation{
				{Output: calc.Index(), Expression: "return 2"},
			},
		})).Error().To(MatchError(validate.ErrValidation))
	})

	It("Should refuse a channel that the Core writes as an output", func(
		ctx SpecContext,
	) {
		var signal channel.Channel
		Expect(svc.Channel.NewRetrieve().
			Where(channel.MatchNames("sy_calculation_set")).
			Entry(&signal).
			Exec(ctx, db)).To(Succeed())
		Expect(signal.Virtual).To(BeTrue())
		Expect(svc.Channel.IsBuiltIn(signal.Key())).To(BeTrue())
		u := createUser(ctx)
		grant(ctx, access.AllActions, u.ResourceID(), anyCalculation, anyChannel)
		Expect(create(AuthedCtx(ctx, u), apicalculation.CreateRequest{
			Calculations: []calculation.Calculation{
				{Output: signal.Key(), Expression: "return 2"},
			},
		})).Error().To(MatchError(validate.ErrValidation))
	})
})
