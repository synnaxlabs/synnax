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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apicalculation "github.com/synnaxlabs/synnax/pkg/api/calculation"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Calculation diagnostics check", func() {
	It("Should keep the diagnostics of a create that does not compile", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grant(ctx, access.AllActions, u.ResourceID(), anyChannel, anyCalculation)
		out := createVirtual(ctx)
		_, err := create(AuthedCtx(ctx, u), apicalculation.CreateRequest{
			Calculations: []apicalculation.Input{{
				Calculation: calculation.Calculation{
					Output:     out.Key(),
					Expression: "return missing * 2",
				},
			}},
		})
		Expect(err).To(MatchError(ContainSubstring("missing")))
		_, ok := errors.AsType[*diagnostics.Diagnostics](err)
		Expect(ok).To(BeTrue(), "%v holds diagnostics", err)
		pathErr := MustBeOk(errors.AsType[validate.PathError](err))
		Expect(pathErr.Path).To(Equal([]string{"calculations", "0", "expression"}))
	})
})
