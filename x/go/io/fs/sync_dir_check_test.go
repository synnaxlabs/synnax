// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fs_test

import (
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MemFS SyncDir", func() {
	It("Should sync the directory of a root file, as the OS FS does", func() {
		Expect(xfs.Default.SyncDir(path.Dir("a.txt"))).To(Succeed())
		mem := xfs.NewMem()
		Expect(xfs.WriteFile(mem, "a.txt", []byte("x"))).To(Succeed())
		Expect(mem.SyncDir(path.Dir("a.txt"))).To(Succeed())
	})

	It("Should name the full path in its error", func() {
		mem := xfs.NewMem()
		MustSucceed(mem.Sub("a/b"))
		var pe *os.PathError
		Expect(mem.SyncDir("a/b/missing")).To(BeAssignableToTypeOf(pe))
		err := mem.SyncDir("a/b/missing").(*os.PathError)
		Expect(err.Path).To(Equal("a/b/missing"))
	})
})
