// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v4 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v4"
)

var _ = Describe("CustomTypeName", func() {
	DescribeTable("Should return the NI gorp type name of each config",
		func(cfg interface{ CustomTypeName() string }, want string) {
			Expect(cfg.CustomTypeName()).To(Equal(want))
		},
		Entry("analog write", v4.AnalogWriteConfig{}, "ni_analog_write_config"),
	)
})
