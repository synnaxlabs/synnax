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
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/sift"
	"github.com/synnaxlabs/synnax/pkg/service/sift/fake"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

const apiKey = "sift-api-key"

var _ = Describe("Factory", func() {
	var f driver.Factory

	BeforeEach(func() {
		f = MustSucceed(sift.NewFactory(sift.FactoryConfig{
			DB:      db,
			Device:  deviceSvc,
			Status:  statusSvc,
			Secret:  secretSvc,
			Timeout: 2 * telem.Second,
		}))
	})

	Describe("NewFactory", func() {
		It("Should require the secret service", func() {
			Expect(sift.NewFactory(sift.FactoryConfig{
				DB:     db,
				Device: deviceSvc,
				Status: statusSvc,
			})).Error().To(MatchError(ContainSubstring("secret: must be non-nil")))
		})
	})

	Describe("Name", func() {
		It("Should return the Sift make", func() {
			Expect(f.Name()).To(Equal(sift.Make))
		})
	})

	Describe("InitialTasks", func() {
		It("Should return one scan task", func() {
			Expect(f.InitialTasks()).To(HaveExactElements(
				HaveField("Type", sift.ScanTaskType),
			))
		})
	})

	Describe("ConfigureTask", func() {
		It("Should not handle another task type", func(ctx SpecContext) {
			t := task.Task{
				Key:  uuid.New(),
				Rack: rackSvc.EmbeddedKey,
				Type: "kafka_read",
			}
			Expect(f.ConfigureTask(ctx, t, driver.NoCommand)).Error().
				To(MatchError(driver.ErrTaskNotHandled))
		})

		It("Should reject a scan rate that is not positive", func(ctx SpecContext) {
			t := task.Task{
				Key:    uuid.New(),
				Rack:   rackSvc.EmbeddedKey,
				Type:   sift.ScanTaskType,
				Config: msgpack.EncodedJSON{"rate": -1},
			}
			Expect(f.ConfigureTask(ctx, t, driver.NoCommand)).Error().
				To(MatchError(ContainSubstring("rate: must be positive")))
		})
	})

	Describe("Scan task", func() {
		var (
			server *fake.Server
			t      task.Task
			dt     driver.Task
		)

		BeforeEach(func(ctx SpecContext) {
			server = MustOpen(fake.Open(fake.Config{
				APIKey: apiKey,
				Assets: []string{"stand", "pad"},
			}))
			t = task.Task{
				Key:    uuid.New(),
				Rack:   rackSvc.EmbeddedKey,
				Name:   "Sift Scanner",
				Type:   sift.ScanTaskType,
				Config: msgpack.EncodedJSON{"rate": 50},
			}
			dt = MustSucceed(f.ConfigureTask(ctx, t, driver.NoCommand))
			DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
		})

		// exec runs a command on the scan task and returns the status that answers it.
		exec := func(ctx context.Context, typ string, args any) task.Status {
			GinkgoHelper()
			key := uuid.New().String()
			Expect(dt.Exec(ctx, task.Command{
				Task: t.Key,
				Type: typ,
				Key:  key,
				Args: toJSON(args),
			})).To(Succeed())
			var stat task.Status
			Eventually(func(g Gomega) {
				stat = taskStatus(ctx, t)
				g.Expect(stat.Details.Cmd).To(Equal(key))
			}).WithTimeout(10 * time.Second).Should(Succeed())
			return stat
		}

		Describe("Check", func() {
			It("Should mark a device that reaches Sift as connected",
				func(ctx SpecContext) {
					dev := createDevice(ctx, sift.Properties{
						URI:    server.Address(),
						APIKey: createAPIKey(ctx, apiKey),
					})
					Eventually(func() device.Status {
						return deviceStatus(ctx, dev)
					}).Should(SatisfyAll(
						HaveField("Variant", status.VariantSuccess),
						HaveField("Message", "Connected to Sift"),
					))
				})

			It("Should warn about a device with a wrong API key",
				func(ctx SpecContext) {
					dev := createDevice(ctx, sift.Properties{
						URI:    server.Address(),
						APIKey: createAPIKey(ctx, "wrong"),
					})
					Eventually(func() device.Status {
						return deviceStatus(ctx, dev)
					}).Should(SatisfyAll(
						HaveField("Variant", status.VariantWarning),
						HaveField("Message", "Failed to connect to Sift"),
						HaveField("Description", ContainSubstring("invalid API key")),
					))
				})
		})

		Describe("test_connection", func() {
			It("Should connect with an API key the grant names",
				func(ctx SpecContext) {
					key := createAPIKey(ctx, apiKey)
					g := MustSucceed(secretSvc.Grant(
						ctx,
						nil,
						rackSvc.EmbeddedKey.OntologyID(),
						key,
					))
					stat := exec(ctx, "test_connection", map[string]any{
						"connection": sift.Properties{
							URI:    server.Address(),
							APIKey: key,
						},
						"grant": g,
					})
					Expect(stat.Variant).To(Equal(status.VariantSuccess))
				})

			It("Should fail for an API key that Sift rejects", func(ctx SpecContext) {
				key := createAPIKey(ctx, "wrong")
				g := MustSucceed(secretSvc.Grant(
					ctx,
					nil,
					rackSvc.EmbeddedKey.OntologyID(),
					key,
				))
				stat := exec(ctx, "test_connection", map[string]any{
					"connection": sift.Properties{URI: server.Address(), APIKey: key},
					"grant":      g,
				})
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Message).To(ContainSubstring("invalid API key"))
			})

			It("Should fail for an API key that no grant or device allows",
				func(ctx SpecContext) {
					key := createAPIKey(ctx, apiKey)
					stat := exec(ctx, "test_connection", map[string]any{
						"connection": sift.Properties{
							URI:    server.Address(),
							APIKey: key,
						},
					})
					Expect(stat.Variant).To(Equal(status.VariantError))
					Expect(stat.Message).To(ContainSubstring("cannot read secret"))
				})

			It("Should fail for a connection with no address", func(ctx SpecContext) {
				stat := exec(ctx, "test_connection", map[string]any{
					"connection": sift.Properties{APIKey: createAPIKey(ctx, apiKey)},
				})
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Message).To(ContainSubstring("uri"))
			})
		})

		Describe("list_assets", func() {
			It("Should list the assets the device can see", func(ctx SpecContext) {
				dev := createDevice(ctx, sift.Properties{
					URI:    server.Address(),
					APIKey: createAPIKey(ctx, apiKey),
				})
				stat := exec(ctx, "list_assets", map[string]any{"device": dev.Key})
				Expect(stat.Variant).To(Equal(status.VariantSuccess))
				Expect(stat.Details.Data).To(HaveKeyWithValue(
					"assets",
					ConsistOf("pad", "stand"),
				))
			})

			It("Should fail for a device that does not exist", func(ctx SpecContext) {
				stat := exec(ctx, "list_assets", map[string]any{"device": "missing"})
				Expect(stat.Variant).To(Equal(status.VariantError))
			})
		})
	})
})
