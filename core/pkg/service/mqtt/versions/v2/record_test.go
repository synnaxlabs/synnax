// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v2"
)

var _ = Describe("CustomTypeName", func() {
	It("Should keep the v1 gorp type name of each stored config", func() {
		Expect(v2.WriteConfig{}.CustomTypeName()).
			To(Equal(v1.WriteConfig{}.CustomTypeName()))
		Expect(v2.ReadConfig{}.CustomTypeName()).
			To(Equal(v1.ReadConfig{}.CustomTypeName()))
		Expect(v2.ScanConfig{}.CustomTypeName()).
			To(Equal(v1.ScanConfig{}.CustomTypeName()))
		Expect(v2.EdgeConfig{}.CustomTypeName()).
			To(Equal(v1.EdgeConfig{}.CustomTypeName()))
	})
})
