// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package index_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Errors", func() {
	It(
		"Should name the position that the index cannot resolve, and its length",
		func() {
			Expect(index.NewDiscontinuousOffsetError(5, 3)).To(SatisfyAll(
				MatchError(index.ErrDiscontinuous),
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"failed to resolve position 5 in continuous index of length 3",
				)),
			))
		},
	)

	It("Should name the stamp that the index does not hold", func() {
		Expect(index.NewDiscontinuousStampError(telem.SecondTS)).To(SatisfyAll(
			MatchError(index.ErrDiscontinuous),
			MatchError(ContainSubstring("does not exist in the index")),
		))
	})
})
