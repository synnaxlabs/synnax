// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v3 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v4"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateArc", func() {
	It("Should lift a v3 Arc as an automation", func(ctx SpecContext) {
		key := uuid.New()
		migrated := MustSucceed(v4.MigrateArc(ctx, v3.Arc{
			Key:  key,
			Name: "legacy",
			Mode: v3.ModeText,
		}))
		Expect(migrated.Key).To(Equal(key))
		Expect(migrated.Name).To(Equal("legacy"))
		Expect(migrated.Mode).To(Equal(v4.ModeText))
		Expect(migrated.Type).To(Equal(v4.TypeAutomation))
	})
})
