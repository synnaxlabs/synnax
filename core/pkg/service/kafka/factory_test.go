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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/address"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/query"
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
			tasks := f.InitialTasks()
			Expect(tasks).To(HaveLen(1))
			Expect(tasks[0].Type).To(Equal(kafka.ScanTaskType))
			Expect(tasks[0].Name).To(Equal("Kafka Scanner"))
		})
	})

	Describe("ConfigureTask", func() {
		It("Should not handle other task types", func(ctx SpecContext) {
			Expect(f.ConfigureTask(ctx, newTask(ctx, "other", nil), driver.NoCommand)).
				Error().To(MatchError(driver.ErrTaskNotHandled))
		})

		DescribeTable("Should reject a scan task with a rate that is not positive",
			func(ctx SpecContext, rate float64) {
				t := newTask(ctx, kafka.ScanTaskType, map[string]any{"rate": rate})
				Expect(f.ConfigureTask(ctx, t, "cmd-1")).Error().
					To(MatchError(validate.ErrValidation))
			},
			Entry("negative", -1.0),
			Entry("an explicit zero", 0.0),
		)

		It("Should only log an invalid config when no command waits on it",
			func(ctx SpecContext) {
				t := newTask(ctx, kafka.WriteTaskType, kafka.WriteConfig{})
				Expect(f.ConfigureTask(ctx, t, driver.NoCommand)).Error().
					To(MatchError(validate.ErrValidation))
				var stat task.Status
				Expect(statusSvc.NewRetrieve[task.StatusDetails]().
					Where(status.MatchOwners[task.StatusDetails](t.ResourceID())).
					Entry(&stat).
					Exec(ctx, db)).To(MatchError(query.ErrNotFound))
			},
		)

		It("Should answer the command with an error status for an invalid config",
			func(ctx SpecContext) {
				t := newTask(ctx, kafka.WriteTaskType, kafka.WriteConfig{})
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

	// readConfig and writeConfig build a minimal task config on dev.
	readConfig := func(ctx context.Context, dev device.Key, autoStart bool) any {
		GinkgoHelper()
		_, data := createIndexed(ctx, telem.Float64T)
		return kafka.ReadConfig{
			AutoStart:   autoStart,
			Device:      dev,
			Topic:       uniqueTopic(),
			StartOffset: kafka.StartOffsetEarliest,
			Records: []kafka.ReadRecord{
				{
					Key: "r",
					Fields: []kafka.ReadField{
						readField(
							kafka.BaseReadField{
								Key:     "v",
								Channel: data.Key(),
								Pointer: "/v",
							},
							telem.Float64T,
						),
					},
				},
			},
		}
	}
	writeConfig := func(ctx context.Context, dev device.Key, autoStart bool) any {
		GinkgoHelper()
		_, data := createIndexed(ctx, telem.Float64T)
		return kafka.WriteConfig{
			AutoStart: autoStart,
			Device:    dev,
			Topic:     uniqueTopic(),
			Channels:  []kafka.WriteChannel{{Key: "c1", Channel: data.Key()}},
		}
	}
	type buildConfig = func(ctx context.Context, dev device.Key, autoStart bool) any

	DescribeTable("Should answer the command when an auto-started task fails to start",
		func(ctx SpecContext, typ string, build buildConfig) {
			dev := createDevice(ctx, kafka.Properties{Brokers: []string{"127.0.0.1:1"}})
			t := newTask(ctx, typ, build(ctx, dev.Key, true))
			Expect(f.ConfigureTask(ctx, t, "cmd-1")).Error().
				To(MatchError(ContainSubstring("connecting to cluster")))
			stat := taskStatus(ctx, t)
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Details.Running).To(BeFalse())
			Expect(stat.Details.Cmd).To(Equal("cmd-1"))
		},
		Entry("read", kafka.ReadTaskType, readConfig),
		Entry("write", kafka.WriteTaskType, writeConfig),
	)

	DescribeTable("Should answer a command that needs no work with the current state",
		func(ctx SpecContext, typ string, build buildConfig) {
			dev := createDevice(ctx, clusterProperties())
			t := newTask(ctx, typ, build(ctx, dev.Key, false))
			dt := MustSucceed(f.ConfigureTask(ctx, t, "cmd-1"))
			DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
			exec := func(cmd, key string) task.Status {
				GinkgoHelper()
				Expect(dt.Exec(ctx, task.Command{Task: t.Key, Type: cmd, Key: key})).
					To(Succeed())
				return taskStatus(ctx, t)
			}
			stat := exec("stop", "cmd-2")
			Expect(stat.Details.Cmd).To(Equal("cmd-2"))
			Expect(stat.Details.Running).To(BeFalse())
			Expect(exec("start", "cmd-3").Details.Running).To(BeTrue())
			stat = exec("start", "cmd-4")
			Expect(stat.Details.Cmd).To(Equal("cmd-4"))
			Expect(stat.Details.Running).To(BeTrue())
			Expect(stat.Variant).To(Equal(status.VariantSuccess))
		},
		Entry("read", kafka.ReadTaskType, readConfig),
		Entry("write", kafka.WriteTaskType, writeConfig),
	)

	DescribeTable("Should reject an unknown command",
		func(ctx SpecContext, typ string, build buildConfig) {
			dev := createDevice(ctx, clusterProperties())
			t := newTask(ctx, typ, build(ctx, dev.Key, false))
			dt := MustSucceed(f.ConfigureTask(ctx, t, "cmd-1"))
			DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
			Expect(dt.Exec(ctx, task.Command{Task: t.Key, Type: "bogus"})).
				To(MatchError(driver.ErrUnsupportedCommand))
		},
		Entry("read", kafka.ReadTaskType, readConfig),
		Entry("write", kafka.WriteTaskType, writeConfig),
	)

	DescribeTable("Should reject a device that is not a Kafka cluster",
		func(ctx SpecContext, typ string, build buildConfig) {
			dev := device.Device{
				Key:        uuid.New().String(),
				Rack:       rackSvc.EmbeddedKey,
				Location:   "elsewhere",
				Make:       "other",
				Model:      "box",
				Name:       "box-" + uuid.New().String()[:8],
				Configured: true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return deviceSvc.NewWriter(tx).Create(ctx, &dev)
			})).To(Succeed())
			t := newTask(ctx, typ, build(ctx, dev.Key, false))
			Expect(f.ConfigureTask(ctx, t, "cmd-1")).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("is not a Kafka cluster")),
			))
			Expect(taskStatus(ctx, t).Message).To(Equal("Failed to configure task"))
		},
		Entry("read", kafka.ReadTaskType, readConfig),
		Entry("write", kafka.WriteTaskType, writeConfig),
	)

	DescribeTable("Should reject a device whose properties do not parse",
		func(ctx SpecContext, typ string, build buildConfig) {
			dev := device.Device{
				Key:        uuid.New().String(),
				Rack:       rackSvc.EmbeddedKey,
				Location:   "kafka",
				Make:       kafka.Make,
				Model:      "cluster",
				Name:       "cluster-" + uuid.New().String()[:8],
				Configured: true,
				Properties: toJSON(map[string]any{"brokers": "localhost:9092"}),
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return deviceSvc.NewWriter(tx).Create(ctx, &dev)
			})).To(Succeed())
			t := newTask(ctx, typ, build(ctx, dev.Key, false))
			Expect(f.ConfigureTask(ctx, t, "cmd-1")).Error().
				To(MatchError(ContainSubstring("parsing properties")))
		},
		Entry("read", kafka.ReadTaskType, readConfig),
		Entry("write", kafka.WriteTaskType, writeConfig),
	)

	DescribeTable("Should reject an unknown device",
		func(ctx SpecContext, typ string, build buildConfig) {
			t := newTask(ctx, typ, build(ctx, uuid.New().String(), false))
			Expect(f.ConfigureTask(ctx, t, "cmd-1")).Error().
				To(MatchError(query.ErrNotFound))
		},
		Entry("read", kafka.ReadTaskType, readConfig),
		Entry("write", kafka.WriteTaskType, writeConfig),
	)

	Describe("Scan task", func() {
		var (
			dt driver.Task
			t  task.Task
		)
		BeforeEach(func(ctx SpecContext) {
			t = newTask(ctx, kafka.ScanTaskType, map[string]any{"rate": 2})
			dt = MustSucceed(f.ConfigureTask(ctx, t, driver.NoCommand))
			DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
		})

		// testConnection runs test_connection against props and returns the status
		// that answers it.
		testConnection := func(ctx SpecContext, props kafka.Properties) task.Status {
			GinkgoHelper()
			key := uuid.New().String()
			Expect(dt.Exec(ctx, task.Command{
				Task: t.Key,
				Type: "test_connection",
				Key:  key,
				Args: toJSON(map[string]any{"connection": props}),
			})).To(Succeed())
			var stat task.Status
			Eventually(func(g Gomega) {
				stat = taskStatus(ctx, t)
				g.Expect(stat.Details.Cmd).To(Equal(key))
			}).WithTimeout(15 * time.Second).Should(Succeed())
			return stat
		}
		expectSuccess := func(stat task.Status) {
			GinkgoHelper()
			Expect(stat.Variant).To(Equal(status.VariantSuccess))
			Expect(stat.Message).To(Equal("Command executed successfully"))
		}
		expectFailure := func(stat task.Status, message string) {
			GinkgoHelper()
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Message).To(Equal(message))
			Expect(stat.Description).ToNot(BeEmpty())
		}
		expectInvalid := func(stat task.Status, reason string) {
			GinkgoHelper()
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Message).To(ContainSubstring(reason))
			Expect(stat.Details.Data).To(BeEmpty())
		}

		It("Should connect to a live cluster", func(ctx SpecContext) {
			expectSuccess(testConnection(ctx, clusterProperties()))
		})

		It("Should stop at a broker that does not answer", func(ctx SpecContext) {
			addr := closedAddress()
			stat := testConnection(ctx, kafka.Properties{Brokers: []string{addr}})
			expectFailure(stat, "Failed to reach broker")
			Expect(stat.Description).To(HavePrefix("Could not reach " + addr))
		})

		It("Should dial the default port of a broker address with no port",
			func(ctx SpecContext) {
				stat := testConnection(
					ctx,
					kafka.Properties{Brokers: []string{"kafka.invalid"}},
				)
				expectFailure(stat, "Failed to reach broker")
				Expect(stat.Description).
					To(HavePrefix("Could not reach kafka.invalid:9092: "))
			},
		)

		It("Should connect through a seed broker after one that does not answer",
			func(ctx SpecContext) {
				brokers := append([]string{closedAddress()}, cluster.ListenAddrs()...)
				expectSuccess(testConnection(ctx, kafka.Properties{Brokers: brokers}))
			},
		)

		It("Should reject properties with no brokers", func(ctx SpecContext) {
			expectInvalid(
				testConnection(ctx, kafka.Properties{}),
				"brokers: must be non-empty",
			)
		})

		Describe("TLS", func() {
			var (
				brokers []string
				caPEM   string
			)
			BeforeEach(func() {
				certs := MustSucceed(
					cert.NewFactory(cert.FactoryConfig{FS: xfs.NewMem()}),
				)
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

			It(
				"Should connect to a broker signed by the given CA",
				func(ctx SpecContext) {
					expectSuccess(
						testConnection(ctx, kafka.Properties{
							Brokers:       brokers,
							Secure:        true,
							CaCertificate: caPEM,
						}),
					)
				},
			)

			It("Should reject a broker signed by a CA outside the system roots",
				func(ctx SpecContext) {
					expectFailure(
						testConnection(
							ctx,
							kafka.Properties{Brokers: brokers, Secure: true},
						),
						"Certificate not trusted",
					)
				},
			)

			It("Should reject the tls property that secure replaced", func(
				ctx SpecContext,
			) {
				key := uuid.New().String()
				Expect(dt.Exec(ctx, task.Command{
					Task: t.Key,
					Type: "test_connection",
					Key:  key,
					Args: toJSON(map[string]any{
						"connection": map[string]any{"brokers": brokers, "tls": true},
					}),
				})).To(Succeed())
				Eventually(func(g Gomega) {
					stat := taskStatus(ctx, t)
					g.Expect(stat.Details.Cmd).To(Equal(key))
					g.Expect(stat.Variant).To(Equal(status.VariantError))
					g.Expect(stat.Message).
						To(ContainSubstring("tls is now named secure"))
				}).WithTimeout(15 * time.Second).Should(Succeed())
			})

			It("Should reject a CA certificate without TLS", func(ctx SpecContext) {
				expectInvalid(testConnection(ctx, kafka.Properties{
					Brokers:       brokers,
					CaCertificate: caPEM,
				}), "requires TLS")
			})

			It(
				"Should fail the TLS stage of a broker without TLS",
				func(ctx SpecContext) {
					props := clusterProperties()
					props.Secure = true
					expectFailure(
						testConnection(ctx, props),
						"TLS handshake failed",
					)
				},
			)

			It("Should reject a CA certificate that is not PEM", func(ctx SpecContext) {
				expectInvalid(testConnection(ctx, kafka.Properties{
					Brokers:       brokers,
					Secure:        true,
					CaCertificate: "not a certificate",
				}), "PEM-encoded certificate")
			})
		})

		Describe("SASL", func() {
			var brokers []string
			BeforeEach(func() {
				saslCluster := MustSucceed(kfake.NewCluster(
					kfake.NumBrokers(1),
					kfake.EnableSASL(),
					kfake.Superuser("PLAIN", "plain-user", "plain-pass"),
					kfake.Superuser("SCRAM-SHA-256", "scram256-user", "scram256-pass"),
					kfake.Superuser("SCRAM-SHA-512", "scram512-user", "scram512-pass"),
				))
				DeferCleanup(saslCluster.Close)
				brokers = saslCluster.ListenAddrs()
			})

			DescribeTable(
				"Should authenticate with each mechanism",
				func(ctx SpecContext, variant kafka.SASLVariant) {
					expectSuccess(
						testConnection(ctx, kafka.Properties{
							Brokers: brokers,
							Sasl:    kafka.SASL{Variant: variant},
						}),
					)
				},
				Entry(
					"plain",
					kafka.PlainSASL{Username: "plain-user", Password: "plain-pass"},
				),
				Entry(
					"SCRAM-SHA-256",
					kafka.ScramSha256SASL{
						Username: "scram256-user",
						Password: "scram256-pass",
					},
				),
				Entry(
					"SCRAM-SHA-512",
					kafka.ScramSha512SASL{
						Username: "scram512-user",
						Password: "scram512-pass",
					},
				),
			)

			It("Should fail to connect without credentials", func(ctx SpecContext) {
				expectFailure(
					testConnection(ctx, kafka.Properties{
						Brokers: brokers,
						Sasl:    kafka.SASL{Variant: kafka.NoneSASL{}},
					}),
					"Failed to read cluster metadata",
				)
			})

			DescribeTable("Should fail to connect with a wrong password",
				func(ctx SpecContext, variant kafka.SASLVariant) {
					expectFailure(
						testConnection(ctx, kafka.Properties{
							Brokers: brokers,
							Sasl:    kafka.SASL{Variant: variant},
						}),
						"Authentication failed",
					)
				},
				Entry(
					"plain",
					kafka.PlainSASL{Username: "plain-user", Password: "wrong"},
				),
				// The broker closes the connection on a failed SCRAM exchange.
				Entry(
					"SCRAM-SHA-256",
					kafka.ScramSha256SASL{Username: "scram256-user", Password: "wrong"},
				),
			)
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
				// A new device starts as a warning, so the wait matches the message.
				Eventually(func(g Gomega) {
					g.Expect(deviceStatus(ctx, live).Message).
						To(Equal("Device connected"))
					g.Expect(deviceStatus(ctx, dead).Message).
						To(Equal("Failed to connect"))
				}).WithTimeout(15 * time.Second).Should(Succeed())
				Expect(deviceStatus(ctx, live).Variant).To(Equal(status.VariantSuccess))
				Expect(deviceStatus(ctx, dead).Variant).To(Equal(status.VariantWarning))
				Expect(deviceStatus(ctx, live).Details.Connected).To(BeTrue())
				Expect(deviceStatus(ctx, dead).Details.Connected).To(BeFalse())
			},
		)
	})
})
