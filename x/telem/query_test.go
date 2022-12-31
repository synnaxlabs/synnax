// Copyright 2022 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Query", func() {
	Describe("SetTimeRange", func() {
		It("Should set and get the time range on a query", func() {
			q := query.New()
			tr := telem.TimeRange{Start: telem.Now(), End: telem.Now().Add(telem.Second)}
			telem.SetTimeRange(q, tr)
			got, err := telem.GetTimeRange(q)
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(Equal(tr))
		})
		It("Should return an error if the time rang eis not set", func() {
			q := query.New()
			got, err := telem.GetTimeRange(q)
			Expect(err).To(HaveOccurredAs(telem.InvalidTimeRange))
			Expect(got).To(Equal(telem.TimeRangeZero))
		})
	})
	Describe("SetDensity", func() {
		It("Should set and get the data type on a query", func() {
			q := query.New()
			dt := telem.Density(8)
			telem.SetDensity(q, dt)
			gdt, err := telem.GetDensity(q)
			Expect(err).ToNot(HaveOccurred())
			Expect(gdt).To(Equal(dt))
		})
		It("Should return an error if the data type is not set", func() {
			q := query.New()
			gdt, err := telem.GetDensity(q)
			Expect(err).To(HaveOccurredAs(telem.InvalidDensity))
			Expect(gdt).To(Equal(telem.Density(0)))
		})
	})
	Describe("SetRate", func() {
		It("Should set and get the data rate on a query", func() {
			q := query.New()
			dr := telem.Rate(8)
			telem.SetRate(q, dr)
			got, err := telem.GetRate(q)
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(Equal(dr))
		})
		It("Should return an error if the data rate is not set", func() {
			q := query.New()
			got, err := telem.GetRate(q)
			Expect(err).To(HaveOccurredAs(telem.InvalidRate))
			Expect(got).To(Equal(telem.Rate(0)))
		})
	})
})
