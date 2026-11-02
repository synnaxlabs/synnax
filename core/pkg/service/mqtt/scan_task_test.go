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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/encoding/msgpack"
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

	// configureRead configures a read task on dev, which opens the shared connection
	// when it starts.
	configureRead := func(ctx context.Context, dev device.Device) driver.Task {
		GinkgoHelper()
		ch := createVirtual(ctx, telem.Float64T)
		return configure(ctx, factory, newTask(
			rackKey, mqtt.ReadTaskType,
			readConfig(dev, plainEntry("plant/edit", field("v", ch, ""))),
		))
	}
	startRead := func(ctx context.Context, read driver.Task) {
		GinkgoHelper()
		Expect(read.Exec(ctx, task.Command{Type: "start", Key: "start"})).
			To(Succeed())
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
				g.Expect(stat.Description).To(ContainSubstring("refused"))
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
				g.Expect(stat.Description).To(Equal(
					"invalid broker properties: port: must be 1 to 65535: " +
						"validation error",
				))
			}).Should(Succeed())
		})

		It("Should probe a broker with a client ID of its own", func(ctx SpecContext) {
			broker := startBroker(0)
			dev := createBrokerDevice(ctx, rackKey, broker.port, msgpack.EncodedJSON{
				"client_id": "line-7",
			})
			scan(ctx)
			Eventually(func() status.Variant {
				return deviceStatus(ctx, dev).Variant
			}).Should(Equal(status.VariantSuccess))
			Expect(broker.connectedIDs()).To(SatisfyAll(
				Not(BeEmpty()), Not(ContainElement("line-7")),
			))
		})

		It("Should not read the TLS files again while the properties stay the same",
			func(ctx SpecContext) {
				tlsCfg, caFile := createCerts()
				broker := startBrokerWith(brokerOptions{tls: tlsCfg})
				properties := msgpack.EncodedJSON{"secure": true, "ca_file": caFile}
				dev := createBrokerDevice(ctx, rackKey, broker.port, properties)
				_, data := createIndexed(ctx, telem.Float64T)
				read := configure(ctx, factory, newTask(
					rackKey, mqtt.ReadTaskType,
					readConfig(dev, plainEntry("plant/tls", field("v", data[0], ""))),
				))
				Expect(read.Exec(ctx, task.Command{Type: "start", Key: "start"})).
					To(Succeed())
				Eventually(broker.clientCount).Should(Equal(1))
				scan(ctx)
				variant := func() status.Variant {
					return deviceStatus(ctx, dev).Variant
				}
				Eventually(variant).Should(Equal(status.VariantSuccess))
				Expect(os.Remove(caFile)).To(Succeed())
				Consistently(variant, 300*time.Millisecond).
					Should(Equal(status.VariantSuccess))
			},
		)

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

	Describe("Device edits", func() {
		It("Should reconnect a running task with the new properties, with no scan task",
			func(ctx SpecContext) {
				broker := startBroker(0)
				dev := createBrokerDevice(ctx, rackKey, broker.port, nil)
				startRead(ctx, configureRead(ctx, dev))
				Eventually(broker.clientCount).Should(Equal(1))
				editDevice(ctx, dev, msgpack.EncodedJSON{
					"port": broker.port, "client_id": "line-9",
				})
				Eventually(broker.connectedIDs).Should(ContainElement("line-9"))
				Eventually(broker.clientCount).Should(Equal(1))
			},
		)

		It("Should start a task configured before an edit with the new properties",
			func(ctx SpecContext) {
				broker := startBroker(0)
				dev := createBrokerDevice(ctx, rackKey, broker.port, nil)
				read := configureRead(ctx, dev)
				editDevice(ctx, dev, msgpack.EncodedJSON{
					"port": broker.port, "client_id": "line-9",
				})
				startRead(ctx, read)
				Eventually(broker.connectedIDs).Should(Equal([]string{"line-9"}))
				Consistently(broker.connectedIDs, 300*time.Millisecond).
					Should(Equal([]string{"line-9"}))
			},
		)

		It("Should answer a start with the error of the current properties",
			func(ctx SpecContext) {
				broker := startBroker(0)
				dev := createBrokerDevice(ctx, rackKey, broker.port, nil)
				read := configureRead(ctx, dev)
				editDevice(ctx, dev, msgpack.EncodedJSON{"port": 0})
				Expect(read.Exec(ctx, task.Command{Type: "start", Key: "start"})).
					To(MatchError(
						"invalid broker properties: port: must be 1 to 65535: " +
							"validation error",
					))
				editDevice(ctx, dev, msgpack.EncodedJSON{"port": broker.port})
				startRead(ctx, read)
				Eventually(broker.clientCount).Should(Equal(1))
			},
		)

		It("Should keep the new properties when a task configured before the edit "+
			"joins the connection",
			func(ctx SpecContext) {
				broker := startBroker(0)
				dev := createBrokerDevice(ctx, rackKey, broker.port, nil)
				first, second := configureRead(ctx, dev), configureRead(ctx, dev)
				startRead(ctx, first)
				Eventually(broker.clientCount).Should(Equal(1))
				editDevice(ctx, dev, msgpack.EncodedJSON{
					"port": broker.port, "client_id": "line-9",
				})
				Eventually(broker.connectedIDs).Should(ContainElement("line-9"))
				ids := broker.connectedIDs()
				startRead(ctx, second)
				Consistently(broker.connectedIDs, 300*time.Millisecond).
					Should(Equal(ids))
			},
		)

		It("Should report an edit that the connection cannot take, and keep it",
			func(ctx SpecContext) {
				broker := startBroker(0)
				dev := createBrokerDevice(ctx, rackKey, broker.port, nil)
				startRead(ctx, configureRead(ctx, dev))
				Eventually(broker.clientCount).Should(Equal(1))
				scan(ctx)
				editDevice(ctx, dev, msgpack.EncodedJSON{
					"port": broker.port, "cert_file": "/certs/client.crt",
				})
				Eventually(func(g Gomega) {
					stat := deviceStatus(ctx, dev)
					g.Expect(stat.Variant).To(Equal(status.VariantWarning))
					g.Expect(stat.Description).To(Equal(
						"invalid broker properties: key_file: cert_file and key_file " +
							"must be set together: validation error",
					))
				}).Should(Succeed())
				Expect(broker.connectedIDs()).To(HaveLen(1))
				editDevice(ctx, dev, msgpack.EncodedJSON{"port": broker.port})
				Eventually(func() status.Variant {
					return deviceStatus(ctx, dev).Variant
				}).Should(Equal(status.VariantSuccess))
			},
		)
	})

	Describe("Test connection", func() {
		// test runs a connection test with args on a new scan task and returns the
		// status that answers it.
		test := func(ctx context.Context, args msgpack.EncodedJSON) task.Status {
			GinkgoHelper()
			configured, t := scan(ctx)
			args["location"] = "127.0.0.1"
			return answer(ctx, configured, t, task.Command{
				Type: "test_connection",
				Key:  "test",
				Args: args,
			})
		}
		testConnection := func(
			ctx context.Context,
			properties msgpack.EncodedJSON,
		) task.Status {
			GinkgoHelper()
			return test(ctx, msgpack.EncodedJSON{"properties": properties})
		}
		expectSuccess := func(stat task.Status) {
			GinkgoHelper()
			Expect(stat.Variant).To(Equal(status.VariantSuccess))
			Expect(stat.Message).To(Equal("Command executed successfully"))
		}
		expectError := func(stat task.Status, message types.GomegaMatcher) {
			GinkgoHelper()
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Message).To(message)
		}

		It("Should connect to a broker that is not a device yet",
			func(ctx SpecContext) {
				broker := startBroker(0)
				expectSuccess(testConnection(ctx, msgpack.EncodedJSON{
					"port": broker.port,
				}))
				Eventually(broker.clientCount).Should(BeZero())
			},
		)

		It("Should report the shared connection of a device with the same settings",
			func(ctx SpecContext) {
				broker := startBroker(0)
				properties := msgpack.EncodedJSON{
					"port": broker.port, "client_id": "line-9",
				}
				dev := createBrokerDevice(ctx, rackKey, 0, properties)
				startRead(ctx, configureRead(ctx, dev))
				Eventually(broker.clientCount).Should(Equal(1))
				expectSuccess(test(ctx, msgpack.EncodedJSON{
					"device": dev.Key, "properties": properties,
				}))
				Expect(broker.connectedIDs()).To(Equal([]string{"line-9"}))
			},
		)

		It("Should report a broker it cannot reach", func(ctx SpecContext) {
			broker := startBroker(0)
			port := broker.port
			broker.stop()
			expectError(
				testConnection(ctx, msgpack.EncodedJSON{"port": port}),
				ContainSubstring("refused"),
			)
		})

		It("Should send the username and the password", func(ctx SpecContext) {
			broker := startBrokerWith(brokerOptions{
				username: "operator", password: "valve-7",
			})
			expectSuccess(testConnection(ctx, msgpack.EncodedJSON{
				"port": broker.port, "username": "operator", "password": "valve-7",
			}))
			expectError(
				testConnection(ctx, msgpack.EncodedJSON{
					"port": broker.port, "username": "operator", "password": "wrong",
				}),
				Equal("the broker rejected the username or password: not Authorized"),
			)
		})

		Describe("Port and keep-alive", func() {
			DescribeTable("Should connect to the default port of a broker",
				func(ctx SpecContext, properties msgpack.EncodedJSON, address string) {
					expectError(
						testConnection(ctx, properties),
						ContainSubstring(address),
					)
				},
				Entry("a plain broker", msgpack.EncodedJSON{}, "127.0.0.1:1883"),
				Entry("a secure broker",
					msgpack.EncodedJSON{"secure": true}, "127.0.0.1:8883",
				),
			)

			DescribeTable("Should send the keep-alive of the properties",
				func(ctx SpecContext, properties msgpack.EncodedJSON, seconds uint16) {
					broker := startBroker(0)
					properties["port"] = broker.port
					expectSuccess(testConnection(ctx, properties))
					Expect(broker.connectedKeepAlives()).To(Equal([]uint16{seconds}))
				},
				Entry("no keep-alive", msgpack.EncodedJSON{}, uint16(30)),
				Entry("a keep-alive of 7 seconds",
					msgpack.EncodedJSON{"keep_alive": 7}, uint16(7),
				),
				Entry("the longest keep-alive",
					msgpack.EncodedJSON{"keep_alive": 65535}, uint16(65535),
				),
			)
		})

		Describe("TLS", func() {
			It("Should verify the broker against a CA file", func(ctx SpecContext) {
				tlsCfg, caFile := createCerts()
				broker := startBrokerWith(brokerOptions{tls: tlsCfg})
				expectSuccess(testConnection(ctx, msgpack.EncodedJSON{
					"port": broker.port, "secure": true, "ca_file": caFile,
				}))
			})

			It("Should reject a broker that its roots do not verify",
				func(ctx SpecContext) {
					tlsCfg, _ := createCerts()
					broker := startBrokerWith(brokerOptions{tls: tlsCfg})
					expectError(
						testConnection(ctx, msgpack.EncodedJSON{
							"port": broker.port, "secure": true,
						}),
						ContainSubstring("failed to verify certificate"),
					)
				},
			)

			It("Should accept any certificate when verification is skipped",
				func(ctx SpecContext) {
					tlsCfg, _ := createCerts()
					broker := startBrokerWith(brokerOptions{tls: tlsCfg})
					expectSuccess(testConnection(ctx, msgpack.EncodedJSON{
						"port":                 broker.port,
						"secure":               true,
						"verification_skipped": true,
					}))
				},
			)

			It("Should report a CA file that does not exist", func(ctx SpecContext) {
				expectError(
					testConnection(ctx, msgpack.EncodedJSON{
						"port": 8883, "secure": true, "ca_file": "/no/such/ca.crt",
					}),
					HavePrefix("failed to read the CA file: "),
				)
			})
		})

		DescribeTable("Should reject properties that are not valid",
			func(
				ctx SpecContext,
				build func() (msgpack.EncodedJSON, string),
			) {
				properties, message := build()
				if _, ok := properties["port"]; !ok {
					properties["port"] = 8883
				}
				expectError(testConnection(ctx, properties), Equal(message))
			},
			Entry("a certificate with no key", func() (msgpack.EncodedJSON, string) {
				return msgpack.EncodedJSON{"cert_file": "/certs/client.crt"},
					"invalid broker properties: key_file: " +
						"cert_file and key_file must be set together: validation error"
			}),
			Entry("a key with no certificate", func() (msgpack.EncodedJSON, string) {
				return msgpack.EncodedJSON{"key_file": "/certs/client.key"},
					"invalid broker properties: key_file: " +
						"cert_file and key_file must be set together: validation error"
			}),
			Entry("a port of zero", func() (msgpack.EncodedJSON, string) {
				return msgpack.EncodedJSON{"port": 0},
					"invalid broker properties: port: must be 1 to 65535: " +
						"validation error"
			}),
			Entry("a port above 65535", func() (msgpack.EncodedJSON, string) {
				return msgpack.EncodedJSON{"port": 65536},
					"invalid broker properties: port: must be 1 to 65535: " +
						"validation error"
			}),
			Entry("a keep-alive of zero", func() (msgpack.EncodedJSON, string) {
				return msgpack.EncodedJSON{"keep_alive": 0},
					"invalid broker properties: keep_alive: must be 1 to 65535: " +
						"validation error"
			}),
			Entry("a negative keep-alive", func() (msgpack.EncodedJSON, string) {
				return msgpack.EncodedJSON{"keep_alive": -1},
					"invalid broker properties: keep_alive: must be 1 to 65535: " +
						"validation error"
			}),
			Entry("a keep-alive above 65535", func() (msgpack.EncodedJSON, string) {
				return msgpack.EncodedJSON{"keep_alive": 65536},
					"invalid broker properties: keep_alive: must be 1 to 65535: " +
						"validation error"
			}),
			Entry("a Sparkplug B host ID that cannot be a topic level",
				func() (msgpack.EncodedJSON, string) {
					return msgpack.EncodedJSON{
						"sparkplug": map[string]any{"host_id": "scada/1"},
					}, "invalid broker properties: sparkplug.host_id: " +
						"scada/1 must not hold /, +, or #: validation error"
				},
			),
			Entry("a Sparkplug B group that cannot be a topic level",
				func() (msgpack.EncodedJSON, string) {
					return msgpack.EncodedJSON{
						"sparkplug": map[string]any{
							"groups": []any{"plant", "l+b"},
						},
					}, "invalid broker properties: sparkplug.groups: " +
						"l+b must not hold /, +, or #: validation error"
				},
			),
			Entry("a CA file that holds no PEM certificates",
				func() (msgpack.EncodedJSON, string) {
					caFile := filepath.Join(GinkgoT().TempDir(), "ca.crt")
					Expect(os.WriteFile(caFile, []byte("not a certificate"), 0o600)).
						To(Succeed())
					return msgpack.EncodedJSON{"secure": true, "ca_file": caFile},
						"CA file " + caFile + " holds no PEM certificates: " +
							"validation error"
				},
			),
		)
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
			stat := answer(ctx, configured, t, task.Command{
				Type: "browse", Key: "browse", Args: args,
			})
			Expect(stat.Variant).To(Equal(status.VariantSuccess))
			return stat
		}

		field := func(pointer, t, sample string) map[string]any {
			return map[string]any{"pointer": pointer, "type": t, "sample": sample}
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
						"topic":    "plant/a",
						"payload":  `{"v":1}`,
						"fields":   []any{field("/v", "number", "1")},
						"retained": true,
					},
					map[string]any{
						"topic":    "plant/b",
						"payload":  `{"v":2}`,
						"fields":   []any{field("/v", "number", "2")},
						"retained": true,
					},
				}))
			},
		)

		It("Should give the scalars of each JSON payload in document order",
			func(ctx SpecContext) {
				broker.publish("plant/a", "23.4", true)
				broker.publish("plant/b", `{"state": "ON", "mode": "AUTO"}`, true)
				broker.publish(
					"plant/c",
					`{"out": {"flow/rate~avg": 1.2}, "n": [1], "e": null, "on": true}`,
					true,
				)
				broker.publish("plant/d", "not json", true)
				broker.publish("plant/e", `{"v": 1, "v": 2}`, true)
				broker.publish("plant/f", `{"v": 1} {"v": 2}`, true)
				stat := browse(ctx, msgpack.EncodedJSON{"duration": 200})
				fields := func(t any) any { return t.(map[string]any)["fields"] }
				Expect(stat.Details.Data["topics"]).To(HaveExactElements(
					WithTransform(fields, Equal([]any{field("", "number", "23.4")})),
					WithTransform(fields, Equal([]any{
						field("/state", "string", `"ON"`),
						field("/mode", "string", `"AUTO"`),
					})),
					WithTransform(fields, Equal([]any{
						field("/out/flow~1rate~0avg", "number", "1.2"),
						field("/on", "boolean", "true"),
					})),
					WithTransform(fields, BeEmpty()),
					WithTransform(fields, BeEmpty()),
					WithTransform(fields, BeEmpty()),
				))
			},
		)

		It("Should give every scalar of a payload longer than its sample",
			func(ctx SpecContext) {
				long := strings.Repeat("x", 300)
				broker.publish("plant/a", `{"text": "`+long+`", "v": 1}`, true)
				stat := browse(ctx, msgpack.EncodedJSON{"duration": 200})
				topic := stat.Details.Data["topics"].([]any)[0]
				Expect(topic).To(HaveKeyWithValue("payload", HaveLen(256)))
				Expect(topic).To(HaveKeyWithValue("fields", []any{
					field("/text", "string", `"`+long[:63]),
					field("/v", "number", "1"),
				}))
			},
		)

		It("Should give at most 256 scalars of a payload", func(ctx SpecContext) {
			names := make([]string, 300)
			for i := range names {
				names[i] = fmt.Sprintf(`"f%03d": %d`, i, i)
			}
			broker.publish("plant/a", "{"+strings.Join(names, ", ")+"}", true)
			stat := browse(ctx, msgpack.EncodedJSON{"duration": 200})
			topic := stat.Details.Data["topics"].([]any)[0]
			fields := topic.(map[string]any)["fields"].([]any)
			Expect(fields).To(HaveLen(256))
			Expect(fields[255]).To(Equal(field("/f255", "number", "255")))
		})

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

		It("Should report a filter that the broker refuses", func(ctx SpecContext) {
			port := broker.port
			broker.stop()
			broker = startBrokerWith(brokerOptions{
				port:    port,
				refused: []string{"plant/#"},
			})
			configured, t := scan(ctx)
			stat := answer(ctx, configured, t, task.Command{
				Type: "browse", Key: "browse",
				Args: msgpack.EncodedJSON{
					"device": dev.Key, "filter": "plant/#", "duration": 50,
				},
			})
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Message).To(Equal(
				"the broker refused the subscription to plant/#",
			))
		})

		It("Should report a device that does not exist", func(ctx SpecContext) {
			configured, t := scan(ctx)
			stat := answer(ctx, configured, t, task.Command{
				Type: "browse", Key: "browse",
				Args: msgpack.EncodedJSON{"device": "missing"},
			})
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Message).To(Equal(
				"Devices with keys [missing] not found: not found: query",
			))
		})
	})

	Describe("Exec", func() {
		It("Should leave a command it does not support unanswered",
			func(ctx SpecContext) {
				configured, t := scan(ctx)
				Expect(configured.Exec(ctx, task.Command{Type: "reboot", Key: "x"})).
					To(Succeed())
				Consistently(func() string { return taskStatus(ctx, t).Details.Cmd }).
					WithTimeout(100 * time.Millisecond).
					ShouldNot(Equal("x"))
			},
		)
	})
})
