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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Rename across directories", func() {
	It("Should keep a file renamed into a new directory after a SyncDir", func() {
		for _, synced := range []string{"n", "m"} {
			fs := MustSucceed(fault.New(fault.Config{}))
			MustSucceed(fs.Sub("n"))
			f := create(fs, "n/x")
			Expect(f.Write([]byte("tacocat"))).To(Equal(7))
			Expect(f.Sync()).To(Succeed())
			Expect(f.Close()).To(Succeed())
			Expect(fs.SyncDir("")).To(Succeed())
			Expect(fs.SyncDir("n")).To(Succeed())
			MustSucceed(fs.Sub("m"))
			Expect(fs.Rename("n/x", "m/x")).To(Succeed())
			Expect(fs.SyncDir(synced)).To(Succeed())
			image := strict(fs)
			inN := MustSucceed(image.Exists("n/x"))
			inM := MustSucceed(image.Exists("m/x"))
			Expect(
				inN || inM,
			).To(BeTrue(), "the file is lost after SyncDir(%q)", synced)
		}
	})
})
