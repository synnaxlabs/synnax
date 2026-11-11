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
	v3 "github.com/synnaxlabs/synnax/pkg/service/modbus/versions/v3"
)

var _ = Describe("CustomTypeName", func() {
	It("Should return the Modbus read config gorp type name", func() {
		Expect(v3.ReadConfig{}.CustomTypeName()).To(Equal("modbus_read_config"))
	})

	It("Should return the Modbus write config gorp type name", func() {
		Expect(v3.WriteConfig{}.CustomTypeName()).To(Equal("modbus_write_config"))
	})
})
