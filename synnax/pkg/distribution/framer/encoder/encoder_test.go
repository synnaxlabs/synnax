// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package encoder_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gmeasure"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/encoder"
	"github.com/synnaxlabs/x/telem"
	"time"
)

var _ = Describe("Encoder", func() {

	var experiment *gmeasure.Experiment

	typeSlice := []telem.DataType{
		"uint8", "uint16", "uint16",
	}
	keySlice := channel.Keys{
		1141, 2434, 3234,
	}
	var cd encoder.Codec = encoder.New(
		typeSlice,
		keySlice,
	)
	// All the same
	series1 := telem.Series{}
	series2 := telem.Series{}
	series3 := telem.Series{}
	// Different data-length
	series4 := telem.Series{}
	// Different time-series
	series5 := telem.Series{}
	series1.TimeRange.Start = 5
	series1.TimeRange.End = 10
	series2.TimeRange.Start = 5
	series2.TimeRange.End = 10
	series3.TimeRange.Start = 5
	series3.TimeRange.End = 10
	series4.TimeRange.Start = 5
	series4.TimeRange.End = 10
	series5.TimeRange.Start = 5
	series5.TimeRange.End = 12
	series1.DataType = "uint8"
	series2.DataType = "uint16"
	series3.DataType = "uint16"
	series4.DataType = "uint16"
	series5.DataType = "uint16"
	series1.Data = []byte{1, 2, 3}
	series2.Data = []byte{5, 4, 2, 4, 1, 6}
	series3.Data = []byte{1, 4, 6, 2, 5, 4}
	series4.Data = []byte{1, 4, 6, 2, 5, 4, 1, 2}
	series5.Data = []byte{4, 2, 3, 4, 2, 3}

	BeforeEach(func() {
		experiment = gmeasure.NewExperiment(CurrentSpecReport().LeafNodeText)
	})

	BeforeEach(func() {
		experiment.RecordDuration("runtime", time.Second, gmeasure.Annotation("first"), gmeasure.Style("{{red}}"), gmeasure.Precision(time.Millisecond), gmeasure.Units("ignored"))
	})

	Describe("Various tests", func() {
		It("Everything is the same", func() {
			testStruct := framer.Frame{
				Keys:   keySlice,
				Series: []telem.Series{series1, series2, series3},
			}
			experiment.MeasureDuration("runtime", func() {
				byteArray, err := cd.Encode(testStruct)
				returnStruct, err := cd.Decode(byteArray)
				Expect(err).NotTo(HaveOccurred())
				Expect(testStruct).To(Equal(returnStruct))
			}, gmeasure.Annotation("second"))
		})
		It("Not all channels sent", func() {
			testStruct := framer.Frame{
				Keys:   keySlice[0:2],
				Series: []telem.Series{series1, series2},
			}
			byteArray, err := cd.Encode(testStruct)
			returnStruct, err := cd.Decode(byteArray)
			Expect(err).NotTo(HaveOccurred())
			Expect(testStruct).To(Equal(returnStruct))
		})
		It("Not all data lengths the same", func() {
			testStruct := framer.Frame{
				Keys:   keySlice,
				Series: []telem.Series{series1, series2, series4},
			}
			byteArray, err := cd.Encode(testStruct)
			returnStruct, err := cd.Decode(byteArray)
			Expect(err).NotTo(HaveOccurred())
			Expect(testStruct).To(Equal(returnStruct))
		})
		It("Not all time-range the same", func() {
			testStruct := framer.Frame{
				Keys:   keySlice,
				Series: []telem.Series{series1, series2, series5},
			}
			byteArray, err := cd.Encode(testStruct)
			returnStruct, err := cd.Decode(byteArray)
			Expect(err).NotTo(HaveOccurred())
			Expect(testStruct).To(Equal(returnStruct))
		})
		It("Missing channels and data lengths", func() {
			testStruct := framer.Frame{
				Keys:   keySlice[0:2],
				Series: []telem.Series{series1, series4},
			}
			byteArray, err := cd.Encode(testStruct)
			returnStruct, err := cd.Decode(byteArray)
			Expect(err).NotTo(HaveOccurred())
			Expect(testStruct).To(Equal(returnStruct))
		})
		It("Missing time-range and channels", func() {
			testStruct := framer.Frame{
				Keys:   keySlice[0:2],
				Series: []telem.Series{series1, series5},
			}
			byteArray, err := cd.Encode(testStruct)
			returnStruct, err := cd.Decode(byteArray)
			Expect(err).NotTo(HaveOccurred())
			Expect(testStruct).To(Equal(returnStruct))
		})
	})
})
