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
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("SyncDir on the OS", func() {
	var dir string
	BeforeEach(func() { dir = GinkgoT().TempDir() })

	It("Should return ENOTDIR for a file without read permission", func() {
		name := filepath.Join(dir, "f.txt")
		Expect(os.WriteFile(name, []byte("x"), 0o200)).To(Succeed())
		Expect(xfs.Default.SyncDir(name)).To(MatchError(syscall.ENOTDIR))
	})

	It("Should return ENOTDIR for a FIFO without blocking", func() {
		name := filepath.Join(dir, "fifo")
		Expect(syscall.Mkfifo(name, 0o600)).To(Succeed())
		done := make(chan error, 1)
		go func() { done <- xfs.Default.SyncDir(name) }()
		var err error
		select {
		case err = <-done:
		case <-time.After(time.Second):
			w := MustSucceed(os.OpenFile(name, os.O_WRONLY, 0))
			Expect(w.Close()).To(Succeed())
			<-done
			Fail("SyncDir blocked for 1s on a FIFO")
		}
		Expect(err).To(MatchError(syscall.ENOTDIR))
	})
})
