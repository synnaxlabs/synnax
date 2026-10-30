// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka_test

import (
	"context"
	"crypto/tls"
	"encoding/pem"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/security/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/address"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/twmb/franz-go/pkg/kfake"
)

var _ = Describe("Factory", func() {
	var f driver.Factory
	BeforeEach(func() { f = newFactory() })

	Describe("NewFactory", func() {
		It("Should reject a config without a DB", func() {
			Expect(kafka.NewFactory(kafka.FactoryConfig{
				Device:  deviceSvc,
				Channel: channelSvc,
				Framer:  framerSvc,
				Status:  statusSvc,
			})).Error().To(MatchError(ContainSubstring("db: must be non-nil")))
		})
	})

	It("Should be named after the device make", func() {
		Expect(f.Name()).To(Equal(kafka.Make))
	})

	Describe("InitialTasks", func() {
		It("Should return the scan task", func(ctx SpecContext) {
			tasks := MustSucceed(f.InitialTasks(ctx, rackSvc.EmbeddedKey))
			Expect(tasks).To(HaveLen(1))
			Expect(tasks[0].Type).To(Equal(kafka.ScanTaskType))
			Expect(tasks[0].Name).To(Equal("Kafka Scanner"))
		})
	})

	Describe("ConfigureTask", func() {
		It("Should not handle other task types", func(ctx SpecContext) {
			Expect(f.ConfigureTask(ctx, newTask("other", nil), driver.NoCommand)).
				Error().To(MatchError(driver.ErrTaskNotHandled))
		})

		It("Should reject a scan task with a rate that is not positive",
			func(ctx SpecContext) {
				t := newTask(kafka.ScanTaskType, map[string]any{"rate": -1})
				Expect(f.ConfigureTask(ctx, t, "cmd-1")).Error().
					To(MatchError(validate.ErrValidation))
			},
		)

		It("Should answer the command with an error status for an invalid config",
			func(ctx SpecContext) {
				t := newTask(kafka.WriteTaskType, kafka.WriteConfig{})
				Expect(f.ConfigureTask(ctx, t, "cmd-1")).Error().
					To(MatchError(validate.ErrValidation))
				stat := taskStatus(ctx, t)
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Message).To(Equal("Failed to configure task"))
				Expect(stat.Description).To(ContainSubstring("topic"))
				Expect(stat.Details.Cmd).To(Equal("cmd-1"))
			},
		)
	})

	DescribeTable("Should answer the command when an auto-started task fails to start",
		func(
			ctx SpecContext,
			typ string,
			build func(ctx context.Context, dev device.Key) any,
		) {
			dev := createDevice(ctx, kafka.Properties{Brokers: []string{"127.0.0.1:1"}})
			t := newTask(typ, build(ctx, dev.Key))
			Expect(f.ConfigureTask(ctx, t, "cmd-1")).Error().
				To(MatchError(ContainSubstring("connecting to cluster")))
			stat := taskStatus(ctx, t)
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Details.Running).To(BeFalse())
			Expect(stat.Details.Cmd).To(Equal("cmd-1"))
		},
		Entry("read", kafka.ReadTaskType,
			func(ctx context.Context, dev device.Key) any {
				_, data := createIndexed(ctx, telem.Float64T)
				return kafka.ReadConfig{
					AutoStart:   true,
					Device:      dev,
					Topic:       "t",
					StartOffset: kafka.StartOffsetEarliest,
					Records: []kafka.ReadRecord{{
						Key: "r",
						Fields: []kafka.ReadField{{
							Key:      "v",
							Channel:  data.Key(),
							Pointer:  "/v",
							DataType: telem.Float64T,
						}},
					}},
				}
			}),
		Entry("write", kafka.WriteTaskType,
			func(ctx context.Context, dev device.Key) any {
				_, data := createIndexed(ctx, telem.Float64T)
				return kafka.WriteConfig{
					AutoStart: true,
					Device:    dev,
					Topic:     "t",
					Channels:  []kafka.WriteChannel{{Key: "c1", Channel: data.Key()}},
				}
			}),
	)

	Describe("Scan task", func() {
		var (
			dt driver.Task
			t  task.Task
		)
		BeforeEach(func(ctx SpecContext) {
			t = newTask(kafka.ScanTaskType, map[string]any{"rate": 20})
			dt = MustSucceed(f.ConfigureTask(ctx, t, driver.NoCommand))
			DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
		})

		It("Should answer test_connection with success for a live cluster",
			func(ctx SpecContext) {
				Expect(dt.Exec(ctx, task.Command{
					Task: t.Key,
					Type: "test_connection",
					Key:  "cmd-2",
					Args: toJSON(map[string]any{"connection": clusterProperties()}),
				})).To(Succeed())
				stat := taskStatus(ctx, t)
				Expect(stat.Variant).To(Equal(status.VariantSuccess))
				Expect(stat.Message).To(Equal("Connected"))
				Expect(stat.Details.Cmd).To(Equal("cmd-2"))
			},
		)

		It("Should answer test_connection with an error for an unreachable cluster",
			func(ctx SpecContext) {
				Expect(dt.Exec(ctx, task.Command{
					Task: t.Key,
					Type: "test_connection",
					Key:  "cmd-3",
					Args: toJSON(map[string]any{
						"connection": kafka.Properties{
							Brokers: []string{"127.0.0.1:1"},
						},
					}),
				})).To(Succeed())
				stat := taskStatus(ctx, t)
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Message).To(Equal("Failed to connect"))
				Expect(stat.Details.Cmd).To(Equal("cmd-3"))
			},
		)

		It("Should answer test_connection with an error for invalid properties",
			func(ctx SpecContext) {
				Expect(dt.Exec(ctx, task.Command{
					Task: t.Key,
					Type: "test_connection",
					Key:  "cmd-4",
					Args: toJSON(map[string]any{"connection": kafka.Properties{}}),
				})).To(Succeed())
				stat := taskStatus(ctx, t)
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Description).To(ContainSubstring("brokers"))
			},
		)

		Describe("TLS", func() {
			var (
				brokers []string
				caPEM   string
			)
			BeforeEach(func() {
				certs := MustSucceed(cert.NewFactory(cert.FactoryConfig{
					FS:      xfs.NewMem(),
					KeySize: testutil.SmallKeySize,
				}))
				Expect(certs.CreateCAPair()).To(Succeed())
				ca := MustSucceed(certs.Loader.LoadCAs())[0]
				caPEM = string(
					pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.Raw}),
				)
				leaf := MustSucceed(certs.SignNodeCert([]address.Address{"127.0.0.1"}))
				tlsCluster := MustSucceed(kfake.NewCluster(
					kfake.NumBrokers(1),
					kfake.TLS(&tls.Config{Certificates: []tls.Certificate{*leaf}}),
				))
				DeferCleanup(tlsCluster.Close)
				brokers = tlsCluster.ListenAddrs()
			})

			testConnection := func(ctx SpecContext, props kafka.Properties) task.Status {
				GinkgoHelper()
				Expect(dt.Exec(ctx, task.Command{
					Task: t.Key,
					Type: "test_connection",
					Key:  "cmd-tls",
					Args: toJSON(map[string]any{"connection": props}),
				})).To(Succeed())
				return taskStatus(ctx, t)
			}

			It(
				"Should connect to a broker signed by the given CA",
				func(ctx SpecContext) {
					stat := testConnection(ctx, kafka.Properties{
						Brokers:       brokers,
						TLS:           true,
						CaCertificate: caPEM,
					})
					Expect(stat.Variant).To(Equal(status.VariantSuccess))
				},
			)

			It("Should reject a broker signed by a CA outside the system roots",
				func(ctx SpecContext) {
					stat := testConnection(
						ctx,
						kafka.Properties{Brokers: brokers, TLS: true},
					)
					Expect(stat.Variant).To(Equal(status.VariantError))
					Expect(stat.Message).To(Equal("Failed to connect"))
				},
			)

			It("Should reject a CA certificate without TLS", func(ctx SpecContext) {
				stat := testConnection(ctx, kafka.Properties{
					Brokers:       brokers,
					CaCertificate: caPEM,
				})
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Description).To(ContainSubstring("requires TLS"))
			})

			It("Should reject a CA certificate that is not PEM", func(ctx SpecContext) {
				stat := testConnection(ctx, kafka.Properties{
					Brokers:       brokers,
					TLS:           true,
					CaCertificate: "not a certificate",
				})
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Description).To(ContainSubstring("PEM-encoded certificate"))
			})
		})

		It("Should reject unknown commands", func(ctx SpecContext) {
			Expect(dt.Exec(ctx, task.Command{Task: t.Key, Type: "bogus"})).
				To(MatchError(driver.ErrUnsupportedCommand))
		})

		It("Should answer stop and start", func(ctx SpecContext) {
			Expect(dt.Exec(ctx, task.Command{Task: t.Key, Type: "stop", Key: "cmd-5"})).
				To(Succeed())
			stat := taskStatus(ctx, t)
			Expect(stat.Details.Running).To(BeFalse())
			Expect(stat.Details.Cmd).To(Equal("cmd-5"))
			Expect(dt.Exec(ctx, task.Command{
				Task: t.Key, Type: "start", Key: "cmd-6",
			})).To(Succeed())
			stat = taskStatus(ctx, t)
			Expect(stat.Details.Running).To(BeTrue())
			Expect(stat.Details.Cmd).To(Equal("cmd-6"))
		})

		It("Should set the status of every Kafka device on the rack",
			func(ctx SpecContext) {
				live := createDevice(ctx, clusterProperties())
				dead := createDevice(ctx, kafka.Properties{
					Brokers: []string{"127.0.0.1:1"},
				})
				Eventually(func(g Gomega) {
					g.Expect(deviceStatus(ctx, live).Variant).
						To(Equal(status.VariantSuccess))
					g.Expect(deviceStatus(ctx, dead).Variant).
						To(Equal(status.VariantError))
				}).WithTimeout(15 * time.Second).Should(Succeed())
				Expect(deviceStatus(ctx, live).Message).To(Equal("Device connected"))
				Expect(deviceStatus(ctx, dead).Message).To(Equal("Failed to connect"))
			},
		)
	})
})
