// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v4 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
)

var _ = Describe("Record", func() {
	It("Should store a write config under the v4 table prefix", func() {
		Expect(v5.WriteConfig{}.CustomTypeName()).
			To(Equal(v4.WriteConfig{}.CustomTypeName()))
	})

	It("Should store a read config under the v4 table prefix", func() {
		Expect(v5.ReadConfig{}.CustomTypeName()).
			To(Equal(v4.ReadConfig{}.CustomTypeName()))
	})
})
