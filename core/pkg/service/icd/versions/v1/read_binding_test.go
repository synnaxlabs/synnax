// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
	"github.com/synnaxlabs/x/telem"
)

const binding = `{"field":"00000000-0000-0000-0000-000000000001","channel":7`

var _ = Describe("ReadBinding", func() {
	DescribeTable("Should decode the data type of a binding",
		func(data string, expected *telem.DataType) {
			var b v1.ReadBinding
			Expect(json.Unmarshal([]byte(data), &b)).To(Succeed())
			Expect(b.Channel).To(BeEquivalentTo(7))
			Expect(b.DataType).To(Equal(expected))
		},
		Entry("an empty data type as absent",
			binding+`,"data_type":""}`,
			nil,
		),
		Entry("an absent data type as absent",
			binding+`}`,
			nil,
		),
		Entry("a data type as itself",
			binding+`,"data_type":"int32"}`,
			new(telem.Int32T),
		),
	)

	It("Should return the error of a binding that is not valid", func() {
		var b v1.ReadBinding
		Expect(json.Unmarshal([]byte(binding+`,"data_type":7}`), &b)).
			To(MatchError(ContainSubstring("data_type")))
	})
})
