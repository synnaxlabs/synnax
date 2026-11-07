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
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/sift"
	"github.com/synnaxlabs/synnax/pkg/service/sift/fake"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
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
		t = task.Task{
			Key:    uuid.New(),
			Rack:   rackSvc.EmbeddedKey,
			Name:   "Sift Scanner",
			Type:   sift.ScanTaskType,
			Config: msgpack.EncodedJSON{"rate": 1},
		}
		configure(ctx)
		DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
		dev = createDevice(ctx, sift.Properties{
			URI:    server.Address(),
			APIKey: createAPIKey(ctx, apiKey),
		})
		idx := createIndex(ctx)
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
			HaveField("ClientKey", HaveValue(Equal(rng.Key.String()))),
			HaveField("AssetIds", ConsistOf("asset-stand")),
		)))
		var children []ontology.Resource
		Expect(otg.NewRetrieve().
			WhereIDs(ranger.OntologyID(rng.Key)).
			TraverseTo(ontology.ChildrenTraverser).
			Entries(&children).
			Exec(ctx, nil)).To(Succeed())
		Expect(children).To(ContainElement(
			HaveField("ID", status.OntologyID(key)),
		))
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

	It("Should cancel an upload whose status is deleted", func(ctx SpecContext) {
		server.Pause()
		key := upload(ctx)
		statusOf(ctx, key).Should(HaveField("Message", "Uploading to Sift"))
		Expect(statusSvc.NewWriter(nil).Delete(ctx, key)).To(Succeed())
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

	DescribeTable("Should reject an invalid upload",
		func(ctx SpecContext, args func() map[string]any, msg string) {
			stat := execCommand(ctx, dt, t, "upload", args())
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Message).To(ContainSubstring(msg))
		},
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
		}, "channels"),
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
		}, "has no time span to upload"),
	)
})
