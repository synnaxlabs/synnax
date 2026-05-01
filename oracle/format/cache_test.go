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
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/format"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Cache", func() {
	var repoRoot string

	BeforeEach(func() {
		repoRoot = MustSucceed(os.MkdirTemp("", "cache"))
		DeferCleanup(func() {
			Expect(os.RemoveAll(repoRoot)).To(Succeed())
		})
	})

	It("Should return an empty cache when none exists on disk", func() {
		c := format.LoadCache(repoRoot)
		_, ok := c.LookupRaw("foo")
		Expect(ok).To(BeFalse())
	})

	It("Should round-trip raw hashes through Save/Load", func() {
		c := format.LoadCache(repoRoot)
		c.PutRaw("foo/bar.go", "deadbeef")
		Expect(c.Save()).To(Succeed())

		c2 := format.LoadCache(repoRoot)
		Expect(MustBeOk(c2.LookupRaw("foo/bar.go"))).To(Equal("deadbeef"))
	})

	It("Should round-trip stamps through Save/Load", func() {
		c := format.LoadCache(repoRoot)
		c.PutStamp("buf-generate", "abc123")
		Expect(c.Save()).To(Succeed())

		c2 := format.LoadCache(repoRoot)
		Expect(MustBeOk(c2.LookupStamp("buf-generate"))).To(Equal("abc123"))
	})

	It("Should drop entries not in the keep set when pruned", func() {
		c := format.LoadCache(repoRoot)
		c.PutRaw("a", "1")
		c.PutRaw("b", "2")
		c.PutRaw("c", "3")
		c.PruneRawTo(set.New("a", "c"))

		Expect(MustBeOk(c.LookupRaw("a"))).To(Equal("1"))
		_, ok := c.LookupRaw("b")
		Expect(ok).To(BeFalse())
		Expect(MustBeOk(c.LookupRaw("c"))).To(Equal("3"))
	})

	It("Should produce stable hashes for identical content", func() {
		Expect(format.Hash([]byte("hello"))).To(Equal(format.Hash([]byte("hello"))))
		Expect(format.Hash([]byte("hello"))).ToNot(Equal(format.Hash([]byte("world"))))
	})
})
