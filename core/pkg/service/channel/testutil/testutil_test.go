// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/x/set"
)

var _ = Describe("UniqueChannelName", func() {
	It("Should generate a name with the test channel prefix", func() {
		Expect(UniqueChannelName()).To(HavePrefix("test_ch_"))
	})
	It("Should generate unique channel names", func() {
		count := 100
		existingNames := make(set.Set[string], count)
		for range count {
			nextName := UniqueChannelName()
			Expect(existingNames.Contains(nextName)).To(BeFalse())
			existingNames.Add(nextName)
		}
	})
})
