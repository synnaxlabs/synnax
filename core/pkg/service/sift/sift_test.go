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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	typev1 "github.com/sift-stack/sift/go/gen/sift/common/type/v1"
	ingestv1 "github.com/sift-stack/sift/go/gen/sift/ingest/v1"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/sift"
	"github.com/synnaxlabs/synnax/pkg/service/sift/client"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	. "github.com/synnaxlabs/x/testutil"
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

		It("Should return error for unsupported type", func() {
			_, err := sift.MapDataType(telem.UUIDT)
			Expect(err).To(HaveOccurred())
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

		It("Should convert Float32 series correctly", func() {
			series := telem.NewSeriesV[float32](1.5, 2.5, 3.5)
			Expect(sift.ConvertSeriesToValues(series)).
				To(Equal([]any{float32(1.5), float32(2.5), float32(3.5)}))
		})

		It("Should convert Uint64 series correctly", func() {
			series := telem.NewSeriesV[uint64](1, 2, 3)
			Expect(sift.ConvertSeriesToValues(series)).
				To(Equal([]any{uint64(1), uint64(2), uint64(3)}))
		})

		It("Should return error for unsupported type", func() {
			series := telem.Series{DataType: telem.UUIDT}
			_, err := sift.ConvertSeriesToValues(series)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ParseDeviceProperties", func() {
		It("Should parse valid JSON properties", func() {
			jsonStr := `{
				"uri": "api.siftstack.com:443",
				"api_key": "sk-test-key"
			}`
			Expect(sift.ParseDeviceProperties(jsonStr)).To(Equal(sift.DeviceProperties{
				URI:    "api.siftstack.com:443",
				APIKey: "sk-test-key",
			}))
		})

		It("Should return error for invalid JSON", func() {
			_, err := sift.ParseDeviceProperties("invalid json")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ParseTaskConfig", func() {
		It("Should parse valid JSON config", func() {
			jsonStr := `{
				"device_key": "sift-device-1",
				"asset_name": "test-asset",
				"flow_name": "telemetry",
				"run_name": "Test Run 1",
				"channels": [1, 2, 3],
				"time_range": {"start": 1000000000, "end": 2000000000}
			}`
			cfg, err := sift.ParseTaskConfig(jsonStr)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.DeviceKey).To(Equal("sift-device-1"))
			Expect(cfg.AssetName).To(Equal("test-asset"))
			Expect(cfg.FlowName).To(Equal("telemetry"))
			Expect(cfg.RunName).To(Equal("Test Run 1"))
			Expect(cfg.Channels).To(HaveLen(3))
			Expect(cfg.TimeRange.Start).To(Equal(telem.TimeStamp(1000000000)))
			Expect(cfg.TimeRange.End).To(Equal(telem.TimeStamp(2000000000)))
		})
	})

	Describe("Uploader", func() {
		var ctx context.Context

		BeforeEach(func() {
			ctx = context.Background()
		})

		It("Should upload data from channels to Sift", func() {
			mockClient, err := MustSucceed(client.NewMockFactory())(ctx, "", "")
			Expect(err).ToNot(HaveOccurred())

			// Create index channel first
			indexCh := channel.Channel{
				Name: "upload_test_index_1", IsIndex: true, DataType: telem.TimeStampT,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &indexCh)).To(Succeed())

			// Create data channel linked to index
			dataCh := channel.Channel{
				Name: "upload_test_data_1", DataType: telem.Float64T, LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &dataCh)).To(Succeed())

			channels := []channel.Channel{indexCh, dataCh}

			// Write test data
			timeRange := telem.TimeRange{
				Start: telem.SecondTS * 10,
				End:   telem.SecondTS * 13,
			}
			w := MustSucceed(dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:  channel.KeysFromChannels(channels),
				Start: timeRange.Start,
				Sync:  config.True(),
			}))
			MustSucceed(w.Write(frame.NewMulti(
				channel.KeysFromChannels(channels),
				[]telem.Series{
					telem.NewSeriesSecondsTSV(10, 11, 12),
					telem.NewSeriesV(1.5, 2.5, 3.5),
				},
			)))
			MustSucceed(w.Commit())
			Expect(w.Close()).To(Succeed())

			// Create uploader with mock client
			uploader := &sift.Uploader{
				Client:     mockClient,
				Framer:     framerSvc,
				ChannelSvc: dist.Channel,
			}

			// Run upload
			params := sift.UploadParams{
				ClientKey: "test-client-key",
				AssetName: "test-asset",
				FlowName:  "telemetry",
				RunName:   "test-run",
				Channels:  channel.KeysFromChannels(channels),
				TimeRange: timeRange,
			}
			Expect(uploader.Upload(ctx, params)).To(Succeed())

			_ = indexCh
			_ = dataCh
		})

		It("Should send data values to Sift stream", func() {
			// Create a stream to capture requests
			requests := confluence.NewStream[*ingestv1.IngestWithConfigDataStreamRequest](10)
			mockClient, err := MustSucceed(client.NewMockFactory(client.MockFactoryConfig{
				Requests: requests,
			}))(ctx, "", "")
			Expect(err).ToNot(HaveOccurred())

			// Create index channel first
			indexCh := channel.Channel{
				Name: "upload_stream_index_1", IsIndex: true, DataType: telem.TimeStampT,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &indexCh)).To(Succeed())

			// Create data channel linked to index
			dataCh := channel.Channel{
				Name: "upload_stream_data_1", DataType: telem.Float64T, LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &dataCh)).To(Succeed())

			channels := []channel.Channel{indexCh, dataCh}

			// Write test data
			timeRange := telem.TimeRange{
				Start: telem.SecondTS * 20,
				End:   telem.SecondTS * 23,
			}
			w := MustSucceed(dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:  channel.KeysFromChannels(channels),
				Start: timeRange.Start,
				Sync:  config.True(),
			}))
			MustSucceed(w.Write(frame.NewMulti(
				channel.KeysFromChannels(channels),
				[]telem.Series{
					telem.NewSeriesSecondsTSV(20, 21, 22),
					telem.NewSeriesV(10.0, 20.0, 30.0),
				},
			)))
			MustSucceed(w.Commit())
			Expect(w.Close()).To(Succeed())

			// Create uploader and run
			uploader := &sift.Uploader{
				Client:     mockClient,
				Framer:     framerSvc,
				ChannelSvc: dist.Channel,
			}

			params := sift.UploadParams{
				ClientKey: "test-key",
				AssetName: "asset",
				FlowName:  "flow",
				RunName:   "run",
				Channels:  channel.KeysFromChannels(channels),
				TimeRange: timeRange,
			}
			Expect(uploader.Upload(ctx, params)).To(Succeed())

			// Verify data was sent to the stream
			req1 := <-requests.Outlet()
			Expect(req1.Flow).To(Equal("flow"))
			Expect(req1.ChannelValues[0].GetDouble()).To(Equal(10.0))
			req2 := <-requests.Outlet()
			Expect(req2.ChannelValues[0].GetDouble()).To(Equal(20.0))
			req3 := <-requests.Outlet()
			Expect(req3.ChannelValues[0].GetDouble()).To(Equal(30.0))
		})

		It("Should fail when no valid channels to upload", func() {
			mockClient, err := MustSucceed(client.NewMockFactory())(ctx, "", "")
			Expect(err).ToNot(HaveOccurred())

			// Create only an index channel (no data channels)
			indexCh := channel.Channel{
				Name: "upload_nodata_index_1", IsIndex: true, DataType: telem.TimeStampT,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &indexCh)).To(Succeed())
			channels := []channel.Channel{indexCh}

			uploader := &sift.Uploader{
				Client:     mockClient,
				Framer:     framerSvc,
				ChannelSvc: dist.Channel,
			}

			params := sift.UploadParams{
				ClientKey: "key",
				AssetName: "asset",
				FlowName:  "flow",
				RunName:   "run",
				Channels:  channel.KeysFromChannels(channels),
				TimeRange: telem.TimeRange{Start: telem.SecondTS * 40, End: telem.SecondTS * 41},
			}

			err = uploader.Upload(ctx, params)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no valid channels"))
		})

		It("Should cancel upload when Stop is called", func() {
			// Use unbuffered channel so sends block until read
			requests := confluence.NewStream[*ingestv1.IngestWithConfigDataStreamRequest](0)
			mockClient, err := MustSucceed(client.NewMockFactory(client.MockFactoryConfig{
				Requests: requests,
			}))(ctx, "", "")
			Expect(err).ToNot(HaveOccurred())

			indexCh := channel.Channel{
				Name: "upload_cancel_index_1", IsIndex: true, DataType: telem.TimeStampT,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &indexCh)).To(Succeed())

			dataCh := channel.Channel{
				Name: "upload_cancel_data_1", DataType: telem.Float64T, LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &dataCh)).To(Succeed())
			channels := []channel.Channel{indexCh, dataCh}

			// Write a lot of data to make the upload take some time
			timeRange := telem.TimeRange{
				Start: telem.SecondTS * 60,
				End:   telem.SecondTS * 70,
			}
			w := MustSucceed(dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:  channel.KeysFromChannels(channels),
				Start: timeRange.Start,
				Sync:  config.True(),
			}))

			// Write multiple frames of data
			for i := range 10 {
				MustSucceed(w.Write(frame.NewMulti(
					channel.KeysFromChannels(channels),
					[]telem.Series{
						telem.NewSeriesSecondsTSV(telem.TimeStamp(60 + i)),
						telem.NewSeriesV(float64(i)),
					},
				)))
			}
			MustSucceed(w.Commit())
			Expect(w.Close()).To(Succeed())

			uploader := &sift.Uploader{
				Client:     mockClient,
				Framer:     framerSvc,
				ChannelSvc: dist.Channel,
			}

			params := sift.UploadParams{
				ClientKey: "key",
				AssetName: "asset",
				FlowName:  "flow",
				RunName:   "run",
				Channels:  channel.KeysFromChannels(channels),
				TimeRange: timeRange,
			}

			// Start upload in background
			done := make(chan error)
			go func() {
				done <- uploader.Upload(ctx, params)
			}()

			// Let the first request through
			<-requests.Outlet()

			// Stop the upload
			uploader.Stop()

			// Wait for upload to finish and check it was cancelled
			err = <-done
			Expect(err).To(MatchError(context.Canceled))
		})
	})
})
