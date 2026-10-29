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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/arc/versions"
)

var _ = Describe("NewMigrations", func() {
	It("Should end the chain with the group migration", func() {
		chain := versions.NewMigrations(versions.MigrationsConfig{})
		Expect(chain[0].Key()).To(Equal("normalize_keys"))
		Expect(chain[len(chain)-1].Key()).To(Equal("v61_arcs_group"))
	})
})

var _ = DescribeTable("FileType",
	func(t versions.Type, header string) {
		Expect(versions.FileType(t)).To(Equal(header))
	},
	Entry("automation", versions.TypeAutomation, "arc-automation"),
	Entry("module", versions.TypeModule, "arc-module"),
)
