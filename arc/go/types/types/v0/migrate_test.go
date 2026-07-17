// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/types/types/v0"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateFunctionProperties", func() {
	It("Should carry inputs and outputs while dropping the removed config", func(ctx SpecContext) {
		migrated := MustSucceed(v0.MigrateFunctionProperties(ctx, v0.FunctionProperties{
			Inputs:  v0.Params{{Name: "in"}},
			Outputs: v0.Params{{Name: "out"}},
			Config:  v0.Params{{Name: "cfg"}},
		}))
		Expect(migrated.Inputs).To(HaveLen(1))
		Expect(migrated.Inputs[0].Name).To(Equal("in"))
		Expect(migrated.Outputs).To(HaveLen(1))
		Expect(migrated.Outputs[0].Name).To(Equal("out"))
	})
})
