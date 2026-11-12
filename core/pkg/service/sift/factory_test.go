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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/sift"
	"github.com/synnaxlabs/synnax/pkg/service/sift/fake"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

const apiKey = "sift-api-key"

var _ = Describe("Factory", func() {
	var f driver.Factory

	BeforeEach(func() {
		f = MustSucceed(sift.NewFactory(factoryConfig()))
	})

	Describe("NewFactory", func() {
		It("Should require the secret service", func() {
			Expect(sift.NewFactory(sift.FactoryConfig{
				DB:     db,
				Device: deviceSvc,
				Status: statusSvc,
			})).Error().To(MatchError(ContainSubstring("secret: must be non-nil")))
		})

		DescribeTable("Should reject an invalid config",
			func(mutate func(*sift.FactoryConfig), msg string) {
				cfg := factoryConfig()
				mutate(&cfg)
				Expect(
					sift.NewFactory(cfg),
				).Error().
					To(MatchError(ContainSubstring(msg)))
			},
			Entry("no label service",
				func(c *sift.FactoryConfig) { c.Label = nil },
				"label: must be non-nil",
			),
			Entry("a checkpoint interval that is not positive",
				func(c *sift.FactoryConfig) { c.CheckpointInterval = -telem.Second },
				"checkpoint_interval: must be positive",
			),
		)
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
				To(MatchError(ContainSubstring("rate")))
		})

		It("Should run one scan task for each rack", func(ctx SpecContext) {
			cfg := msgpack.EncodedJSON{"rate": 50}
			first := newTask(ctx, sift.ScanTaskType, cfg)
			second := newTask(ctx, sift.ScanTaskType, cfg)
			dt := MustSucceed(f.ConfigureTask(ctx, first, driver.NoCommand))
			Expect(f.ConfigureTask(ctx, second, driver.NoCommand)).Error().To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("the rack already has a Sift scan task")),
			))
			By("Reconfiguring the scan task that runs")
			Expect(dt.Stop(false)).To(Succeed())
			dt = MustSucceed(f.ConfigureTask(ctx, first, driver.NoCommand))
			Expect(dt.Stop(false)).To(Succeed())
			By("Running the other scan task once the first stops")
			dt = MustSucceed(f.ConfigureTask(ctx, second, driver.NoCommand))
			Expect(dt.Stop(false)).To(Succeed())
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
			t = newTask(ctx, sift.ScanTaskType, msgpack.EncodedJSON{"rate": 50})
			dt = MustSucceed(f.ConfigureTask(ctx, t, driver.NoCommand))
			DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
		})

		exec := func(ctx context.Context, typ string, args any) task.Status {
			GinkgoHelper()
			return execCommand(ctx, dt, t, typ, args)
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
						HaveField(
							"Description",
							ContainSubstring("rejected the API key"),
						),
					))
				})
		})

		It("Should reject a command it does not know", func(ctx SpecContext) {
			stat := exec(ctx, "restart", map[string]any{})
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Message).To(Equal("Failed to execute command"))
			Expect(stat.Description).To(Equal(
				"The task does not accept the 'restart' command.",
			))
		})

		Describe("test_connection", func() {
			It("Should connect with an API key the grant names",
				func(ctx SpecContext) {
					key := createAPIKey(ctx, apiKey)
					g := grantToRack(ctx, key)
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
				g := grantToRack(ctx, key)
				stat := exec(ctx, "test_connection", map[string]any{
					"connection": sift.Properties{URI: server.Address(), APIKey: key},
					"grant":      g,
				})
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Message).To(Equal("Authentication failed"))
				Expect(stat.Description).To(ContainSubstring("rejected the API key"))
			})

			DescribeTable("Should connect through a URI copied from a browser",
				func(ctx SpecContext, format string) {
					key := createAPIKey(ctx, apiKey)
					stat := exec(ctx, "test_connection", map[string]any{
						"connection": sift.Properties{
							URI:    fmt.Sprintf(format, server.Address()),
							APIKey: key,
						},
						"grant": grantToRack(ctx, key),
					})
					Expect(stat.Variant).To(Equal(status.VariantSuccess))
				},
				Entry("https", "https://%s"),
				Entry("http with a trailing slash", "http://%s/"),
				Entry("an upper case scheme", "HTTPS://%s"),
			)

			It("Should say that a closed port refused the connection",
				func(ctx SpecContext) {
					key := createAPIKey(ctx, apiKey)
					stat := exec(ctx, "test_connection", map[string]any{
						"connection": sift.Properties{URI: "127.0.0.1:1", APIKey: key},
						"grant":      grantToRack(ctx, key),
					})
					Expect(stat.Variant).To(Equal(status.VariantError))
					Expect(stat.Message).To(Equal("Failed to reach Sift"))
					Expect(stat.Description).To(Equal(
						"Could not reach 127.0.0.1:1: the connection was refused. Check " +
							"the URI, and that the network of the Core can reach Sift.",
					))
				})

			It("Should fail without a grant, even for the key of a saved device",
				func(ctx SpecContext) {
					key := createAPIKey(ctx, apiKey)
					createDevice(
						ctx,
						sift.Properties{URI: server.Address(), APIKey: key},
					)
					stat := exec(ctx, "test_connection", map[string]any{
						"connection": sift.Properties{
							URI:    server.Address(),
							APIKey: key,
						},
					})
					Expect(stat.Variant).To(Equal(status.VariantError))
					Expect(
						stat.Message,
					).To(ContainSubstring("has ended or does not exist"))
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
			It("Should list the assets of every page", func(ctx SpecContext) {
				// Sift returns at most 1000 assets in a page.
				names := make([]string, 1001)
				for i := range names {
					names[i] = fmt.Sprintf("asset-%04d", i)
				}
				paged := MustOpen(fake.Open(fake.Config{APIKey: apiKey, Assets: names}))
				dev := createDevice(ctx, sift.Properties{
					URI:    paged.Address(),
					APIKey: createAPIKey(ctx, apiKey),
				})
				stat := exec(ctx, "list_assets", map[string]any{"device": dev.Key})
				Expect(stat.Variant).To(Equal(status.VariantSuccess))
				Expect(stat.Details.Data).To(HaveKeyWithValue("assets", SatisfyAll(
					HaveLen(len(names)),
					ContainElement(names[len(names)-1]),
				)))
			})

			It("Should list the assets the device can see", func(ctx SpecContext) {
				dev := createDevice(ctx, sift.Properties{
					URI:    server.Address(),
					APIKey: createAPIKey(ctx, apiKey),
				})
				stat := exec(ctx, "list_assets", map[string]any{"device": dev.Key})
				Expect(stat.Variant).To(Equal(status.VariantSuccess))
				Expect(stat.Details.Data).To(SatisfyAll(
					HaveKeyWithValue("assets", ConsistOf("pad", "stand")),
					HaveKeyWithValue("last", ""),
				))
			})

			It("Should return the asset of the newest upload from the device",
				func(ctx SpecContext) {
					dev := createDevice(ctx, sift.Properties{
						URI:    server.Address(),
						APIKey: createAPIKey(ctx, apiKey),
					})
					record := func(device, asset string) {
						GinkgoHelper()
						s := sift.UploadStatus{
							Key:     "sift_upload:" + uuid.NewV7().String(),
							Name:    asset,
							Variant: status.VariantSuccess,
							Message: "Uploaded to Sift",
							Time:    telem.Now(),
							Details: sift.UploadDetails{Device: device, Asset: asset},
						}
						Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
							return statusSvc.NewWriter(tx).Set(ctx, &s)
						})).To(Succeed())
					}
					record(dev.Key, "stand")
					record(dev.Key, "pad")
					record("other", "rig")
					stat := exec(ctx, "list_assets", map[string]any{"device": dev.Key})
					Expect(stat.Details.Data).To(HaveKeyWithValue("last", "pad"))
				})

			It("Should fail for a device that does not exist", func(ctx SpecContext) {
				stat := exec(ctx, "list_assets", map[string]any{"device": "missing"})
				Expect(stat.Variant).To(Equal(status.VariantError))
			})
		})

		Describe("list_uploads", func() {
			record := func(ctx SpecContext, rng *ranger.Key) string {
				GinkgoHelper()
				s := sift.UploadStatus{
					Key:     "sift_upload:" + uuid.NewV7().String(),
					Name:    "upload",
					Variant: status.VariantSuccess,
					Message: "Uploaded to Sift",
					Time:    telem.Now(),
					Details: sift.UploadDetails{Range: rng, Asset: "stand"},
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return statusSvc.NewWriter(tx).Set(ctx, &s)
				})).To(Succeed())
				return s.Key
			}

			It(
				"Should list the uploads of a range, newest first",
				func(ctx SpecContext) {
					rng, other := uuid.NewV7(), uuid.NewV7()
					first := record(ctx, &rng)
					record(ctx, &other)
					second := record(ctx, &rng)
					stat := exec(ctx, "list_uploads", map[string]any{"range": rng})
					Expect(stat.Variant).To(Equal(status.VariantSuccess))
					Expect(stat.Details.Data).To(
						HaveKeyWithValue("uploads", Equal([]any{second, first})),
					)
				},
			)

			It("Should list no uploads for a range with none", func(ctx SpecContext) {
				stat := exec(ctx, "list_uploads", map[string]any{"range": uuid.NewV7()})
				Expect(stat.Details.Data).To(HaveKeyWithValue("uploads", BeEmpty()))
			})

			It("Should reject a command without a range", func(ctx SpecContext) {
				record(ctx, nil)
				stat := exec(ctx, "list_uploads", map[string]any{})
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Message).To(Equal("range: required"))
			})
		})
	})
})
