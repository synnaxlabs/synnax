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
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Transient", func() {
	It("Should mark members reachable only through omitted fields", func(
		ctx SpecContext,
	) {
		r := resolverFor(map[string]string{
			"schemas/synnax/versions/task/v0.oracle": `
Task struct {
	key uuid @key
	size Size
	status Details {
		@go marshal omit
	}

	@go marshal
}

Size uint32

Details struct {
	state State
	payload Payload
	span Span
	ref Ref
}

State enum {
	ok = "ok"
}

Payload union on kind {
	a A
	b B
}

A struct {
	x int64
}

B struct {
	y string
}

Span int64

Ref = A
`,
		})
		f := MustSucceed(r.File(ctx, "schemas/synnax/task", 0))
		transient := f.Transient()
		Expect(transient.Slice()).To(ConsistOf(
			"Details", "State", "Payload", "A", "B", "Span", "Ref",
		))
	})

	It("Should report nothing for a file without marshal declarations", func(
		ctx SpecContext,
	) {
		r := resolverFor(map[string]string{
			"schemas/synnax/versions/task/v0.oracle": `
Task struct {
	key uuid @key
	status Details {
		@go marshal omit
	}
}

Details struct {
	running bool
}
`,
		})
		f := MustSucceed(r.File(ctx, "schemas/synnax/task", 0))
		Expect(f.Transient()).To(BeEmpty())
	})

	It("Should keep members reachable through a persisted path stored", func(
		ctx SpecContext,
	) {
		r := resolverFor(map[string]string{
			"schemas/synnax/versions/task/v0.oracle": `
Task struct {
	key uuid @key
	stored Details
	cached Details {
		@go marshal omit
	}

	@go marshal
}

Details struct {
	running bool
}
`,
		})
		f := MustSucceed(r.File(ctx, "schemas/synnax/task", 0))
		Expect(f.Transient()).To(BeEmpty())
	})
})
