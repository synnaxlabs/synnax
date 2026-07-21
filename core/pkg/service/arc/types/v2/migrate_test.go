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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	textv0 "github.com/synnaxlabs/arc/text/types/v0"
	"github.com/synnaxlabs/synnax/pkg/service/arc/types/v0"
)

var _ = Describe("v1 -> current Arc migration", func() {
	It("Should seed the document from the previously persisted raw text", func(ctx SpecContext) {
		got := migrateFromV0(ctx, v0.Arc{
			Key:  uuid.New(),
			Name: "legacy",
			Mode: v0.ModeText,
			Text: textv0.Text{Raw: "x := 1"},
		})
		Expect(got.Text.Materialize().Raw).To(Equal("x := 1"))
	})

	It("Should produce an empty document when there is no prior text", func(ctx SpecContext) {
		got := migrateFromV0(ctx, v0.Arc{
			Key:  uuid.New(),
			Name: "empty",
			Mode: v0.ModeGraph,
		})
		Expect(got.Text.Materialize().Raw).To(Equal(""))
	})
})
