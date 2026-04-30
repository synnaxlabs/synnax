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

var _ = Describe("Go formatter", func() {
	BeforeEach(func() {
		if _, err := exec.LookPath("gofmt"); err != nil {
			Skip("gofmt not on PATH")
		}
	})

	It("Should format unformatted Go source", func(ctx SpecContext) {
		raw := []byte("package x\n\nfunc Foo()    int {return  1}\n")
		out := MustSucceed(format.NewGo().Format(ctx, raw, "/x/foo.go"))
		Expect(string(out)).To(ContainSubstring("func Foo() int { return 1 }"))
	})

	It("Should be idempotent", func(ctx SpecContext) {
		raw := []byte("package x\n\nfunc Foo()    int {return  1}\n")
		first := MustSucceed(format.NewGo().Format(ctx, raw, "/x/foo.go"))
		second := MustSucceed(format.NewGo().Format(ctx, first, "/x/foo.go"))
		Expect(string(first)).To(Equal(string(second)))
	})

	It("Should apply -s simplification", func(ctx SpecContext) {
		raw := []byte(`package x

type T struct{ V int }

var Vs = []T{T{V: 1}, T{V: 2}}
`)
		out := MustSucceed(format.NewGo().Format(ctx, raw, "/x/foo.go"))
		Expect(string(out)).To(ContainSubstring("[]T{{V: 1}, {V: 2}}"))
	})

	It("Should surface errors on invalid Go input", func(ctx SpecContext) {
		raw := []byte("package x\n\nfunc Foo( {\n")
		Expect(format.NewGo().Format(ctx, raw, "/x/foo.go")).Error().To(MatchError(ContainSubstring("gofmt")))
	})
})
