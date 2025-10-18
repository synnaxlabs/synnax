// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/symbol"
)

var _ = Describe("Channels", func() {
	Describe("NewChannels", func() {
		It("Should create empty Channels with initialized maps", func() {
			ch := symbol.NewChannels()
			Expect(ch.Read).ToNot(BeNil())
			Expect(ch.Write).ToNot(BeNil())
			Expect(ch.Read).To(HaveLen(0))
			Expect(ch.Write).To(HaveLen(0))
		})
	})
})
