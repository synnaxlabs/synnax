// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build !windows

package fs_test

import (
	"os"
	"path/filepath"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xfs "github.com/synnaxlabs/x/io/fs"
)

var _ = Describe("SyncDir on the OS", func() {
	var dir string
	BeforeEach(func() { dir = GinkgoT().TempDir() })

	It("Should return ENOTDIR for a file that it cannot read", func() {
		name := filepath.Join(dir, "f.txt")
		Expect(os.WriteFile(name, []byte("tacocat"), 0o200)).To(Succeed())
		Expect(xfs.Default.SyncDir(name)).To(MatchError(syscall.ENOTDIR))
	})

	It("Should return ENOTDIR for a FIFO and not wait for a writer", func(SpecContext) {
		name := filepath.Join(dir, "fifo")
		Expect(syscall.Mkfifo(name, 0o600)).To(Succeed())
		Expect(xfs.Default.SyncDir(name)).To(MatchError(syscall.ENOTDIR))
	}, SpecTimeout(5*time.Second))
})
