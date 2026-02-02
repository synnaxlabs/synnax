// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package upload_test

import (
	"context"
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ingestv1 "github.com/sift-stack/sift/go/gen/sift/ingest/v1"
	ingestionconfigsv1 "github.com/sift-stack/sift/go/gen/sift/ingestion_configs/v1"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/sift/client"
	siftdevice "github.com/synnaxlabs/synnax/pkg/service/sift/device"
	"github.com/synnaxlabs/synnax/pkg/service/sift/upload"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Upload Task", func() {
	var (
		ctx        context.Context
		pool       *client.Pool
		mockClient client.Factory
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	AfterEach(func() {
		if pool != nil {
			Expect(pool.Close()).To(Succeed())
		}
	})

	createPool := func(mockCfg ...client.MockFactoryConfig) {
		mockClient = MustSucceed(client.NewMockFactory(mockCfg...))
		pool = client.NewPool(mockClient)
	}

	createDevice := func(key string) {
		props, _ := json.Marshal(siftdevice.Properties{
			URI:    "grpc-api.siftstack.com:443",
			APIKey: "sk-test-key",
		})
		dev := device.Device{
			Key:        key,
			Name:       "Test Sift Device",
			Make:       siftdevice.Make,
			Model:      siftdevice.Model,
			Properties: string(props),
			Rack:       testRack.Key,
			Location:   "test-location",
		}
		Expect(deviceSvc.NewWriter(nil).Create(ctx, dev)).To(Succeed())
	}

	deps := func() upload.Dependencies {
		return upload.Dependencies{
			Device:  deviceSvc,
			Framer:  framerSvc,
			Channel: dist.Channel,
			Status:  statusSvc,
			Task:    taskSvc,
			Pool:    pool,
			L:       nil,
		}
	}

	noopSetStatus := func(_ driver.Context, _ task.Task, _ xstatus.Variant, _ string, _ bool) {}

	Describe("Basic Upload", func() {
		It("Should upload data from channels to Sift", func() {
			requests := confluence.NewStream[*ingestv1.IngestWithConfigDataStreamRequest](10)
			createPool(client.MockFactoryConfig{Requests: requests})
			createDevice("sift-device-upload-1")

			indexCh := channel.Channel{
				Name: "upload_index_1", IsIndex: true, DataType: telem.TimeStampT,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &indexCh)).To(Succeed())

			dataCh := channel.Channel{
				Name: "upload_data_1", DataType: telem.Float64T, LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &dataCh)).To(Succeed())

			channels := []channel.Channel{indexCh, dataCh}

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

			taskCfg := upload.Config{
				DeviceKey: "sift-device-upload-1",
				AssetName: "test-asset",
				FlowName:  "telemetry",
				RunName:   "test-run",
				Channels:  channel.KeysFromChannels(channels),
				TimeRange: timeRange,
			}
			cfgBytes, _ := json.Marshal(taskCfg)

			t := task.Task{
				Key:    1,
				Name:   "Test Upload",
				Type:   upload.TaskType,
				Config: string(cfgBytes),
			}

			driverTask, err := upload.Configure(
				driver.NewContext(ctx, statusSvc),
				t,
				deps(),
				noopSetStatus,
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(driverTask).ToNot(BeNil())

			Eventually(requests.Outlet(), 5*time.Second).Should(Receive())
			Eventually(requests.Outlet(), 5*time.Second).Should(Receive())
			Eventually(requests.Outlet(), 5*time.Second).Should(Receive())
		})

		It("Should cancel upload when Stop is called", func() {
			requests := confluence.NewStream[*ingestv1.IngestWithConfigDataStreamRequest](0)
			createPool(client.MockFactoryConfig{Requests: requests})
			createDevice("sift-device-cancel-1")

			indexCh := channel.Channel{
				Name: "cancel_index_1", IsIndex: true, DataType: telem.TimeStampT,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &indexCh)).To(Succeed())

			dataCh := channel.Channel{
				Name: "cancel_data_1", DataType: telem.Float64T, LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &dataCh)).To(Succeed())

			channels := []channel.Channel{indexCh, dataCh}

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

			taskCfg := upload.Config{
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
				Type:   upload.TaskType,
				Config: string(cfgBytes),
			}

			driverTask, err := upload.Configure(
				driver.NewContext(ctx, statusSvc),
				t,
				deps(),
				noopSetStatus,
			)
			Expect(err).ToNot(HaveOccurred())

			<-requests.Outlet()

			Expect(driverTask.Stop()).To(Succeed())
		})

		It("Should upload data from multiple index groups to Sift", func() {
			requests := confluence.NewStream[*ingestv1.IngestWithConfigDataStreamRequest](20)
			var ingestionConfigCalls int
			createPool(client.MockFactoryConfig{
				Requests: requests,
				OnCreateIngestionConfig: func(
					_ *ingestionconfigsv1.CreateIngestionConfigRequest,
				) {
					ingestionConfigCalls++
				},
			})
			createDevice("sift-device-multi-index-1")

			indexA := channel.Channel{
				Name: "upload_multi_index_a", IsIndex: true, DataType: telem.TimeStampT,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &indexA)).To(Succeed())

			pressure1 := channel.Channel{
				Name: "upload_pressure_1", DataType: telem.Float64T, LocalIndex: indexA.LocalKey,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &pressure1)).To(Succeed())

			pressure2 := channel.Channel{
				Name: "upload_pressure_2", DataType: telem.Float64T, LocalIndex: indexA.LocalKey,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &pressure2)).To(Succeed())

			indexB := channel.Channel{
				Name: "upload_multi_index_b", IsIndex: true, DataType: telem.TimeStampT,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &indexB)).To(Succeed())

			temp1 := channel.Channel{
				Name: "upload_temp_1", DataType: telem.Float32T, LocalIndex: indexB.LocalKey,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &temp1)).To(Succeed())

			temp2 := channel.Channel{
				Name: "upload_temp_2", DataType: telem.Float32T, LocalIndex: indexB.LocalKey,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &temp2)).To(Succeed())

			Expect(indexA.LocalKey).ToNot(Equal(indexB.LocalKey))
			Expect(pressure1.Index()).To(Equal(indexA.Key()))
			Expect(pressure2.Index()).To(Equal(indexA.Key()))
			Expect(temp1.Index()).To(Equal(indexB.Key()))
			Expect(temp2.Index()).To(Equal(indexB.Key()))

			groupA := []channel.Channel{indexA, pressure1, pressure2}
			groupB := []channel.Channel{indexB, temp1, temp2}
			allChannels := append(groupA, groupB...)

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

			taskCfg := upload.Config{
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
				Type:   upload.TaskType,
				Config: string(cfgBytes),
			}

			driverTask, err := upload.Configure(
				driver.NewContext(ctx, statusSvc),
				t,
				deps(),
				noopSetStatus,
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(driverTask).ToNot(BeNil())

			Expect(ingestionConfigCalls).To(Equal(2))

			receivedRequests := make([]*ingestv1.IngestWithConfigDataStreamRequest, 0)
			for range 12 {
				var req *ingestv1.IngestWithConfigDataStreamRequest
				Eventually(requests.Outlet()).Should(Receive(&req))
				receivedRequests = append(receivedRequests, req)
			}

			configIDs := make(map[string]int)
			for _, req := range receivedRequests {
				configIDs[req.IngestionConfigId]++
			}
			Expect(configIDs).To(HaveLen(2))

			runIDs := make(map[string]bool)
			for _, req := range receivedRequests {
				runIDs[req.RunId] = true
			}
			Expect(runIDs).To(HaveLen(1))
		})
	})

	Describe("Edge Cases", func() {
		It("Should fail when no channels are provided", func() {
			createPool()
			createDevice("sift-device-empty")

			taskCfg := upload.Config{
				DeviceKey: "sift-device-empty",
				AssetName: "asset",
				FlowName:  "flow",
				RunName:   "run",
				Channels:  []channel.Key{},
				TimeRange: telem.TimeRange{Start: telem.SecondTS, End: telem.SecondTS * 2},
			}
			cfgBytes, _ := json.Marshal(taskCfg)

			_, err := upload.Configure(driver.NewContext(ctx, statusSvc), task.Task{
				Key: 100, Name: "Empty Channels", Type: upload.TaskType, Config: string(cfgBytes),
			}, deps(), noopSetStatus)
			Expect(err).To(HaveOccurred())
		})

		It("Should successfully upload index-only channels as INT64 data", func() {
			requests := confluence.NewStream[*ingestv1.IngestWithConfigDataStreamRequest](20)
			createPool(client.MockFactoryConfig{Requests: requests})
			createDevice("sift-device-index-only")

			indexCh := channel.Channel{
				Name: "upload_index_only_test", IsIndex: true, DataType: telem.TimeStampT,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &indexCh)).To(Succeed())

			timeRange := telem.TimeRange{Start: telem.SecondTS * 400, End: telem.SecondTS * 403}
			w := MustSucceed(dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:  []channel.Key{indexCh.Key()},
				Start: timeRange.Start,
			}))
			MustSucceed(w.Write(frame.NewMulti(
				[]channel.Key{indexCh.Key()},
				[]telem.Series{telem.NewSeriesSecondsTSV(400, 401, 402)},
			)))
			MustSucceed(w.Commit())
			Expect(w.Close()).To(Succeed())

			taskCfg := upload.Config{
				DeviceKey: "sift-device-index-only",
				AssetName: "asset", FlowName: "flow", RunName: "run",
				Channels:  []channel.Key{indexCh.Key()},
				TimeRange: timeRange,
			}
			cfgBytes, _ := json.Marshal(taskCfg)

			driverTask, err := upload.Configure(driver.NewContext(ctx, statusSvc), task.Task{
				Key: 101, Name: "Index Only", Type: upload.TaskType, Config: string(cfgBytes),
			}, deps(), noopSetStatus)
			Expect(err).ToNot(HaveOccurred())

			Eventually(requests.Outlet(), 5*time.Second).Should(Receive())
			Expect(driverTask.Stop()).To(Succeed())
		})

		It("Should include index channel as data when explicitly requested alongside data channels", func() {
			requests := confluence.NewStream[*ingestv1.IngestWithConfigDataStreamRequest](20)
			var ingestionConfigReq *ingestionconfigsv1.CreateIngestionConfigRequest
			createPool(client.MockFactoryConfig{
				Requests: requests,
				OnCreateIngestionConfig: func(req *ingestionconfigsv1.CreateIngestionConfigRequest) {
					ingestionConfigReq = req
				},
			})
			createDevice("sift-device-explicit-index")

			indexCh := channel.Channel{
				Name: "upload_explicit_index", IsIndex: true, DataType: telem.TimeStampT,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &indexCh)).To(Succeed())

			dataCh := channel.Channel{
				Name: "upload_data_with_index", DataType: telem.Float64T, LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &dataCh)).To(Succeed())

			timeRange := telem.TimeRange{Start: telem.SecondTS * 500, End: telem.SecondTS * 503}
			w := MustSucceed(dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:  []channel.Key{indexCh.Key(), dataCh.Key()},
				Start: timeRange.Start,
			}))
			MustSucceed(w.Write(frame.NewMulti(
				[]channel.Key{indexCh.Key(), dataCh.Key()},
				[]telem.Series{
					telem.NewSeriesSecondsTSV(500, 501, 502),
					telem.NewSeriesV(10.0, 20.0, 30.0),
				},
			)))
			MustSucceed(w.Commit())
			Expect(w.Close()).To(Succeed())

			taskCfg := upload.Config{
				DeviceKey: "sift-device-explicit-index",
				AssetName: "asset", FlowName: "flow", RunName: "run",
				Channels:  []channel.Key{indexCh.Key(), dataCh.Key()},
				TimeRange: timeRange,
			}
			cfgBytes, _ := json.Marshal(taskCfg)

			_, err := upload.Configure(driver.NewContext(ctx, statusSvc), task.Task{
				Key: 102, Name: "Explicit Index", Type: upload.TaskType, Config: string(cfgBytes),
			}, deps(), noopSetStatus)
			Expect(err).ToNot(HaveOccurred())

			Expect(ingestionConfigReq).ToNot(BeNil())
			Expect(ingestionConfigReq.Flows).To(HaveLen(1))
			channelNames := make([]string, 0)
			for _, ch := range ingestionConfigReq.Flows[0].Channels {
				channelNames = append(channelNames, ch.Name)
			}
			Expect(channelNames).To(ContainElements("upload_explicit_index", "upload_data_with_index"))
		})

		It("Should handle channels with no data in the time range", func() {
			requests := confluence.NewStream[*ingestv1.IngestWithConfigDataStreamRequest](10)
			createPool(client.MockFactoryConfig{Requests: requests})
			createDevice("sift-device-empty-range")

			indexCh := channel.Channel{
				Name: "upload_empty_range_index", IsIndex: true, DataType: telem.TimeStampT,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &indexCh)).To(Succeed())

			dataCh := channel.Channel{
				Name: "upload_empty_range_data", DataType: telem.Float64T, LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &dataCh)).To(Succeed())

			timeWritten := telem.TimeRange{Start: telem.SecondTS * 100, End: telem.SecondTS * 103}
			w := MustSucceed(dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:  []channel.Key{indexCh.Key(), dataCh.Key()},
				Start: timeWritten.Start,
			}))
			MustSucceed(w.Write(frame.NewMulti(
				[]channel.Key{indexCh.Key(), dataCh.Key()},
				[]telem.Series{
					telem.NewSeriesSecondsTSV(100, 101, 102),
					telem.NewSeriesV(1.0, 2.0, 3.0),
				},
			)))
			MustSucceed(w.Commit())
			Expect(w.Close()).To(Succeed())

			timeRange := telem.TimeRange{Start: telem.SecondTS * 900, End: telem.SecondTS * 903}
			taskCfg := upload.Config{
				DeviceKey: "sift-device-empty-range",
				AssetName: "asset", FlowName: "flow", RunName: "run",
				Channels:  []channel.Key{indexCh.Key(), dataCh.Key()},
				TimeRange: timeRange,
			}
			cfgBytes, _ := json.Marshal(taskCfg)

			driverTask, err := upload.Configure(driver.NewContext(ctx, statusSvc), task.Task{
				Key: 103, Name: "Empty Range", Type: upload.TaskType, Config: string(cfgBytes),
			}, deps(), noopSetStatus)
			Expect(err).ToNot(HaveOccurred())

			time.Sleep(100 * time.Millisecond)
			Expect(driverTask.Stop()).To(Succeed())
		})

		It("Should fail when any channel has an unsupported data type", func() {
			createPool()
			createDevice("sift-device-unsupported-type")

			indexCh := channel.Channel{
				Name: "upload_unsupported_index", IsIndex: true, DataType: telem.TimeStampT,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &indexCh)).To(Succeed())

			uuidCh := channel.Channel{
				Name: "upload_uuid_channel", DataType: telem.UUIDT, LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &uuidCh)).To(Succeed())

			timeRange := telem.TimeRange{Start: telem.SecondTS * 600, End: telem.SecondTS * 603}
			taskCfg := upload.Config{
				DeviceKey: "sift-device-unsupported-type",
				AssetName: "asset", FlowName: "flow", RunName: "run",
				Channels:  []channel.Key{indexCh.Key(), uuidCh.Key()},
				TimeRange: timeRange,
			}
			cfgBytes, _ := json.Marshal(taskCfg)

			_, err := upload.Configure(driver.NewContext(ctx, statusSvc), task.Task{
				Key: 104, Name: "Unsupported Type", Type: upload.TaskType, Config: string(cfgBytes),
			}, deps(), noopSetStatus)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported data type"))
		})
	})

	Describe("Error Cases", func() {
		It("Should fail with invalid JSON config", func() {
			createPool()
			_, err := upload.Configure(driver.NewContext(ctx, statusSvc), task.Task{
				Key: 200, Name: "Invalid JSON", Type: upload.TaskType, Config: "not json",
			}, deps(), noopSetStatus)
			Expect(err).To(HaveOccurred())
		})

		It("Should fail when device not found", func() {
			createPool()
			taskCfg := upload.Config{
				DeviceKey: "nonexistent-device",
				AssetName: "asset", FlowName: "flow", RunName: "run",
				Channels:  []channel.Key{1},
				TimeRange: telem.TimeRange{Start: telem.SecondTS, End: telem.SecondTS * 2},
			}
			cfgBytes, _ := json.Marshal(taskCfg)

			_, err := upload.Configure(driver.NewContext(ctx, statusSvc), task.Task{
				Key: 201, Name: "Device Not Found", Type: upload.TaskType, Config: string(cfgBytes),
			}, deps(), noopSetStatus)
			Expect(err).To(HaveOccurred())
		})

		It("Should fail with wrong device make", func() {
			createPool()
			props, _ := json.Marshal(siftdevice.Properties{URI: "uri", APIKey: "key"})
			dev := device.Device{
				Key: "wrong-make-device", Name: "Wrong Make",
				Make: "not-sift", Model: siftdevice.Model,
				Properties: string(props), Rack: testRack.Key, Location: "test",
			}
			Expect(deviceSvc.NewWriter(nil).Create(ctx, dev)).To(Succeed())

			taskCfg := upload.Config{
				DeviceKey: "wrong-make-device",
				AssetName: "asset", FlowName: "flow", RunName: "run",
				Channels:  []channel.Key{1},
				TimeRange: telem.TimeRange{Start: telem.SecondTS, End: telem.SecondTS * 2},
			}
			cfgBytes, _ := json.Marshal(taskCfg)

			_, err := upload.Configure(driver.NewContext(ctx, statusSvc), task.Task{
				Key: 202, Name: "Wrong Make", Type: upload.TaskType, Config: string(cfgBytes),
			}, deps(), noopSetStatus)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("make"))
		})

		It("Should fail when ingestion config creation fails", func() {
			createPool(client.MockFactoryConfig{
				ErrorOnCreateIngestionConfig: config.True(),
			})
			createDevice("sift-device-ingestion-fail")

			indexCh := channel.Channel{
				Name: "upload_ingestion_fail_index", IsIndex: true, DataType: telem.TimeStampT,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &indexCh)).To(Succeed())

			dataCh := channel.Channel{
				Name: "upload_ingestion_fail_data", DataType: telem.Float64T, LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &dataCh)).To(Succeed())

			taskCfg := upload.Config{
				DeviceKey: "sift-device-ingestion-fail",
				AssetName: "asset", FlowName: "flow", RunName: "run",
				Channels:  []channel.Key{indexCh.Key(), dataCh.Key()},
				TimeRange: telem.TimeRange{Start: telem.SecondTS * 700, End: telem.SecondTS * 703},
			}
			cfgBytes, _ := json.Marshal(taskCfg)

			_, err := upload.Configure(driver.NewContext(ctx, statusSvc), task.Task{
				Key: 203, Name: "Ingestion Fail", Type: upload.TaskType, Config: string(cfgBytes),
			}, deps(), noopSetStatus)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ingestion config"))
		})

		It("Should fail when run creation fails", func() {
			createPool(client.MockFactoryConfig{ErrorOnCreateRun: config.True()})
			createDevice("sift-device-run-fail")

			indexCh := channel.Channel{
				Name: "upload_run_fail_index", IsIndex: true, DataType: telem.TimeStampT,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &indexCh)).To(Succeed())

			dataCh := channel.Channel{
				Name: "upload_run_fail_data", DataType: telem.Float64T, LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &dataCh)).To(Succeed())

			taskCfg := upload.Config{
				DeviceKey: "sift-device-run-fail",
				AssetName: "asset", FlowName: "flow", RunName: "run",
				Channels:  []channel.Key{indexCh.Key(), dataCh.Key()},
				TimeRange: telem.TimeRange{Start: telem.SecondTS * 800, End: telem.SecondTS * 803},
			}
			cfgBytes, _ := json.Marshal(taskCfg)

			_, err := upload.Configure(driver.NewContext(ctx, statusSvc), task.Task{
				Key: 204, Name: "Run Fail", Type: upload.TaskType, Config: string(cfgBytes),
			}, deps(), noopSetStatus)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("run"))
		})

		It("Should fail when ingester open fails", func() {
			createPool(client.MockFactoryConfig{ErrorOnOpenIngester: config.True()})
			createDevice("sift-device-ingester-fail")

			indexCh := channel.Channel{
				Name: "upload_ingester_fail_index", IsIndex: true, DataType: telem.TimeStampT,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &indexCh)).To(Succeed())

			dataCh := channel.Channel{
				Name: "upload_ingester_fail_data", DataType: telem.Float64T, LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.NewWriter(nil).Create(ctx, &dataCh)).To(Succeed())

			taskCfg := upload.Config{
				DeviceKey: "sift-device-ingester-fail",
				AssetName: "asset", FlowName: "flow", RunName: "run",
				Channels:  []channel.Key{indexCh.Key(), dataCh.Key()},
				TimeRange: telem.TimeRange{Start: telem.SecondTS * 850, End: telem.SecondTS * 853},
			}
			cfgBytes, _ := json.Marshal(taskCfg)

			_, err := upload.Configure(driver.NewContext(ctx, statusSvc), task.Task{
				Key: 205, Name: "Ingester Fail", Type: upload.TaskType, Config: string(cfgBytes),
			}, deps(), noopSetStatus)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ingester"))
		})
	})
})
