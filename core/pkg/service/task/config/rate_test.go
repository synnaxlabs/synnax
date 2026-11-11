// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package config_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("ValidateStreamRate", func() {
	validateStreamRate := func(sampleRate, streamRate telem.Rate) error {
		v := validate.New("read")
		config.ValidateStreamRate(v, sampleRate, streamRate)
		return v.Error()
	}

	DescribeTable("Should accept a positive stream rate up to the sample rate",
		func(streamRate telem.Rate) {
			Expect(validateStreamRate(10, streamRate)).To(Succeed())
		},
		Entry("below the sample rate", telem.Rate(5)),
		Entry("equal to the sample rate", telem.Rate(10)),
	)

	DescribeTable("Should reject an invalid stream rate",
		func(sampleRate, streamRate telem.Rate, msg string) {
			err := validateStreamRate(sampleRate, streamRate)
			Expect(err).To(MatchError(validate.ErrValidation))
			Expect(err).To(MatchError(ContainSubstring(msg)))
		},
		Entry(
			"zero",
			telem.Rate(10),
			telem.Rate(0),
			"stream_rate: must be greater than 0",
		),
		Entry(
			"negative",
			telem.Rate(10),
			telem.Rate(-1),
			"stream_rate: must be greater than 0",
		),
		Entry(
			"above the sample rate",
			telem.Rate(10),
			telem.Rate(20),
			"stream_rate: must be less than or equal to the sample rate",
		),
	)
})
