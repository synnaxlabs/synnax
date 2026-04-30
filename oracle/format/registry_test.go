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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/format"
	. "github.com/synnaxlabs/x/testutil"
)

// fakeFormatter wraps content in a known prefix so order of application
// is observable from the output.
type fakeFormatter struct {
	tag string
}

func (f *fakeFormatter) Format(_ context.Context, content []byte, _ string) ([]byte, error) {
	return append([]byte(f.tag+":"), content...), nil
}

var _ = Describe("Registry", func() {
	It("Should pass content through unchanged when no formatter is registered", func(ctx SpecContext) {
		r := format.NewRegistry()
		out := MustSucceed(r.Format(ctx, []byte("hello"), "/x/foo.unknown"))
		Expect(string(out)).To(Equal("hello"))
	})

	It("Should route by extension and chain in registration order", func(ctx SpecContext) {
		r := format.NewRegistry()
		r.Register(".go", &fakeFormatter{tag: "first"})
		r.Register(".go", &fakeFormatter{tag: "second"})
		out := MustSucceed(r.Format(ctx, []byte("body"), "/x/main.go"))
		Expect(string(out)).To(Equal("second:first:body"))
	})

	It("Should ignore other extensions", func(ctx SpecContext) {
		r := format.NewRegistry()
		r.Register(".go", &fakeFormatter{tag: "go"})
		out := MustSucceed(r.Format(ctx, []byte("ts source"), "/x/main.ts"))
		Expect(string(out)).To(Equal("ts source"))
	})

	It("Should report whether an extension is registered", func() {
		r := format.NewRegistry()
		Expect(r.Has(".go")).To(BeFalse())
		r.Register(".go", &fakeFormatter{tag: "x"})
		Expect(r.Has(".go")).To(BeTrue())
	})

	Describe("FormatBatch", func() {
		It("Should preserve order of input files", func(ctx SpecContext) {
			r := format.NewRegistry()
			r.Register(".go", &fakeFormatter{tag: "f"})
			files := []format.File{
				{Path: "/x/a.go", Content: []byte("A")},
				{Path: "/x/b.go", Content: []byte("B")},
				{Path: "/x/c.go", Content: []byte("C")},
			}
			out := MustSucceed(r.FormatBatch(ctx, files, 4))
			Expect(out).To(HaveLen(3))
			Expect(out[0].Path).To(Equal("/x/a.go"))
			Expect(string(out[0].Content)).To(Equal("f:A"))
			Expect(out[1].Path).To(Equal("/x/b.go"))
			Expect(string(out[1].Content)).To(Equal("f:B"))
			Expect(out[2].Path).To(Equal("/x/c.go"))
			Expect(string(out[2].Content)).To(Equal("f:C"))
		})

		It("Should pass through files with unknown extensions", func(ctx SpecContext) {
			r := format.NewRegistry()
			files := []format.File{{Path: "/x/a.unknown", Content: []byte("body")}}
			out := MustSucceed(r.FormatBatch(ctx, files, 1))
			Expect(string(out[0].Content)).To(Equal("body"))
		})
	})
})
