// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	graph "github.com/synnaxlabs/arc/graph/types/v0"
	ir "github.com/synnaxlabs/arc/ir/types/v0"
	text "github.com/synnaxlabs/arc/text/types/v0"
	v0 "github.com/synnaxlabs/synnax/pkg/service/arc/types/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/arc/types/v1"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateArc", func() {
	It("Should carry the arc's key, name, mode, graph, and text", func(ctx SpecContext) {
		key := uuid.New()
		migrated := MustSucceed(v1.MigrateArc(ctx, v0.Arc{
			Key:  key,
			Name: "my-arc",
			Mode: v0.ModeText,
			Text: text.Text{Raw: "x := 1"},
			Graph: graph.Graph{
				Functions: ir.Functions{{Key: "scale", Body: ir.Body{Raw: "x * 2"}}},
			},
		}))
		Expect(migrated.Key).To(Equal(key))
		Expect(migrated.Name).To(Equal("my-arc"))
		Expect(migrated.Mode).To(Equal(v1.ModeText))
		Expect(migrated.Text.Raw).To(Equal("x := 1"))
		Expect(migrated.Graph.Functions).To(HaveLen(1))
		Expect(migrated.Graph.Functions[0].Key).To(Equal("scale"))
	})

	It("Should drop the persisted program and status", func(ctx SpecContext) {
		migrated := MustSucceed(v1.MigrateArc(ctx, v0.Arc{
			Key:    uuid.New(),
			Name:   "loaded",
			Status: &v0.Status{Name: "running", Variant: "success"},
		}))
		Expect(migrated.Status).To(BeNil())
		Expect(migrated.Program).To(BeNil())
	})
})
