// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apischematic "github.com/synnaxlabs/synnax/pkg/api/schematic"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Retrieve", func() {
	It(
		"Should return no schematics for a request with no filter",
		func(ctx SpecContext) {
			s := schematic.Schematic{Project: &proj.Key, Name: "unfiltered"}
			Expect(schematicSvc.NewWriter(nil).Create(ctx, &s)).To(Succeed())
			res := MustSucceed(apiSvc.Retrieve(
				AuthedCtx(ctx, author),
				apischematic.RetrieveRequest{},
			))
			Expect(res.Schematics).To(BeEmpty())
			Expect(res.Schematics).ToNot(BeNil())
		},
	)
})
