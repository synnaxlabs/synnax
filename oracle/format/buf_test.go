// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package format_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/format"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("BufFormat formatter", func() {
	BeforeEach(func() {
		if _, err := exec.LookPath("buf"); err != nil {
			Skip("buf not on PATH")
		}
	})

	It("Should normalise spacing in a .proto file", func(ctx SpecContext) {
		raw := []byte(`syntax = "proto3";
package x;
message  M { int32   a=1;  }
`)
		out := MustSucceed(format.NewBufFormat().Format(ctx, raw, "/abs/x.proto"))
		Expect(string(out)).To(ContainSubstring("message M {"))
		Expect(string(out)).To(ContainSubstring("int32 a = 1;"))
	})

	It("Should be idempotent", func(ctx SpecContext) {
		raw := []byte(`syntax = "proto3";
package x;
message M {
  int32 a = 1;
}
`)
		first := MustSucceed(format.NewBufFormat().Format(ctx, raw, "/abs/x.proto"))
		second := MustSucceed(format.NewBufFormat().Format(ctx, first, "/abs/x.proto"))
		Expect(string(first)).To(Equal(string(second)))
	})

	It("Should surface an error on syntactically invalid input", func(ctx SpecContext) {
		raw := []byte(`syntax = "proto3";
package x;
message {{{
`)
		Expect(format.NewBufFormat().Format(ctx, raw, "/abs/x.proto")).Error().
			To(MatchError(ContainSubstring("buf format")))
	})
})
