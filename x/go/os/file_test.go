// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package os_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xos "github.com/synnaxlabs/x/os"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("FileExists", func() {
	It("Should return true for an existing file", func() {
		tmp := filepath.Join(GinkgoT().TempDir(), "exists.txt")
		Expect(os.WriteFile(tmp, []byte("data"), 0644)).To(Succeed())
		exists := MustSucceed(xos.FileExists(tmp))
		Expect(exists).To(BeTrue())
	})

	It("Should return false for a non-existent file", func() {
		exists := MustSucceed(xos.FileExists(filepath.Join(GinkgoT().TempDir(), "no-such-file")))
		Expect(exists).To(BeFalse())
	})
})
