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

var _ = Describe("MonoClock", func() {
	It("Should return increasing timestamps from a real clock", func() {
		var c telem.MonoClock
		prev := c.Now()
		for range 100 {
			next := c.Now()
			Expect(next).To(BeNumerically(">", prev))
			prev = next
		}
	})

	It("Should bump by 1ns when the source returns the same value", func() {
		fixed := telem.TimeStamp(1000)
		c := telem.MonoClock{Source: func() telem.TimeStamp { return fixed }}
		first := c.Now()
		Expect(first).To(Equal(fixed))
		Expect(c.Now()).To(Equal(fixed + 1))
		Expect(c.Now()).To(Equal(fixed + 2))
		Expect(c.Now()).To(Equal(fixed + 3))
	})

	It("Should use the source value when it advances past the last", func() {
		cursor := telem.TimeStamp(100)
		c := telem.MonoClock{Source: func() telem.TimeStamp { return cursor }}
		Expect(c.Now()).To(Equal(telem.TimeStamp(100)))
		Expect(c.Now()).To(Equal(telem.TimeStamp(101)))
		cursor = 500
		Expect(c.Now()).To(Equal(telem.TimeStamp(500)))
		Expect(c.Now()).To(Equal(telem.TimeStamp(501)))
	})

	It("Should handle the source going backwards", func() {
		cursor := telem.TimeStamp(1000)
		c := telem.MonoClock{Source: func() telem.TimeStamp { return cursor }}
		Expect(c.Now()).To(Equal(telem.TimeStamp(1000)))
		cursor = 500
		Expect(c.Now()).To(Equal(telem.TimeStamp(1001)))
		Expect(c.Now()).To(Equal(telem.TimeStamp(1002)))
		cursor = 2000
		Expect(c.Now()).To(Equal(telem.TimeStamp(2000)))
	})

	It("Should default to the package-level Now when no source is set", func() {
		var c telem.MonoClock
		ts := c.Now()
		Expect(ts).To(BeNumerically(">", 0))
		Expect(ts).To(BeNumerically("~", telem.Now(), telem.TimeStamp(telem.Millisecond)))
	})
})
