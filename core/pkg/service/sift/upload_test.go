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
	"fmt"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/sift"
	"github.com/synnaxlabs/synnax/pkg/service/sift/fake"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	xuuid "github.com/synnaxlabs/x/uuid"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

// at returns the timestamp s seconds after the epoch.
func at(s int) telem.TimeStamp { return telem.TimeStamp(s) * telem.SecondTS }

var _ = Describe("Upload", func() {
	var (
		f      driver.Factory
		server *fake.Server
		t      task.Task
		dt     driver.Task
		dev    device.Device
		idx    channel.Channel
		ch     channel.Channel
		rng    ranger.Range
	)

	configure := func(ctx context.Context) {
		GinkgoHelper()
		dt = MustSucceed(f.ConfigureTask(ctx, t, driver.NoCommand))
	}

	BeforeEach(func(ctx SpecContext) {
		f = MustSucceed(sift.NewFactory(factoryConfig()))
		server = MustOpen(fake.Open(fake.Config{APIKey: apiKey}))
		t = newTask(ctx, sift.ScanTaskType, msgpack.EncodedJSON{"rate": 1})
		configure(ctx)
		DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
		dev = createDevice(ctx, sift.Properties{
			URI:    server.Address(),
			APIKey: createAPIKey(ctx, apiKey),
		})
		idx = createIndex(ctx)
		ch = createData(ctx, idx)
		write(ctx, idx, ch,
			[]telem.TimeStamp{at(1), at(2), at(3), at(4)},
			telem.NewSeriesV(1.0, 2.0, 3.0, 4.0),
		)
		rng = createRange(ctx, telem.TimeRange{Start: at(1), End: at(5)})
	})

	// upload queues an upload of ch in rng and returns the key of its status.
	upload := func(ctx context.Context) status.Key {
		GinkgoHelper()
		stat := execCommand(ctx, dt, t, "upload", map[string]any{
			"device":   dev.Key,
			"range":    rng.Key,
			"channels": []channel.Key{ch.Key()},
			"asset":    "stand",
		})
		Expect(stat.Variant).To(Equal(status.VariantSuccess))
		return stat.Details.Data["status"].(string)
	}

	statusOf := func(ctx context.Context, key status.Key) AsyncAssertion {
		return Eventually(func() (sift.UploadStatus, error) {
			return uploadStatus(ctx, key)
		}).WithTimeout(10 * time.Second)
	}

	It("Should send the data of the range and make a run of it", func(ctx SpecContext) {
		key := upload(ctx)
		statusOf(ctx, key).Should(SatisfyAll(
			HaveField("Name", rng.Name),
			HaveField("Variant", status.VariantSuccess),
			HaveField("Message", "Uploaded to Sift"),
			HaveField("Description", "Sent 4 samples to asset stand."),
		))
		Expect(server.Points("stand", ch.Name)).To(HaveLen(4))
		Expect(server.Runs()).To(HaveExactElements(SatisfyAll(
			HaveField("Name", rng.Name),
			HaveField(
				"ClientKey",
				HaveValue(Equal(xuuid.NewV5(rng.Key, "stand").String())),
			),
			HaveField("AssetIds", ConsistOf("asset-stand")),
		)))
		Expect(uploadStatus(ctx, key)).To(HaveField("Details.Range", rng.Key))
	})

	It("Should tag the run with the labels of the range", func(ctx SpecContext) {
		l := label.Label{Name: "hotfire"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := labelSvc.NewWriter(tx).Create(ctx, &l); err != nil {
				return err
			}
			return rangerSvc.NewWriter(tx).AddLabels(ctx, rng.Key, []label.Key{l.Key})
		})).To(Succeed())
		key := upload(ctx)
		statusOf(ctx, key).Should(HaveField("Variant", status.VariantSuccess))
		Expect(server.Runs()).To(HaveExactElements(
			HaveField("Tags", ConsistOf("hotfire")),
		))
	})

	It("Should count a single sample in the singular", func(ctx SpecContext) {
		idx := createIndex(ctx)
		ch = createData(ctx, idx)
		write(ctx, idx, ch, []telem.TimeStamp{at(10)}, telem.NewSeriesV(1.0))
		rng = createRange(ctx, telem.TimeRange{Start: at(10), End: at(12)})
		key := upload(ctx)
		statusOf(ctx, key).Should(
			HaveField("Description", "Sent 1 sample to asset stand."),
		)
	})

	It("Should group the digits of the sample count", func(ctx SpecContext) {
		idx := createIndex(ctx)
		ch = createData(ctx, idx)
		stamps := make([]telem.TimeStamp, 1234)
		for i := range stamps {
			stamps[i] = at(10) + telem.TimeStamp(i)*telem.MillisecondTS
		}
		write(ctx, idx, ch, stamps, telem.NewSeries(make([]float64, len(stamps))))
		rng = createRange(ctx, telem.TimeRange{Start: at(10), End: at(12)})
		key := upload(ctx)
		statusOf(ctx, key).Should(
			HaveField("Description", "Sent 1,234 samples to asset stand."),
		)
	})

	It("Should run the uploads of one device one at a time", func(ctx SpecContext) {
		server.Pause()
		first := upload(ctx)
		statusOf(ctx, first).Should(HaveField("Message", "Uploading to Sift"))
		second := upload(ctx)
		Consistently(func() (sift.UploadStatus, error) {
			return uploadStatus(ctx, second)
		}).WithTimeout(200 * time.Millisecond).
			Should(HaveField("Message", "Waiting to upload to Sift"))
		server.Resume()
		statusOf(ctx, first).Should(HaveField("Variant", status.VariantSuccess))
		statusOf(ctx, second).Should(HaveField("Variant", status.VariantSuccess))
		Expect(server.Runs()).To(HaveLen(1))
	})

	It("Should start the waiting uploads of a device oldest first",
		func(ctx SpecContext) {
			server.Pause()
			want := []string{rng.Name}
			upload(ctx)
			Eventually(server.Streams).Should(Equal(1))
			for range 3 {
				rng = createRange(ctx, telem.TimeRange{Start: at(1), End: at(5)})
				want = append(want, rng.Name)
				upload(ctx)
			}
			server.Resume()
			Eventually(func() []string {
				var names []string
				for _, r := range server.Runs() {
					names = append(names, r.Name)
				}
				return names
			}).WithTimeout(10 * time.Second).Should(Equal(want))
		})

	It("Should not run the uploads of a device on another rack",
		func(ctx SpecContext) {
			other := rack.Rack{Name: "other"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return rackSvc.NewWriter(tx).Create(ctx, &other)
			})).To(Succeed())
			moved := dev
			moved.Key = uuid.New().String()
			moved.Rack = other.Key
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return deviceSvc.NewWriter(tx).Create(ctx, &moved)
			})).To(Succeed())
			foreign := createRange(ctx, telem.TimeRange{Start: at(1), End: at(5)})
			s := sift.UploadStatus{
				Key:     "sift_upload:" + uuid.NewV7().String(),
				Name:    foreign.Name,
				Variant: status.VariantLoading,
				Message: "Waiting to upload to Sift",
				Time:    telem.Now(),
				Details: sift.UploadDetails{
					Device:    moved.Key,
					Range:     foreign.Key,
					TimeRange: foreign.TimeRange,
					Channels:  []channel.Key{ch.Key()},
					Asset:     "stand",
				},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return statusSvc.NewWriter(tx).Set(ctx, &s)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return statusSvc.NewWriter(tx).Delete(ctx, s.Key)
				})).To(Succeed())
			})
			upload(ctx)
			Eventually(server.Runs).WithTimeout(10 * time.Second).
				Should(ContainElement(HaveField("Name", rng.Name)))
			Consistently(server.Runs, 300*time.Millisecond).
				Should(HaveExactElements(HaveField("Name", rng.Name)))
			Expect(uploadStatus(ctx, s.Key)).
				To(HaveField("Message", "Waiting to upload to Sift"))
		})

	It("Should cancel an upload whose status is deleted", func(ctx SpecContext) {
		server.Pause()
		key := upload(ctx)
		statusOf(ctx, key).Should(HaveField("Message", "Uploading to Sift"))
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return statusSvc.NewWriter(tx).Delete(ctx, key)
		})).To(Succeed())
		server.Resume()
		Eventually(func() error {
			_, err := uploadStatus(ctx, key)
			return err
		}).Should(MatchError(query.ErrNotFound))
		Consistently(func() error {
			_, err := uploadStatus(ctx, key)
			return err
		}).WithTimeout(300 * time.Millisecond).Should(MatchError(query.ErrNotFound))
		Expect(server.Runs()).To(BeEmpty())
	})

	It("Should finish without a run when its range is deleted during the upload",
		func(ctx SpecContext) {
			server.Pause()
			key := upload(ctx)
			Eventually(server.Streams).Should(Equal(1))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return rangerSvc.NewWriter(tx).Delete(ctx, rng.Key)
			})).To(Succeed())
			server.Resume()
			statusOf(ctx, key).Should(SatisfyAll(
				HaveField("Variant", status.VariantSuccess),
				HaveField("Message", "Uploaded to Sift"),
			))
			Expect(server.Points("stand", ch.Name)).To(HaveLen(4))
			Expect(server.Runs()).To(BeEmpty())
		})

	It("Should not write back the status of an upload deleted as it ends",
		func(ctx SpecContext) {
			server.FailRuns(grpcstatus.Error(codes.Unavailable, "runs are down"))
			key := upload(ctx)
			statusOf(ctx, key).Should(HaveField("Variant", status.VariantWarning))
			// The open transaction holds the write lock, so the final status write
			// waits for the delete to commit.
			tx := DeferClose(db.OpenTx())
			Expect(statusSvc.NewWriter(tx).Delete(ctx, key)).To(Succeed())
			Eventually(server.Runs).WithTimeout(10 * time.Second).Should(HaveLen(1))
			// Lets the upload reach the final status write.
			Consistently(server.Runs, 100*time.Millisecond).Should(HaveLen(1))
			Expect(tx.Commit(ctx)).To(Succeed())
			Consistently(func() error {
				_, err := uploadStatus(ctx, key)
				return err
			}).WithTimeout(300 * time.Millisecond).Should(MatchError(query.ErrNotFound))
		})

	It("Should not start a waiting upload whose status is deleted",
		func(ctx SpecContext) {
			server.Pause()
			first := upload(ctx)
			statusOf(ctx, first).Should(HaveField("Message", "Uploading to Sift"))
			second := upload(ctx)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return statusSvc.NewWriter(tx).Delete(ctx, second)
			})).To(Succeed())
			server.Resume()
			statusOf(ctx, first).Should(HaveField("Variant", status.VariantSuccess))
			Consistently(func() error {
				_, err := uploadStatus(ctx, second)
				return err
			}).WithTimeout(300 * time.Millisecond).Should(MatchError(query.ErrNotFound))
			Expect(server.Points("stand", ch.Name)).To(HaveLen(4))
		})

	It("Should warn while it retries a temporary Sift failure", func(ctx SpecContext) {
		server.Fail(grpcstatus.Error(codes.Unavailable, "Sift is down"))
		key := upload(ctx)
		statusOf(ctx, key).Should(SatisfyAll(
			HaveField("Variant", status.VariantWarning),
			HaveField("Message", "Retrying upload to Sift"),
			HaveField("Description", ContainSubstring("Sift is down")),
		))
		statusOf(ctx, key).Should(HaveField("Variant", status.VariantSuccess))
		Expect(server.Points("stand", ch.Name)).To(HaveLen(4))
	})

	It("Should send the data written so far of a range that has not ended",
		func(ctx SpecContext) {
			rng = createRange(ctx, telem.TimeRange{
				Start: at(1),
				End:   telem.Now().Add(telem.Hour),
			})
			before := telem.Now()
			key := upload(ctx)
			statusOf(ctx, key).Should(HaveField("Variant", status.VariantSuccess))
			Expect(server.Points("stand", ch.Name)).To(HaveLen(4))
			Expect(uploadStatus(ctx, key)).To(HaveField(
				"Details.TimeRange.End",
				SatisfyAll(
					BeNumerically(">=", before),
					BeNumerically("<=", telem.Now()),
				),
			))
		})

	It("Should resume an upload after the task restarts", func(ctx SpecContext) {
		server.Pause()
		key := upload(ctx)
		statusOf(ctx, key).Should(HaveField("Message", "Uploading to Sift"))
		stopped := make(chan error, 1)
		go func() { stopped <- dt.Stop(false) }()
		server.Resume()
		Eventually(stopped).WithTimeout(10 * time.Second).Should(Receive(Succeed()))
		Expect(uploadStatus(ctx, key)).To(HaveField("Variant", status.VariantLoading))
		configure(ctx)
		statusOf(ctx, key).Should(HaveField("Variant", status.VariantSuccess))
		Expect(server.Points("stand", ch.Name)).To(HaveLen(4))
		Expect(server.Runs()).To(HaveLen(1))
	})

	It("Should resume from its saved positions an upload that was retrying",
		func(ctx SpecContext) {
			server.FailRuns(
				grpcstatus.Error(codes.Unavailable, "runs are down"),
				grpcstatus.Error(codes.Unavailable, "runs are down"),
			)
			key := upload(ctx)
			statusOf(ctx, key).Should(HaveField("Variant", status.VariantWarning))
			Expect(dt.Stop(false)).To(Succeed())
			Expect(uploadStatus(ctx, key)).To(SatisfyAll(
				HaveField("Variant", status.VariantWarning),
				HaveField("Details.Samples", BeEquivalentTo(4)),
				HaveField(
					"Details.Positions",
					HaveExactElements(HaveField("Index", idx.Key())),
				),
			))
			received := server.Received()
			configure(ctx)
			statusOf(ctx, key).Should(SatisfyAll(
				HaveField("Variant", status.VariantSuccess),
				HaveField("Description", "Sent 4 samples to asset stand."),
			))
			Expect(server.Received()).To(Equal(received))
			Expect(server.Runs()).To(HaveLen(1))
		})

	It("Should fail a waiting upload whose device is deleted", func(ctx SpecContext) {
		server.Pause()
		first := upload(ctx)
		statusOf(ctx, first).Should(HaveField("Message", "Uploading to Sift"))
		second := upload(ctx)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return deviceSvc.NewWriter(tx).Delete(ctx, dev.Key)
		})).To(Succeed())
		server.Resume()
		statusOf(ctx, second).Should(SatisfyAll(
			HaveField("Variant", status.VariantError),
			HaveField("Message", "Failed to upload to Sift"),
			HaveField("Description", ContainSubstring(
				fmt.Sprintf("Devices with keys [%s] not found", dev.Key),
			)),
		))
	})

	It("Should hand the uploads to a scan task configured while the last one stops",
		func(ctx SpecContext) {
			server.Pause()
			key := upload(ctx)
			// The stop waits for Sift to confirm only what the upload sent.
			Eventually(server.Streams).Should(Equal(1))
			last := dt
			stopped := make(chan error, 1)
			go func() { stopped <- last.Stop(false) }()
			// Lets the stop start before the next scan task configures.
			Consistently(stopped, 50*time.Millisecond).ShouldNot(Receive())
			scan := newTask(ctx, sift.ScanTaskType, msgpack.EncodedJSON{"rate": 1})
			var next driver.Task
			configured := make(chan error, 1)
			go func() {
				var err error
				next, err = f.ConfigureTask(ctx, scan, driver.NoCommand)
				configured <- err
			}()
			Consistently(configured, 50*time.Millisecond).ShouldNot(Receive())
			server.Resume()
			Eventually(stopped).WithTimeout(10 * time.Second).Should(Receive(Succeed()))
			Eventually(configured).Should(Receive(Succeed()))
			dt = next
			statusOf(ctx, key).Should(HaveField("Variant", status.VariantSuccess))
			Expect(server.Points("stand", ch.Name)).To(HaveLen(4))
		})

	It("Should fail an upload that Sift rejects", func(ctx SpecContext) {
		dev = createDevice(ctx, sift.Properties{
			URI:    server.Address(),
			APIKey: createAPIKey(ctx, "wrong"),
		})
		key := upload(ctx)
		statusOf(ctx, key).Should(SatisfyAll(
			HaveField("Variant", status.VariantError),
			HaveField("Message", "Failed to upload to Sift"),
			HaveField("Description", ContainSubstring("Sift rejected the API key")),
		))
	})

	missingDevice, missingRange := uuid.New().String(), uuid.New()

	DescribeTable("Should reject an invalid upload",
		func(ctx SpecContext, args func() map[string]any, msg string) {
			stat := execCommand(ctx, dt, t, "upload", args())
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Message).To(Equal("Failed to upload to Sift"))
			Expect(stat.Description).To(ContainSubstring(msg))
		},
		Entry("no device", func() map[string]any {
			return map[string]any{
				"range":    rng.Key,
				"channels": []channel.Key{ch.Key()},
				"asset":    "stand",
			}
		}, "device: required"),
		Entry("no range", func() map[string]any {
			return map[string]any{
				"device":   dev.Key,
				"channels": []channel.Key{ch.Key()},
				"asset":    "stand",
			}
		}, "range: required"),
		Entry("a device that does not exist", func() map[string]any {
			return map[string]any{
				"device":   missingDevice,
				"range":    rng.Key,
				"channels": []channel.Key{ch.Key()},
				"asset":    "stand",
			}
		}, fmt.Sprintf("Devices with keys [%s] not found", missingDevice)),
		Entry("a device that is not a Sift device", func() map[string]any {
			GinkgoHelper()
			ctx := context.Background()
			other := dev
			other.Key = uuid.New().String()
			other.Make = "other"
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return deviceSvc.NewWriter(tx).Create(ctx, &other)
			})).To(Succeed())
			return map[string]any{
				"device":   other.Key,
				"range":    rng.Key,
				"channels": []channel.Key{ch.Key()},
				"asset":    "stand",
			}
		}, "is not a Sift device"),
		Entry("a range that does not exist", func() map[string]any {
			return map[string]any{
				"device":   dev.Key,
				"range":    missingRange,
				"channels": []channel.Key{ch.Key()},
				"asset":    "stand",
			}
		}, fmt.Sprintf("Ranges with keys [%s] not found", missingRange)),
		Entry("no asset", func() map[string]any {
			return map[string]any{
				"device":   dev.Key,
				"range":    rng.Key,
				"channels": []channel.Key{ch.Key()},
			}
		}, "asset: required"),
		Entry("no channels", func() map[string]any {
			return map[string]any{
				"device": dev.Key,
				"range":  rng.Key,
				"asset":  "stand",
			}
		}, "channels: must be non-empty"),
		Entry("a channel that Sift cannot take", func() map[string]any {
			GinkgoHelper()
			ctx := context.Background()
			virtual := channel.Channel{
				Name:     "virtual_" + uuid.New().String()[:8],
				DataType: telem.Float64T,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, &virtual)
			})).To(Succeed())
			return map[string]any{
				"device":   dev.Key,
				"range":    rng.Key,
				"channels": []channel.Key{virtual.Key()},
				"asset":    "stand",
			}
		}, "is virtual, and the Core stores no data for virtual channels"),
		Entry("a device on another rack", func() map[string]any {
			GinkgoHelper()
			ctx := context.Background()
			other := rack.Rack{Name: "other"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return rackSvc.NewWriter(tx).Create(ctx, &other)
			})).To(Succeed())
			moved := dev
			moved.Key = uuid.New().String()
			moved.Rack = other.Key
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return deviceSvc.NewWriter(tx).Create(ctx, &moved)
			})).To(Succeed())
			return map[string]any{
				"device":   moved.Key,
				"range":    rng.Key,
				"channels": []channel.Key{ch.Key()},
				"asset":    "stand",
			}
		}, "belongs to another rack"),
		Entry("a range that has not started", func() map[string]any {
			future := createRange(context.Background(), telem.TimeRange{
				Start: telem.Now().Add(telem.Hour),
				End:   telem.Now().Add(2 * telem.Hour),
			})
			return map[string]any{
				"device":   dev.Key,
				"range":    future.Key,
				"channels": []channel.Key{ch.Key()},
				"asset":    "stand",
			}
		}, "starts in the future, so it has no data yet"),
		Entry("a range with no time span", func() map[string]any {
			now := telem.Now()
			empty := createRange(context.Background(), telem.TimeRange{
				Start: now,
				End:   now,
			})
			return map[string]any{
				"device":   dev.Key,
				"range":    empty.Key,
				"channels": []channel.Key{ch.Key()},
				"asset":    "stand",
			}
		}, "has no time span to upload"),
	)
})
