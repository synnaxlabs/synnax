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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Mem", func() {
	It("Should advance the modification time on every write", func() {
		fs := fs.NewMem()
		var prev time.Time
		for range 100 {
			f := MustSucceed(fs.Open("f.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC))
			MustSucceed(f.Write([]byte("tacocat")))
			Expect(f.Close()).To(Succeed())
			mod := MustSucceed(fs.Stat("f.txt")).ModTime()
			Expect(mod).To(BeTemporally(">", prev))
			prev = mod
		}
	})
})
