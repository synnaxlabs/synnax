// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sift_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	typev1 "github.com/sift-stack/sift/go/gen/sift/common/type/v1"
	"github.com/synnaxlabs/synnax/pkg/service/sift"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Sift", func() {
	Describe("MapDataType", func() {
		It("Should map Float64 correctly", func() {
			Expect(sift.MapDataType(telem.Float64T)).
				To(Equal(typev1.ChannelDataType_CHANNEL_DATA_TYPE_DOUBLE))
		})

		It("Should map Float32 correctly", func() {
			Expect(sift.MapDataType(telem.Float32T)).
				To(Equal(typev1.ChannelDataType_CHANNEL_DATA_TYPE_FLOAT))
		})

		It("Should map Int64 correctly", func() {
			Expect(sift.MapDataType(telem.Int64T)).
				To(Equal(typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_64))
		})

		It("Should map Int32 correctly", func() {
			Expect(sift.MapDataType(telem.Int32T)).
				To(Equal(typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_32))
		})

		It("Should map Uint64 correctly", func() {
			Expect(sift.MapDataType(telem.Uint64T)).
				To(Equal(typev1.ChannelDataType_CHANNEL_DATA_TYPE_UINT_64))
		})

		It("Should map Uint32 correctly", func() {
			Expect(sift.MapDataType(telem.Uint32T)).
				To(Equal(typev1.ChannelDataType_CHANNEL_DATA_TYPE_UINT_32))
		})

		It("Should map String correctly", func() {
			Expect(sift.MapDataType(telem.StringT)).
				To(Equal(typev1.ChannelDataType_CHANNEL_DATA_TYPE_STRING))
		})

		It("Should map TimeStamp correctly", func() {
			Expect(sift.MapDataType(telem.TimeStampT)).
				To(Equal(typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_64))
		})

		It("Should map Int8 to Int32", func() {
			Expect(sift.MapDataType(telem.Int8T)).
				To(Equal(typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_32))
		})

		It("Should map Int16 to Int32", func() {
			Expect(sift.MapDataType(telem.Int16T)).
				To(Equal(typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_32))
		})

		It("Should map Uint8 to Uint32", func() {
			Expect(sift.MapDataType(telem.Uint8T)).
				To(Equal(typev1.ChannelDataType_CHANNEL_DATA_TYPE_UINT_32))
		})

		It("Should map Uint16 to Uint32", func() {
			Expect(sift.MapDataType(telem.Uint16T)).
				To(Equal(typev1.ChannelDataType_CHANNEL_DATA_TYPE_UINT_32))
		})
	})

	Describe("ConvertSeriesToValues", func() {
		It("Should convert Float64 series correctly", func() {
			series := telem.NewSeriesV(1.5, 2.5, 3.5)
			Expect(sift.ConvertSeriesToValues(series)).
				To(Equal([]any{1.5, 2.5, 3.5}))
		})

		It("Should convert Int64 series correctly", func() {
			series := telem.NewSeriesV[int64](1, 2, 3)
			Expect(sift.ConvertSeriesToValues(series)).
				To(Equal([]any{int64(1), int64(2), int64(3)}))
		})

		It("Should convert Int8 to Int32", func() {
			series := telem.NewSeriesV[int8](1, 2, 3)
			Expect(sift.ConvertSeriesToValues(series)).
				To(Equal([]any{int32(1), int32(2), int32(3)}))
		})
	})

	Describe("ParseDeviceProperties", func() {
		It("Should parse valid JSON properties", func() {
			jsonStr := `{
				"uri": "api.siftstack.com:443",
				"api_key": "sk-test-key",
				"asset_name": "test-asset"
			}`
			Expect(sift.ParseDeviceProperties(jsonStr)).To(Equal(sift.DeviceProperties{
				URI:       "api.siftstack.com:443",
				APIKey:    "sk-test-key",
				AssetName: "test-asset",
			}))
		})

		It("Should return error for invalid JSON", func() {
			Expect(sift.ParseDeviceProperties("invalid json")).Error().
				To(HaveOccurred())
		})
	})
})
