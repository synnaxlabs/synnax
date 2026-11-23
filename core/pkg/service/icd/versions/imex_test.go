// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/icd/versions"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("DecodeImExEnvelope", func() {
	It("Should decode an envelope exported at v0", func(ctx SpecContext) {
		ic := MustSucceed(versions.DecodeImExEnvelope(
			ctx,
			LoadEnvelope("testdata/import_v0.json"),
		))
		Expect(ic.Name).To(Equal("Engine"))
		Expect(ic.Messages).To(HaveExactElements(SatisfyAll(
			HaveField("Key", uuid.MustParse("00000000-0000-4000-8000-000000000001")),
			HaveField("Name", "Status"),
			HaveField("CycleTime", BeNil()),
		)))
	})
})
