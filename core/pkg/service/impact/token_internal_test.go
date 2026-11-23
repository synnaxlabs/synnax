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
	"github.com/synnaxlabs/synnax/pkg/service/channel"
)

// The package declares Entry, so the Ginkgo and Gomega names stay qualified.
var _ = ginkgo.Describe("Token", func() {
	ginkgo.It("Should differ for the same hidden total in other counts", func() {
		keys := []channel.Key{1}
		gomega.Expect(channelsSum(keys, Plan{Hidden: Hidden{Restrict: 1}})).
			ToNot(gomega.Equal(channelsSum(keys, Plan{Hidden: Hidden{Detach: 1}})))
		gomega.Expect(channelsSum(keys, Plan{Hidden: Hidden{Cascade: 1}})).
			ToNot(gomega.Equal(channelsSum(keys, Plan{Hidden: Hidden{Detach: 1}})))
	})
})
