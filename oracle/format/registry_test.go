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
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/format"
	"github.com/synnaxlabs/x/errors"
)

// fakeFormatter wraps content in a known prefix so order of application
// is observable from the output.
type fakeFormatter struct {
	tag string
}

func (f *fakeFormatter) Format(
	_ context.Context,
	content []byte,
	_ string,
) ([]byte, error) {
	return append([]byte(f.tag+":"), content...), nil
}

// failingFormatter fails every file whose content is "bad" and counts its calls.
type failingFormatter struct{ calls atomic.Int32 }

func (f *failingFormatter) Format(
	_ context.Context,
	content []byte,
	_ string,
) ([]byte, error) {
	f.calls.Add(1)
	if string(content) == "bad" {
		return nil, errors.New("synthetic failure")
	}
	return content, nil
}

var _ = Describe("Registry", func() {
	format1 := func(ctx context.Context, r *format.Registry, path, body string) string {
		GinkgoHelper()
		out := r.FormatBatch(ctx, []format.File{{Path: path, Content: []byte(body)}}, 1)
		Expect(out).To(HaveLen(1))
		Expect(out[0].Err).ToNot(HaveOccurred())
		return string(out[0].Content)
	}

	It(
		"Should pass content through unchanged when no formatter is registered",
		func(ctx SpecContext) {
			Expect(format1(ctx, format.NewRegistry(), "/x/foo.unknown", "hello")).
				To(Equal("hello"))
		},
	)

	It(
		"Should route by extension and chain in registration order",
		func(ctx SpecContext) {
			r := format.NewRegistry()
			r.Register(".go", &fakeFormatter{tag: "first"})
			r.Register(".go", &fakeFormatter{tag: "second"})
			Expect(format1(ctx, r, "/x/main.go", "body")).To(Equal("second:first:body"))
		},
	)

	It("Should ignore other extensions", func(ctx SpecContext) {
		r := format.NewRegistry()
		r.Register(".go", &fakeFormatter{tag: "go"})
		Expect(format1(ctx, r, "/x/main.ts", "ts source")).To(Equal("ts source"))
	})

	Describe("FormatBatch", func() {
		It("Should preserve order of input files", func(ctx SpecContext) {
			r := format.NewRegistry()
			r.Register(".go", &fakeFormatter{tag: "f"})
			r.Register(".ts", &fakeFormatter{tag: "t"})
			files := []format.File{
				{Path: "/x/a.go", Content: []byte("A")},
				{Path: "/x/b.ts", Content: []byte("B")},
				{Path: "/x/c.unknown", Content: []byte("C")},
				{Path: "/x/d.go", Content: []byte("D")},
			}
			out := r.FormatBatch(ctx, files, 4)
			Expect(out).To(Equal([]format.Result{
				{Content: []byte("f:A")},
				{Content: []byte("t:B")},
				{Content: []byte("C")},
				{Content: []byte("f:D")},
			}))
		})

		It(
			"Should fail only the file whose chain fails and stop its chain",
			func(ctx SpecContext) {
				r := format.NewRegistry()
				first := &failingFormatter{}
				second := &failingFormatter{}
				r.Register(".go", first)
				r.Register(".go", second)
				out := r.FormatBatch(ctx, []format.File{
					{Path: "/x/a.go", Content: []byte("ok")},
					{Path: "/x/b.go", Content: []byte("bad")},
				}, 2)
				Expect(out[0]).To(Equal(format.Result{Content: []byte("ok")}))
				Expect(out[1].Err).To(MatchError(And(
					ContainSubstring("format /x/b.go"),
					ContainSubstring("synthetic failure"),
				)))
				Expect(first.calls.Load()).To(BeEquivalentTo(2))
				Expect(second.calls.Load()).To(BeEquivalentTo(1))
			},
		)

		It("Should fail every file on a canceled context", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			r := format.NewRegistry()
			r.Register(".go", &fakeFormatter{tag: "f"})
			out := r.FormatBatch(ctx, []format.File{
				{Path: "/x/a.go", Content: []byte("A")},
				{Path: "/x/b.go", Content: []byte("B")},
			}, 1)
			Expect(out[0].Err).To(MatchError(context.Canceled))
			Expect(out[1].Err).To(MatchError(context.Canceled))
		})
	})
})
