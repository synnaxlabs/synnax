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
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/secret/versions"
)

var _ = Describe("NewMigrations", func() {
	It("Should pin the migration chain keys in order", func() {
		Expect(lo.Map(
			versions.NewMigrations(1),
			func(m gorp.Migration, _ int) string { return m.Key() },
		)).To(Equal([]string{"v63_file_rack"}))
	})
})
