// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package constraint_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/constraint"
)

var _ = Describe("Constraint", func() {
	It("Should return ErrInvalidConstraintKind if the constraint kind is invalid", func() {
		Expect(constraint.Constraint{}.Enforce(ctx, params)).Error().
			To(MatchError(constraint.ErrInvalidKind))
	})

})
