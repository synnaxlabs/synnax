// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Convert break", func() {
	It("Should fail a series of a data type it does not know", func() {
		s := telem.Series{DataType: telem.DataType("float16"), Data: []byte{1, 2, 3, 4}}
		converted, err := telem.Convert(s, telem.Float64T)
		Expect(err).To(
			MatchError(ContainSubstring("float16")),
			"Convert returned %d samples of %s",
			converted.Len(),
			converted.DataType,
		)
	})
})
