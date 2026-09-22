// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mqtt_test

import (
	"context"
	"crypto/tls"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// createCerts writes a CA and a certificate for the local host to a temporary
// directory. It returns the TLS configuration of a broker and the path of the CA.
func createCerts() (*tls.Config, string) {
	GinkgoHelper()
	dir := GinkgoT().TempDir()
	f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
		CertsDir: dir,
		Hosts:    []address.Address{"127.0.0.1"},
	}))
	Expect(f.CreateCAPair()).To(Succeed())
	Expect(f.CreateNodePair()).To(Succeed())
	pair := MustSucceed(tls.LoadX509KeyPair(
		filepath.Join(dir, "node.crt"), filepath.Join(dir, "node.key"),
	))
	return &tls.Config{Certificates: []tls.Certificate{pair}},
		filepath.Join(dir, "ca.crt")
}

var _ = Describe("Scan task", func() {
	var (
		rackKey rack.Key
		factory driver.Factory
	)

	BeforeEach(func(ctx SpecContext) {
		rackKey = createRack(ctx)
		factory = newFactory(64)
	})

	// scan configures a scan task that checks its devices 50 times each second.
	scan := func(ctx context.Context) (driver.Task, task.Task) {
		GinkgoHelper()
		t := newTask(rackKey, mqtt.ScanTaskType, msgpack.EncodedJSON{"rate": 50})
		return configure(ctx, factory, t), t
	}

	Describe("InitialTasks", func() {
		It("Should ask for one scan task", func() {
			initial := factory.InitialTasks()
			Expect(initial).To(HaveLen(1))
			Expect(initial[0].Name).To(Equal("MQTT Scanner"))
			Expect(initial[0].Type).To(Equal(mqtt.ScanTaskType))
		})
	})

	Describe("ConfigureTask", func() {
		It("Should not handle a task of another integration", func(ctx SpecContext) {
			t := newTask(rackKey, "opc_read", msgpack.EncodedJSON{})
			Expect(factory.ConfigureTask(ctx, t, "configure")).Error().
				To(MatchError(driver.ErrTaskNotHandled))
		})

		It("Should start a scan task on its own", func(ctx SpecContext) {
			_, t := scan(ctx)
			stat := taskStatus(ctx, t)
			Expect(stat.Variant).To(Equal(status.VariantSuccess))
			Expect(stat.Details.Running).To(BeTrue())
		})

		It("Should not start a disabled scan task", func(ctx SpecContext) {
			t := newTask(
				rackKey, mqtt.ScanTaskType, msgpack.EncodedJSON{"disabled": true},
			)
			configure(ctx, factory, t)
			Expect(taskStatus(ctx, t).Details.Running).To(BeFalse())
		})
	})

	Describe("Device status", func() {
		It("Should follow the broker as it stops and returns", func(ctx SpecContext) {
			broker := startBroker(0)
			dev := createBrokerDevice(ctx, rackKey, broker.port, nil)
			scan(ctx)
			Eventually(func(g Gomega) {
				stat := deviceStatus(ctx, dev)
				g.Expect(stat.Variant).To(Equal(status.VariantSuccess))
				g.Expect(stat.Message).To(Equal("Broker connected"))
				g.Expect(stat.Details.Rack).To(Equal(rackKey))
			}).Should(Succeed())
			port := broker.port
			broker.stop()
			Eventually(func(g Gomega) {
				stat := deviceStatus(ctx, dev)
				g.Expect(stat.Variant).To(Equal(status.VariantWarning))
				g.Expect(stat.Message).To(Equal("Failed to reach broker"))
				g.Expect(stat.Description).To(ContainSubstring("connection refused"))
			}).Should(Succeed())
			startBroker(port)
			Eventually(func() status.Variant { return deviceStatus(ctx, dev).Variant }).
				Should(Equal(status.VariantSuccess))
		})

		It("Should report properties that are not valid", func(ctx SpecContext) {
			dev := createBrokerDevice(ctx, rackKey, 70000, nil)
			scan(ctx)
			Eventually(func(g Gomega) {
				stat := deviceStatus(ctx, dev)
				g.Expect(stat.Variant).To(Equal(status.VariantWarning))
				g.Expect(stat.Description).To(ContainSubstring(
					"invalid broker properties: port: must be 0 to 65535",
				))
			}).Should(Succeed())
		})

		It("Should report the connection of a running task in place of its own",
			func(ctx SpecContext) {
				broker := startBroker(0)
				dev := createBrokerDevice(ctx, rackKey, broker.port, nil)
				_, data := createIndexed(ctx, telem.Float64T)
				read := configure(ctx, factory, newTask(
					rackKey, mqtt.ReadTaskType,
					readConfig(dev, plainEntry("plant/scan", field("v", data[0], ""))),
				))
				Expect(read.Exec(ctx, task.Command{Type: "start", Key: "start"})).
					To(Succeed())
				Eventually(broker.clientCount).Should(Equal(1))
				scan(ctx)
				Eventually(func() status.Variant {
					return deviceStatus(ctx, dev).Variant
				}).Should(Equal(status.VariantSuccess))
				Consistently(broker.clientCount, 200*time.Millisecond).Should(Equal(1))
			},
		)
	})

	Describe("Test connection", func() {
		// tested is the scan task of the last call to testConnection.
		var tested task.Task

		testConnection := func(
			ctx context.Context,
			port int,
			properties msgpack.EncodedJSON,
		) error {
			GinkgoHelper()
			if properties == nil {
				properties = msgpack.EncodedJSON{}
			}
			properties["port"] = port
			var configured driver.Task
			configured, tested = scan(ctx)
			return configured.Exec(ctx, task.Command{
				Type: "test_connection",
				Key:  "test",
				Args: msgpack.EncodedJSON{
					"location":   "127.0.0.1",
					"properties": properties,
				},
			})
		}

		It("Should connect to a broker that is not a device yet",
			func(ctx SpecContext) {
				broker := startBroker(0)
				Expect(testConnection(ctx, broker.port, nil)).To(Succeed())
				stat := taskStatus(ctx, tested)
				Expect(stat.Variant).To(Equal(status.VariantSuccess))
				Expect(stat.Details.Cmd).To(Equal("test"))
				Eventually(broker.clientCount).Should(BeZero())
			},
		)

		It("Should report a broker it cannot reach", func(ctx SpecContext) {
			broker := startBroker(0)
			port := broker.port
			broker.stop()
			Expect(testConnection(ctx, port, nil)).
				To(MatchError(ContainSubstring("connection refused")))
			stat := taskStatus(ctx, tested)
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Message).To(ContainSubstring("connection refused"))
			Expect(stat.Details.Cmd).To(Equal("test"))
		})

		It("Should send the username and the password", func(ctx SpecContext) {
			broker := startBrokerWith(brokerOptions{
				username: "operator", password: "valve-7",
			})
			Expect(testConnection(ctx, broker.port, msgpack.EncodedJSON{
				"username": "operator", "password": "valve-7",
			})).To(Succeed())
			Expect(testConnection(ctx, broker.port, msgpack.EncodedJSON{
				"username": "operator", "password": "wrong",
			})).To(MatchError(ContainSubstring(
				"the broker rejected the username or password",
			)))
		})

		Describe("TLS", func() {
			It("Should verify the broker against a CA file", func(ctx SpecContext) {
				tlsCfg, caFile := createCerts()
				broker := startBrokerWith(brokerOptions{tls: tlsCfg})
				Expect(testConnection(ctx, broker.port, msgpack.EncodedJSON{
					"secure": true, "ca_file": caFile,
				})).To(Succeed())
			})

			It("Should reject a broker that its roots do not verify",
				func(ctx SpecContext) {
					tlsCfg, _ := createCerts()
					broker := startBrokerWith(brokerOptions{tls: tlsCfg})
					Expect(testConnection(ctx, broker.port, msgpack.EncodedJSON{
						"secure": true,
					})).To(MatchError(ContainSubstring(
						"failed to verify certificate",
					)))
				},
			)

			It("Should accept any certificate when verification is skipped",
				func(ctx SpecContext) {
					tlsCfg, _ := createCerts()
					broker := startBrokerWith(brokerOptions{tls: tlsCfg})
					Expect(testConnection(ctx, broker.port, msgpack.EncodedJSON{
						"secure": true, "verification_skipped": true,
					})).To(Succeed())
				},
			)

			It("Should report a CA file that does not exist", func(ctx SpecContext) {
				Expect(testConnection(ctx, 8883, msgpack.EncodedJSON{
					"secure": true, "ca_file": "/no/such/ca.crt",
				})).To(MatchError(ContainSubstring("failed to read the CA file")))
			})
		})
	})

	Describe("Browse", func() {
		var (
			broker *testBroker
			dev    device.Device
		)

		BeforeEach(func(ctx SpecContext) {
			broker = startBroker(0)
			dev = createBrokerDevice(ctx, rackKey, broker.port, nil)
		})

		browse := func(ctx context.Context, args msgpack.EncodedJSON) task.Status {
			GinkgoHelper()
			args["device"] = dev.Key
			configured, t := scan(ctx)
			Expect(configured.Exec(ctx, task.Command{
				Type: "browse", Key: "browse", Args: args,
			})).To(Succeed())
			stat := taskStatus(ctx, t)
			Expect(stat.Variant).To(Equal(status.VariantSuccess))
			Expect(stat.Details.Cmd).To(Equal("browse"))
			return stat
		}

		It("Should list the retained topics with their payloads, in order",
			func(ctx SpecContext) {
				broker.publish("plant/b", `{"v":2}`, true)
				broker.publish("plant/a", `{"v":1}`, true)
				broker.publish("other/c", "3", true)
				stat := browse(ctx, msgpack.EncodedJSON{
					"filter": "plant/#", "duration": 200,
				})
				Expect(stat.Details.Data).To(HaveKeyWithValue("truncated", false))
				Expect(stat.Details.Data["topics"]).To(Equal([]any{
					map[string]any{
						"topic": "plant/a", "payload": `{"v":1}`, "retained": true,
					},
					map[string]any{
						"topic": "plant/b", "payload": `{"v":2}`, "retained": true,
					},
				}))
			},
		)

		It("Should report a list cut at the limit", func(ctx SpecContext) {
			broker.publish("plant/a", "1", true)
			broker.publish("plant/b", "2", true)
			broker.publish("plant/c", "3", true)
			stat := browse(ctx, msgpack.EncodedJSON{"duration": 200, "limit": 2})
			Expect(stat.Details.Data).To(HaveKeyWithValue("truncated", true))
			Expect(stat.Details.Data["topics"]).To(HaveLen(2))
		})

		It("Should cut a long payload and leave out one that is not text",
			func(ctx SpecContext) {
				broker.publish("plant/long", strings.Repeat("é", 200), true)
				broker.publish("plant/binary", "\xff\xfe\x00", true)
				stat := browse(ctx, msgpack.EncodedJSON{"duration": 200})
				topics := stat.Details.Data["topics"].([]any)
				Expect(topics).To(HaveLen(2))
				Expect(topics[0]).To(HaveKeyWithValue("payload", ""))
				Expect(topics[1]).
					To(HaveKeyWithValue("payload", strings.Repeat("é", 128)))
			},
		)

		It("Should leave the broker when the browse ends", func(ctx SpecContext) {
			browse(ctx, msgpack.EncodedJSON{"duration": 50})
			Eventually(broker.clientCount).Should(BeZero())
		})

		It("Should report a device that does not exist", func(ctx SpecContext) {
			configured, t := scan(ctx)
			Expect(configured.Exec(ctx, task.Command{
				Type: "browse", Key: "browse",
				Args: msgpack.EncodedJSON{"device": "missing"},
			})).To(MatchError(query.ErrNotFound))
			Expect(taskStatus(ctx, t).Variant).To(Equal(status.VariantError))
		})
	})

	Describe("Exec", func() {
		It("Should reject a command it does not support", func(ctx SpecContext) {
			configured, _ := scan(ctx)
			Expect(configured.Exec(ctx, task.Command{Type: "reboot", Key: "x"})).
				To(MatchError(driver.ErrUnsupportedCommand))
		})
	})
})
