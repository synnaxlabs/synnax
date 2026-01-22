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
			dt, err := sift.MapDataType(telem.Float64T)
			Expect(err).ToNot(HaveOccurred())
			Expect(dt).To(Equal(typev1.ChannelDataType_CHANNEL_DATA_TYPE_DOUBLE))
		})

		It("Should map Float32 correctly", func() {
			dt, err := sift.MapDataType(telem.Float32T)
			Expect(err).ToNot(HaveOccurred())
			Expect(dt).To(Equal(typev1.ChannelDataType_CHANNEL_DATA_TYPE_FLOAT))
		})

		It("Should map Int64 correctly", func() {
			dt, err := sift.MapDataType(telem.Int64T)
			Expect(err).ToNot(HaveOccurred())
			Expect(dt).To(Equal(typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_64))
		})

		It("Should map Int32 correctly", func() {
			dt, err := sift.MapDataType(telem.Int32T)
			Expect(err).ToNot(HaveOccurred())
			Expect(dt).To(Equal(typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_32))
		})

		It("Should map Uint64 correctly", func() {
			dt, err := sift.MapDataType(telem.Uint64T)
			Expect(err).ToNot(HaveOccurred())
			Expect(dt).To(Equal(typev1.ChannelDataType_CHANNEL_DATA_TYPE_UINT_64))
		})

		It("Should map Uint32 correctly", func() {
			dt, err := sift.MapDataType(telem.Uint32T)
			Expect(err).ToNot(HaveOccurred())
			Expect(dt).To(Equal(typev1.ChannelDataType_CHANNEL_DATA_TYPE_UINT_32))
		})

		It("Should map String correctly", func() {
			dt, err := sift.MapDataType(telem.StringT)
			Expect(err).ToNot(HaveOccurred())
			Expect(dt).To(Equal(typev1.ChannelDataType_CHANNEL_DATA_TYPE_STRING))
		})

		It("Should map TimeStamp correctly", func() {
			dt, err := sift.MapDataType(telem.TimeStampT)
			Expect(err).ToNot(HaveOccurred())
			Expect(dt).To(Equal(typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_64))
		})

		It("Should map Int8 to Int32", func() {
			dt, err := sift.MapDataType(telem.Int8T)
			Expect(err).ToNot(HaveOccurred())
			Expect(dt).To(Equal(typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_32))
		})

		It("Should map Int16 to Int32", func() {
			dt, err := sift.MapDataType(telem.Int16T)
			Expect(err).ToNot(HaveOccurred())
			Expect(dt).To(Equal(typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_32))
		})

		It("Should map Uint8 to Uint32", func() {
			dt, err := sift.MapDataType(telem.Uint8T)
			Expect(err).ToNot(HaveOccurred())
			Expect(dt).To(Equal(typev1.ChannelDataType_CHANNEL_DATA_TYPE_UINT_32))
		})

		It("Should map Uint16 to Uint32", func() {
			dt, err := sift.MapDataType(telem.Uint16T)
			Expect(err).ToNot(HaveOccurred())
			Expect(dt).To(Equal(typev1.ChannelDataType_CHANNEL_DATA_TYPE_UINT_32))
		})
	})

	Describe("ConvertSeriesToValues", func() {
		It("Should convert Float64 series correctly", func() {
			series := telem.NewSeriesV[float64](1.5, 2.5, 3.5)
			values, err := sift.ConvertSeriesToValues(series)
			Expect(err).ToNot(HaveOccurred())
			Expect(values).To(HaveLen(3))
			Expect(values[0]).To(Equal(1.5))
			Expect(values[1]).To(Equal(2.5))
			Expect(values[2]).To(Equal(3.5))
		})

		It("Should convert Int64 series correctly", func() {
			series := telem.NewSeriesV[int64](1, 2, 3)
			values, err := sift.ConvertSeriesToValues(series)
			Expect(err).ToNot(HaveOccurred())
			Expect(values).To(HaveLen(3))
			Expect(values[0]).To(Equal(int64(1)))
			Expect(values[1]).To(Equal(int64(2)))
			Expect(values[2]).To(Equal(int64(3)))
		})

		It("Should convert Int8 to Int32", func() {
			series := telem.NewSeriesV[int8](1, 2, 3)
			values, err := sift.ConvertSeriesToValues(series)
			Expect(err).ToNot(HaveOccurred())
			Expect(values).To(HaveLen(3))
			Expect(values[0]).To(Equal(int32(1)))
			Expect(values[1]).To(Equal(int32(2)))
			Expect(values[2]).To(Equal(int32(3)))
		})
	})

	Describe("ParseDeviceProperties", func() {
		It("Should parse valid JSON properties", func() {
			jsonStr := `{
				"uri": "api.siftstack.com:443",
				"api_key": "sk-test-key",
				"asset_name": "test-asset",
				"client_key": "synnax-test"
			}`
			props, err := sift.ParseDeviceProperties(jsonStr)
			Expect(err).ToNot(HaveOccurred())
			Expect(props.URI).To(Equal("api.siftstack.com:443"))
			Expect(props.APIKey).To(Equal("sk-test-key"))
			Expect(props.AssetName).To(Equal("test-asset"))
			Expect(props.ClientKey).To(Equal("synnax-test"))
		})

		It("Should parse properties with optional organization_id", func() {
			jsonStr := `{
				"uri": "api.siftstack.com:443",
				"api_key": "sk-test-key",
				"asset_name": "test-asset",
				"client_key": "synnax-test",
				"organization_id": "org-123"
			}`
			props, err := sift.ParseDeviceProperties(jsonStr)
			Expect(err).ToNot(HaveOccurred())
			Expect(props.OrganizationID).To(Equal("org-123"))
		})

		It("Should return error for invalid JSON", func() {
			_, err := sift.ParseDeviceProperties("invalid json")
			Expect(err).To(HaveOccurred())
		})
	})
})
