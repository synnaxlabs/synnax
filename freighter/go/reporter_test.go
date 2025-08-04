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

var _ = DescribeTable("Reporter", func(protocol string, encodings []string) {
	r := freighter.Reporter{Protocol: protocol, Encodings: encodings}
	report := r.Report()
	Expect(report["protocol"]).To(Equal(protocol))
	Expect(report["encodings"]).To(Equal(encodings))
},
	Entry("should correctly report the protocol and encodings",
		"test",
		[]string{"test"},
	),
	Entry("should allow for protocol to be empty", "", []string{"test"}),
	Entry("should allow for encodings to be nil", "test", nil),
	Entry("should allow for encodings to be empty", "test", []string{}),
)
