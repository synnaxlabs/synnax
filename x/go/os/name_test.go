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
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xos "github.com/synnaxlabs/x/os"
)

var _ = Describe("Name", func() {
	It("Should return the display name of the operating system", func() {
		names := map[string]string{
			"darwin":  "macOS",
			"windows": "Windows",
			"linux":   "Linux",
		}
		Expect(xos.Name()).To(Equal(names[runtime.GOOS]))
	})
})
