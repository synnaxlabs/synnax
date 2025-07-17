// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package freighter_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/freighter"
)

var _ = DescribeTable("Reporter",
	func(
		protocol string,
		encodings []string,
		expectedProtocol string,
		expectedEncodings []string,
	) {
		r := freighter.Reporter{Protocol: protocol, Encodings: encodings}
		Expect(r.Report()["protocol"]).To(Equal(expectedProtocol))
		Expect(r.Report()["encodings"]).To(Equal(expectedEncodings))
	},
	Entry("Should correctly report the protocol and encodings", "test", []string{"test"}, "test", []string{"test"}),
	Entry("should allow for protocol to be empty", "", []string{"test"}, "", []string{"test"}),
	Entry("should allow for encodings to be empty", "test", nil, "test", nil),
)
