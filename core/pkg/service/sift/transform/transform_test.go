// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package transform_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	typev1 "github.com/sift-stack/sift/go/gen/sift/common/type/v1"
	"github.com/synnaxlabs/synnax/pkg/service/sift/transform"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Transform", func() {
	DescribeTable("DataType",
		func(input telem.DataType, expected typev1.ChannelDataType) {
			Expect(transform.DataType(input)).To(Equal(expected))
		},
		Entry("Should map Float64 correctly", telem.Float64T, typev1.ChannelDataType_CHANNEL_DATA_TYPE_DOUBLE),
		Entry("Should map Float32 correctly", telem.Float32T, typev1.ChannelDataType_CHANNEL_DATA_TYPE_FLOAT),
		Entry("Should map Int64 correctly", telem.Int64T, typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_64),
		Entry("Should map Int32 correctly", telem.Int32T, typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_32),
		Entry("Should map Uint64 correctly", telem.Uint64T, typev1.ChannelDataType_CHANNEL_DATA_TYPE_UINT_64),
		Entry("Should map Uint32 correctly", telem.Uint32T, typev1.ChannelDataType_CHANNEL_DATA_TYPE_UINT_32),
		Entry("Should map String correctly", telem.StringT, typev1.ChannelDataType_CHANNEL_DATA_TYPE_STRING),
		Entry("Should map TimeStamp correctly", telem.TimeStampT, typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_64),
		Entry("Should map Int8 to Int32", telem.Int8T, typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_32),
		Entry("Should map Int16 to Int32", telem.Int16T, typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_32),
		Entry("Should map Uint8 to Uint32", telem.Uint8T, typev1.ChannelDataType_CHANNEL_DATA_TYPE_UINT_32),
		Entry("Should map Uint16 to Uint32", telem.Uint16T, typev1.ChannelDataType_CHANNEL_DATA_TYPE_UINT_32),
	)

	It("Should return error for unsupported type", func() {
		Expect(transform.DataType(telem.UUIDT)).Error().
			To(MatchError(ContainSubstring("unsupported data type for Sift: uuid")))
	})

	Describe("SeriesToProtoValues", func() {
		It("Should convert Float64 series", func() {
			series := telem.NewSeriesV(1.5, 2.5, 3.5)
			values := MustSucceed(transform.SeriesToProtoValues(series))
			Expect(values).To(HaveLen(3))
			Expect(values[0].GetDouble()).To(Equal(1.5))
			Expect(values[2].GetDouble()).To(Equal(3.5))
		})

		It("Should convert Float32 series", func() {
			series := telem.NewSeriesV[float32](1.5, 2.5, 3.5)
			values := MustSucceed(transform.SeriesToProtoValues(series))
			Expect(values).To(HaveLen(3))
			Expect(values[0].GetFloat()).To(Equal(float32(1.5)))
		})

		It("Should convert Int64 series", func() {
			series := telem.NewSeriesV[int64](10, 20, 30)
			values := MustSucceed(transform.SeriesToProtoValues(series))
			Expect(values).To(HaveLen(3))
			Expect(values[0].GetInt64()).To(Equal(int64(10)))
		})

		It("Should convert Int32 series", func() {
			series := telem.NewSeriesV[int32](10, 20, 30)
			values := MustSucceed(transform.SeriesToProtoValues(series))
			Expect(values[0].GetInt32()).To(Equal(int32(10)))
		})

		It("Should convert Int16 series to Int32", func() {
			series := telem.NewSeriesV[int16](10, 20, 30)
			values := MustSucceed(transform.SeriesToProtoValues(series))
			Expect(values[0].GetInt32()).To(Equal(int32(10)))
		})

		It("Should convert Int8 series to Int32", func() {
			series := telem.NewSeriesV[int8](10, 20, 30)
			values := MustSucceed(transform.SeriesToProtoValues(series))
			Expect(values[0].GetInt32()).To(Equal(int32(10)))
		})

		It("Should convert Uint64 series", func() {
			series := telem.NewSeriesV[uint64](10, 20, 30)
			values := MustSucceed(transform.SeriesToProtoValues(series))
			Expect(values[0].GetUint64()).To(Equal(uint64(10)))
		})

		It("Should convert Uint32 series", func() {
			series := telem.NewSeriesV[uint32](10, 20, 30)
			values := MustSucceed(transform.SeriesToProtoValues(series))
			Expect(values[0].GetUint32()).To(Equal(uint32(10)))
		})

		It("Should convert Uint16 series to Uint32", func() {
			series := telem.NewSeriesV[uint16](10, 20, 30)
			values := MustSucceed(transform.SeriesToProtoValues(series))
			Expect(values[0].GetUint32()).To(Equal(uint32(10)))
		})

		It("Should convert Uint8 series to Uint32", func() {
			series := telem.NewSeriesV[uint8](10, 20, 30)
			values := MustSucceed(transform.SeriesToProtoValues(series))
			Expect(values[0].GetUint32()).To(Equal(uint32(10)))
		})

		It("Should convert String series", func() {
			series := telem.NewSeriesVariableV("hello", "world")
			values := MustSucceed(transform.SeriesToProtoValues(series))
			Expect(values).To(HaveLen(2))
			Expect(values[0].GetString_()).To(Equal("hello"))
			Expect(values[1].GetString_()).To(Equal("world"))
		})

		It("Should convert Bytes series", func() {
			series := telem.NewSeriesVariableV([]byte("hello"), []byte("world"))
			values := MustSucceed(transform.SeriesToProtoValues(series))
			Expect(values).To(HaveLen(2))
			Expect(values[0].GetBytes()).To(Equal([]byte("hello")))
			Expect(values[1].GetBytes()).To(Equal([]byte("world")))
		})

		It("Should convert TimeStamp series to Int64", func() {
			series := telem.NewSeriesV(telem.SecondTS, telem.SecondTS*2)
			values := MustSucceed(transform.SeriesToProtoValues(series))
			Expect(values).To(HaveLen(2))
			Expect(values[0].GetInt64()).To(Equal(int64(telem.SecondTS)))
		})

		It("Should return error for unsupported type", func() {
			series := telem.Series{DataType: telem.UUIDT}
			Expect(transform.SeriesToProtoValues(series)).Error().
				To(MatchError(ContainSubstring("unsupported data type for Sift: uuid")))
		})
	})
})
