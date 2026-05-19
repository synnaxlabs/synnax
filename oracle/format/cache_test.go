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
		_, ok := c.Lookup("foo")
		Expect(ok).To(BeFalse())
	})

	It("Should round-trip entries through Save/Load", func() {
		c := format.LoadCache(repoRoot)
		c.Put("foo/bar.go", format.Entry{Raw: "deadbeef", Canonical: "cafef00d"})
		Expect(c.Save()).To(Succeed())

		c2 := format.LoadCache(repoRoot)
		entry := MustBeOk(c2.Lookup("foo/bar.go"))
		Expect(entry.Raw).To(Equal("deadbeef"))
		Expect(entry.Canonical).To(Equal("cafef00d"))
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
		c.Put("a", format.Entry{Raw: "1", Canonical: "1c"})
		c.Put("b", format.Entry{Raw: "2", Canonical: "2c"})
		c.Put("c", format.Entry{Raw: "3", Canonical: "3c"})
		c.PruneTo(set.New("a", "c"))

		Expect(MustBeOk(c.Lookup("a")).Raw).To(Equal("1"))
		_, ok := c.Lookup("b")
		Expect(ok).To(BeFalse())
		Expect(MustBeOk(c.Lookup("c")).Raw).To(Equal("3"))
	})

	It("Should produce stable hashes for identical content", func() {
		Expect(format.Hash([]byte("hello"))).To(Equal(format.Hash([]byte("hello"))))
		Expect(format.Hash([]byte("hello"))).ToNot(Equal(format.Hash([]byte("world"))))
	})
})
