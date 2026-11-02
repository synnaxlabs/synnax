// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v3 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v3"
)

var _ = Describe("ScanConfig", func() {
	Describe("CustomTypeName", func() {
		It("Should return the NI scanner config gorp type name", func() {
			Expect(v3.ScanConfig{}.CustomTypeName()).To(Equal("ni_scanner_config"))
		})
	})
})
