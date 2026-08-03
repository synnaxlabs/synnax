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

var _ = Describe("TimeRange", func() {
	Describe("NewRangeSeconds", func() {
		It("Should instantiate a time range between a particular starting number of seconds and ending number of seconds", func() {
			tr := telem.NewRangeSeconds(1, 5)
			Expect(tr.Start).To(Equal(telem.SecondTS * 1))
			Expect(tr.End).To(Equal(telem.SecondTS * 5))
		})
	})

})
