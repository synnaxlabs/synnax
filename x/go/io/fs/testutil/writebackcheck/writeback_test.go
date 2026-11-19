// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writebackcheck_test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

func TestWritebackCheck(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "IO FS Writeback Check Suite")
}

var _ = Describe("Writeback", func() {
	It("Should leave out of the next Sync the bytes that a writeback made durable",
		func() {
			disk := MustSucceed(fault.New(fault.Config{}))
			f := MustSucceed(disk.Open("f", os.O_CREATE|os.O_RDWR))
			Expect(f.WriteAt(make([]byte, 100), 0)).To(Equal(100))
			disk.Writeback()
			b := make([]byte, 100)
			for i := range b {
				b[i] = 1
			}
			Expect(f.WriteAt(b, 100)).To(Equal(100))
			Expect(f.Sync()).To(Succeed())
			stat := MustSucceed(disk.Stat("f")).Sys().(fault.Stat)
			Expect(f.Close()).To(Succeed())
			Expect(stat.Synced).To(Equal([]fault.Span{{From: 100, To: 200}}))
		})
})
