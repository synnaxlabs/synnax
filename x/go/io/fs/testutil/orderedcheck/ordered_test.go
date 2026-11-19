// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package orderedcheck_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Ordered entries", func() {
	// A metadata journal commits in order, so the SyncDir of q also commits each change
	// made before it, in any directory.
	It("Should keep an earlier change in another directory when a later one is synced",
		func() {
			fs := MustSucceed(fault.New(fault.Config{}))
			MustSucceed(fs.Sub("p"))
			MustSucceed(fs.Sub("q"))
			Expect(fs.SyncDir("")).To(Succeed())
			Expect(xfs.WriteFile(fs, "p/x", []byte("x"))).To(Succeed())
			Expect(xfs.WriteFile(fs, "q/y", []byte("y"))).To(Succeed())
			Expect(fs.SyncDir("q")).To(Succeed())
			for key := range uint64(32) {
				image := MustSucceed(fs.Crash(fault.CrashConfig{
					RunKey:            key,
					EntrySurvivalRate: 0.5,
					EntriesOrdered:    true,
				}))
				Expect(MustSucceed(image.Exists("q/y"))).To(BeTrue())
				Expect(MustSucceed(image.Exists("p/x"))).To(BeTrue(),
					"run key %d: q/y, created after p/x, survived without it", key)
			}
		})
})
