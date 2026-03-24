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
	xfs "github.com/synnaxlabs/x/io/fs"
	xos "github.com/synnaxlabs/x/os"
)

var _ = Describe("FileExists", func() {
	It("Should return true for an existing file", func() {
		tmp := filepath.Join(GinkgoT().TempDir(), "exists.txt")
		Expect(os.WriteFile(tmp, []byte("data"), xfs.UserRW|xfs.GroupR|xfs.OtherR)).To(Succeed())
		Expect(xos.FileExists(tmp)).To(BeTrue())
	})

	It("Should return false for a non-existent file", func() {
		Expect(xos.FileExists(filepath.Join(GinkgoT().TempDir(), "no-such-file"))).To(BeFalse())
	})
})
