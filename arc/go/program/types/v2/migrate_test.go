// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	irv1 "github.com/synnaxlabs/arc/ir/types/v1"
	v1 "github.com/synnaxlabs/arc/program/types/v1"
	v2 "github.com/synnaxlabs/arc/program/types/v2"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateProgram", func() {
	It("Should carry the IR into the next version", func(ctx SpecContext) {
		migrated := MustSucceed(v2.MigrateProgram(ctx, v1.Program{
			IR: irv1.IR{Functions: irv1.Functions{{Key: "f"}}},
		}))
		Expect(migrated.IR.Functions).To(HaveLen(1))
		Expect(migrated.IR.Functions[0].Key).To(Equal("f"))
	})
})
