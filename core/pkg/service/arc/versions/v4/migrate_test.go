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
	"github.com/synnaxlabs/arc/program"
	text "github.com/synnaxlabs/arc/text/versions/v1"
	v3 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v4"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateArc", func() {
	It(
		"Should keep the stored fields and drop the derived ones",
		func(ctx SpecContext) {
			key := uuid.New()
			migrated := MustSucceed(v4.MigrateArc(ctx, v3.Arc{
				Key:     key,
				Name:    "startup",
				Mode:    v3.ModeText,
				Text:    text.Text{Doc: text.Create("x := 1")},
				Program: &program.Program{},
				Status:  &v3.Status{Key: "arc_status"},
			}))
			Expect(migrated.Key).To(Equal(key))
			Expect(migrated.Name).To(Equal("startup"))
			Expect(migrated.Mode).To(Equal(v4.ModeText))
			Expect(migrated.Text.Materialize().Raw).To(Equal("x := 1"))
			Expect(migrated.Program).To(BeNil())
		},
	)
})
