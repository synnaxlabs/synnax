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
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	typev1 "github.com/sift-stack/sift/go/gen/sift/common/type/v1"
	ingestv1 "github.com/sift-stack/sift/go/gen/sift/ingest/v1"
	ingestionconfigsv1 "github.com/sift-stack/sift/go/gen/sift/ingestion_configs/v1"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/sift"
	"github.com/synnaxlabs/synnax/pkg/service/sift/client"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Sift", func() {
	Describe("GroupChannelsByIndex", func() {
		It("Should group channels by their index", func() {
			// Create mock channels with different indexes
			channels := []channel.Channel{
				{
					Name:        "indexA",
					IsIndex:     true,
					DataType:    telem.TimeStampT,
					LocalKey:    1,
					Leaseholder: 1,
				},
				{
					Name:        "pressure1",
					DataType:    telem.Float64T,
					LocalKey:    2,
					LocalIndex:  1, // References indexA
					Leaseholder: 1,
				},
				{
					Name:        "pressure2",
					DataType:    telem.Float64T,
					LocalKey:    3,
					LocalIndex:  1, // References indexA
					Leaseholder: 1,
				},
				{
					Name:        "indexB",
					IsIndex:     true,
					DataType:    telem.TimeStampT,
					LocalKey:    4,
					Leaseholder: 1,
				},
				{
					Name:        "temp1",
					DataType:    telem.Float32T,
					LocalKey:    5,
					LocalIndex:  4, // References indexB
					Leaseholder: 1,
				},
				{
					Name:        "temp2",
					DataType:    telem.Float32T,
					LocalKey:    6,
					LocalIndex:  4, // References indexB
					Leaseholder: 1,
				},
			}

			groups := sift.GroupChannelsByIndex(channels)
			Expect(groups).To(HaveLen(2))

			// Check group A (index key = NewKey(1, 1) for LocalIndex 1)
			indexAKey := channel.NewKey(1, 1)
			Expect(groups[indexAKey]).ToNot(BeNil())
			Expect(groups[indexAKey].DataChannelKeys).To(HaveLen(2))

			// Check group B (index key = NewKey(1, 4) for LocalIndex 4)
			indexBKey := channel.NewKey(1, 4)
			Expect(groups[indexBKey]).ToNot(BeNil())
			Expect(groups[indexBKey].DataChannelKeys).To(HaveLen(2))
		})
	})

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
	Describe("Factory", func() {
		var (
			ctx     context.Context
			factory *sift.Factory
		)

		BeforeEach(func() {
			ctx = context.Background()
		})

		AfterEach(func() {
			if factory != nil {
				Expect(factory.Close()).To(Succeed())
			}
		})

		createFactory := func(mockCfg ...client.MockFactoryConfig) {
			factory = MustSucceed(sift.OpenFactory(sift.FactoryConfig{
				Device:        deviceSvc,
				Framer:        framerSvc,
				Channel:       dist.Channel,
				Status:        statusSvc,
				Task:          taskSvc,
				ClientFactory: MustSucceed(client.NewMockFactory(mockCfg...)),
			}))
		}

		createDevice := func(key string) {
			props, _ := json.Marshal(sift.DeviceProperties{
				URI:    "grpc-api.siftstack.com:443",
				APIKey: "sk-test-key",
			})
			dev := device.Device{
				Key:        key,
				Name:       "Test Sift Device",
				Make:       sift.DeviceMake,
				Model:      sift.DeviceModel,
				Properties: string(props),
				Rack:       testRack.Key,
				Location:   "test-location",
			}
			Expect(deviceSvc.NewWriter(nil).Create(ctx, dev)).To(Succeed())
		}

		It("Should return ErrTaskNotHandled for unknown task types", func() {
			createFactory()
			_, err := factory.ConfigureTask(driver.Context{Context: ctx}, task.Task{
				Type: "unknown_type",
			})
			Expect(err).To(MatchError(driver.ErrTaskNotHandled))
		})

		Describe("Upload Task", func() {
			It("Should upload data from channels to Sift", func() {
				requests := confluence.NewStream[*ingestv1.IngestWithConfigDataStreamRequest](10)
				createFactory(client.MockFactoryConfig{Requests: requests})
				createDevice("sift-device-upload-1")

				// Create channels
				indexCh := channel.Channel{
					Name: "factory_upload_index_1", IsIndex: true, DataType: telem.TimeStampT,
				}
				Expect(dist.Channel.NewWriter(nil).Create(ctx, &indexCh)).To(Succeed())

				dataCh := channel.Channel{
					Name: "factory_upload_data_1", DataType: telem.Float64T, LocalIndex: indexCh.LocalKey,
				}
				Expect(dist.Channel.NewWriter(nil).Create(ctx, &dataCh)).To(Succeed())

				channels := []channel.Channel{indexCh, dataCh}

				// Write test data
				timeRange := telem.TimeRange{
					Start: telem.SecondTS * 100,
					End:   telem.SecondTS * 103,
				}
				w := MustSucceed(dist.Framer.OpenWriter(ctx, writer.Config{
					Keys:  channel.KeysFromChannels(channels),
					Start: timeRange.Start,
					Sync:  config.True(),
				}))
				MustSucceed(w.Write(frame.NewMulti(
					channel.KeysFromChannels(channels),
					[]telem.Series{
						telem.NewSeriesSecondsTSV(100, 101, 102),
						telem.NewSeriesV(1.0, 2.0, 3.0),
					},
				)))
				MustSucceed(w.Commit())
				Expect(w.Close()).To(Succeed())

				// Create task config
				taskCfg := sift.UploadTaskConfig{
					DeviceKey: "sift-device-upload-1",
					AssetName: "test-asset",
					FlowName:  "telemetry",
					RunName:   "test-run",
					Channels:  channel.KeysFromChannels(channels),
					TimeRange: timeRange,
				}
				cfgBytes, _ := json.Marshal(taskCfg)

				// Configure task
				t := task.Task{
					Key:    1,
					Name:   "Test Upload",
					Type:   sift.UploadTaskType,
					Config: string(cfgBytes),
				}

				driverTask, err := factory.ConfigureTask(
					driver.NewContext(ctx, statusSvc),
					t,
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(driverTask).ToNot(BeNil())

				// Wait for upload to complete by receiving requests
				Eventually(requests.Outlet(), 5*time.Second).Should(Receive())
				Eventually(requests.Outlet(), 5*time.Second).Should(Receive())
				Eventually(requests.Outlet(), 5*time.Second).Should(Receive())
			})

			It("Should cancel upload when Stop is called", func() {
				// Use unbuffered channel so sends block
				requests := confluence.NewStream[*ingestv1.IngestWithConfigDataStreamRequest](0)
				createFactory(client.MockFactoryConfig{Requests: requests})
				createDevice("sift-device-cancel-1")

				// Create channels
				indexCh := channel.Channel{
					Name: "factory_cancel_index_1", IsIndex: true, DataType: telem.TimeStampT,
				}
				Expect(dist.Channel.NewWriter(nil).Create(ctx, &indexCh)).To(Succeed())

				dataCh := channel.Channel{
					Name: "factory_cancel_data_1", DataType: telem.Float64T, LocalIndex: indexCh.LocalKey,
				}
				Expect(dist.Channel.NewWriter(nil).Create(ctx, &dataCh)).To(Succeed())

				channels := []channel.Channel{indexCh, dataCh}

				// Write more data to make upload take time
				timeRange := telem.TimeRange{
					Start: telem.SecondTS * 200,
					End:   telem.SecondTS * 210,
				}
				w := MustSucceed(dist.Framer.OpenWriter(ctx, writer.Config{
					Keys:  channel.KeysFromChannels(channels),
					Start: timeRange.Start,
					Sync:  config.True(),
				}))
				for i := range 10 {
					MustSucceed(w.Write(frame.NewMulti(
						channel.KeysFromChannels(channels),
						[]telem.Series{
							telem.NewSeriesSecondsTSV(telem.TimeStamp(200 + i)),
							telem.NewSeriesV(float64(i)),
						},
					)))
				}
				MustSucceed(w.Commit())
				Expect(w.Close()).To(Succeed())

				// Create task config
				taskCfg := sift.UploadTaskConfig{
					DeviceKey: "sift-device-cancel-1",
					AssetName: "asset",
					FlowName:  "flow",
					RunName:   "run",
					Channels:  channel.KeysFromChannels(channels),
					TimeRange: timeRange,
				}
				cfgBytes, _ := json.Marshal(taskCfg)

				t := task.Task{
					Key:    2,
					Name:   "Test Cancel",
					Type:   sift.UploadTaskType,
					Config: string(cfgBytes),
				}

				driverTask, err := factory.ConfigureTask(
					driver.NewContext(ctx, statusSvc),
					t,
				)
				Expect(err).ToNot(HaveOccurred())

				// Let first request through
				<-requests.Outlet()

				// Stop the task
				Expect(driverTask.Stop()).To(Succeed())
			})

			It("Should upload data from multiple index groups to Sift", func() {
				requests := confluence.NewStream[*ingestv1.IngestWithConfigDataStreamRequest](20)
				var ingestionConfigCalls int
				createFactory(client.MockFactoryConfig{
					Requests: requests,
					OnCreateIngestionConfig: func(
						_ *ingestionconfigsv1.CreateIngestionConfigRequest,
					) {
						ingestionConfigCalls++
					},
				})
				createDevice("sift-device-multi-index-1")

				// Create first index and its data channels
				indexA := channel.Channel{
					Name: "multi_index_a", IsIndex: true, DataType: telem.TimeStampT,
				}
				Expect(dist.Channel.NewWriter(nil).Create(ctx, &indexA)).To(Succeed())

				pressure1 := channel.Channel{
					Name: "pressure_1", DataType: telem.Float64T, LocalIndex: indexA.LocalKey,
				}
				Expect(dist.Channel.NewWriter(nil).Create(ctx, &pressure1)).To(Succeed())

				pressure2 := channel.Channel{
					Name: "pressure_2", DataType: telem.Float64T, LocalIndex: indexA.LocalKey,
				}
				Expect(dist.Channel.NewWriter(nil).Create(ctx, &pressure2)).To(Succeed())

				// Create second index and its data channels
				indexB := channel.Channel{
					Name: "multi_index_b", IsIndex: true, DataType: telem.TimeStampT,
				}
				Expect(dist.Channel.NewWriter(nil).Create(ctx, &indexB)).To(Succeed())

				temp1 := channel.Channel{
					Name: "temp_1", DataType: telem.Float32T, LocalIndex: indexB.LocalKey,
				}
				Expect(dist.Channel.NewWriter(nil).Create(ctx, &temp1)).To(Succeed())

				temp2 := channel.Channel{
					Name: "temp_2", DataType: telem.Float32T, LocalIndex: indexB.LocalKey,
				}
				Expect(dist.Channel.NewWriter(nil).Create(ctx, &temp2)).To(Succeed())

				// Verify the channels have different index keys
				Expect(indexA.LocalKey).ToNot(Equal(indexB.LocalKey))
				Expect(pressure1.Index()).To(Equal(indexA.Key()))
				Expect(pressure2.Index()).To(Equal(indexA.Key()))
				Expect(temp1.Index()).To(Equal(indexB.Key()))
				Expect(temp2.Index()).To(Equal(indexB.Key()))

				// Verify grouping works correctly with created channels
				allCreatedChannels := []channel.Channel{indexA, pressure1, pressure2, indexB, temp1, temp2}
				groups := sift.GroupChannelsByIndex(allCreatedChannels)
				Expect(groups).To(HaveLen(2))

				// Also verify grouping works with retrieved channels
				allKeys := channel.KeysFromChannels(allCreatedChannels)
				var retrievedChannels []channel.Channel
				Expect(dist.Channel.NewRetrieve().
					WhereKeys(allKeys...).
					Entries(&retrievedChannels).
					Exec(ctx, nil)).To(Succeed())
				Expect(retrievedChannels).To(HaveLen(6))

				retrievedGroups := sift.GroupChannelsByIndex(retrievedChannels)
				Expect(retrievedGroups).To(HaveLen(2))

				groupA := []channel.Channel{indexA, pressure1, pressure2}
				groupB := []channel.Channel{indexB, temp1, temp2}
				allChannels := append(groupA, groupB...)

				// Write test data for group A
				timeRange := telem.TimeRange{
					Start: telem.SecondTS * 300,
					End:   telem.SecondTS * 303,
				}
				wA := MustSucceed(dist.Framer.OpenWriter(ctx, writer.Config{
					Keys:  channel.KeysFromChannels(groupA),
					Start: timeRange.Start,
					Sync:  config.True(),
				}))
				MustSucceed(wA.Write(frame.NewMulti(
					channel.KeysFromChannels(groupA),
					[]telem.Series{
						telem.NewSeriesSecondsTSV(300, 301, 302),
						telem.NewSeriesV(100.0, 101.0, 102.0),
						telem.NewSeriesV(200.0, 201.0, 202.0),
					},
				)))
				MustSucceed(wA.Commit())
				Expect(wA.Close()).To(Succeed())

				// Write test data for group B
				wB := MustSucceed(dist.Framer.OpenWriter(ctx, writer.Config{
					Keys:  channel.KeysFromChannels(groupB),
					Start: timeRange.Start,
					Sync:  config.True(),
				}))
				MustSucceed(wB.Write(frame.NewMulti(
					channel.KeysFromChannels(groupB),
					[]telem.Series{
						telem.NewSeriesSecondsTSV(300, 301, 302),
						telem.NewSeriesV[float32](25.0, 26.0, 27.0),
						telem.NewSeriesV[float32](30.0, 31.0, 32.0),
					},
				)))
				MustSucceed(wB.Commit())
				Expect(wB.Close()).To(Succeed())

				// Create task config with all channels
				taskCfg := sift.UploadTaskConfig{
					DeviceKey: "sift-device-multi-index-1",
					AssetName: "test-asset",
					FlowName:  "telemetry",
					RunName:   "multi-index-run",
					Channels:  channel.KeysFromChannels(allChannels),
					TimeRange: timeRange,
				}
				cfgBytes, _ := json.Marshal(taskCfg)

				t := task.Task{
					Key:    3,
					Name:   "Test Multi-Index Upload",
					Type:   sift.UploadTaskType,
					Config: string(cfgBytes),
				}

				driverTask, err := factory.ConfigureTask(
					driver.NewContext(ctx, statusSvc),
					t,
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(driverTask).ToNot(BeNil())

				// Verify 2 ingestion configs were created (one per group)
				Expect(ingestionConfigCalls).To(Equal(2))

				// Collect requests and verify we get data from both groups
				// Each group has 2 data channels × 3 samples = 6 samples per group.
				// With 2 groups, that's 12 total requests.
				// The iterator may return data in separate frames per channel.
				receivedRequests := make([]*ingestv1.IngestWithConfigDataStreamRequest, 0)
				for range 12 {
					var req *ingestv1.IngestWithConfigDataStreamRequest
					Eventually(requests.Outlet(), 5*time.Second).Should(Receive(&req))
					receivedRequests = append(receivedRequests, req)
				}

				// Verify we received requests with different ingestion config IDs
				configIDs := make(map[string]int)
				for _, req := range receivedRequests {
					configIDs[req.IngestionConfigId]++
				}
				// Should have exactly 2 different config IDs (one per group)
				Expect(configIDs).To(HaveLen(2))

				// All requests should share the same run ID
				runIDs := make(map[string]bool)
				for _, req := range receivedRequests {
					runIDs[req.RunId] = true
				}
				Expect(runIDs).To(HaveLen(1))
			})
		})
	})
})
