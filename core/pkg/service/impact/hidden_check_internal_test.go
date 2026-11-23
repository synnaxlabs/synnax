// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package impact

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Hidden count internal checks", func() {
	ginkgo.It("Should panic on a hidden entry whose effect it does not count", func() {
		var h Hidden
		gomega.Expect(func() { h.count("keep") }).
			To(gomega.PanicWith(`[impact] - no hidden count for effect "keep"`))
	})
})
