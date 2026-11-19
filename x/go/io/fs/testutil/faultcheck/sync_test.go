// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package faultcheck_test

import (
	"strings"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Sync errors", func() {
	It("Should never make the bytes of a failed Sync durable", func() {
		// Find a run key whose first Sync of f.bin fails and whose second succeeds.
		for key := range uint64(256) {
			fs := MustSucceed(fault.New(fault.Config{RunKey: key, SyncErrorRate: 0.5}))
			f := create(fs, "f.bin")
			if fs.SyncDir("") != nil {
				continue
			}
			Expect(f.WriteAt([]byte(strings.Repeat("B", 4096)), 0)).To(Equal(4096))
			if f.Sync() == nil {
				continue
			}
			Expect(f.WriteAt([]byte("C"), 0)).To(Equal(1))
			if f.Sync() != nil {
				continue
			}
			got := read(strict(fs), "f.bin")
			Expect(got).To(HaveLen(4096))
			Expect(got[1:9]).To(Equal("\x00\x00\x00\x00\x00\x00\x00\x00"),
				"run key %d: bytes of the failed Sync reached durable", key)
			return
		}
		Fail("no run key gave a failed Sync then a good one")
	})

	It("Should succeed on the next Sync after a failed one", func() {
		fs := MustSucceed(fault.New(fault.Config{SyncErrorRate: 1}))
		f := create(fs, "f.bin")
		Expect(f.Write([]byte("burrito"))).To(Equal(7))
		Expect(f.Sync()).To(MatchError(syscall.EIO))
		Expect(f.Write([]byte("bowl"))).To(Equal(4))
		Expect(f.Sync()).To(Succeed())
	})
})
