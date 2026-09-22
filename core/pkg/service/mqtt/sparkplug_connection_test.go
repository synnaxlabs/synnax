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
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

const stateTopic = "spBv1.0/STATE/synnax"

// hostState is the payload of one STATE message.
type hostState struct {
	Online bool `json:"online"`
	// Timestamp is in milliseconds.
	Timestamp int64 `json:"timestamp"`
}

func parseHostState(payload string) hostState {
	GinkgoHelper()
	Expect(payload).To(MatchRegexp(`^\{"online":(true|false),"timestamp":\d+\}$`))
	var s hostState
	Expect(json.Unmarshal([]byte(payload), &s)).To(Succeed())
	return s
}

func newTimedFactory(rebirthInterval, birthGrace time.Duration) driver.Factory {
	GinkgoHelper()
	return MustSucceed(mqtt.NewFactory(mqtt.FactoryConfig{
		Device:          deviceSvc,
		Channel:         channelSvc,
		Framer:          framerSvc,
		Status:          statusSvc,
		RebirthInterval: rebirthInterval,
		BirthGrace:      birthGrace,
	}))
}

var _ = Describe("Sparkplug B host session", func() {
	var (
		broker  *testBroker
		rackKey rack.Key
		factory driver.Factory
	)

	BeforeEach(func(ctx SpecContext) {
		broker = startBroker(0)
		rackKey = createRack(ctx)
		factory = newTimedFactory(testRebirthInterval, testBirthGrace)
	})

	// run configures and starts a read task.
	run := func(ctx context.Context, cfg msgpack.EncodedJSON) (driver.Task, task.Task) {
		GinkgoHelper()
		t := newTask(rackKey, mqtt.ReadTaskType, cfg)
		configured := configure(ctx, factory, t)
		Expect(configured.Exec(ctx, task.Command{Type: "start", Key: "start"})).
			To(Succeed())
		return configured, t
	}

	Describe("STATE message", func() {
		hostProperties := func(hostID string) msgpack.EncodedJSON {
			return msgpack.EncodedJSON{"sparkplug": map[string]any{"host_id": hostID}}
		}

		// retainedState returns the STATE message that the broker retains.
		retainedState := func() hostState {
			GinkgoHelper()
			retained := broker.retained(stateTopic)
			Expect(retained).To(HaveLen(1))
			return parseHostState(retained[0])
		}

		It("Should retain an online message while a task runs and an offline one after",
			func(ctx SpecContext) {
				dev := createBrokerDevice(
					ctx, rackKey, broker.port, hostProperties("synnax"),
				)
				_, data := createIndexed(ctx, telem.Float64T)
				before := int64(telem.Now() / telem.MillisecondTS)
				configured, _ := run(ctx, readConfig(
					dev, plainEntry("plant/state", field("v", data[0], "")),
				))
				Eventually(func(g Gomega) {
					g.Expect(broker.retained(stateTopic)).To(HaveLen(1))
				}).Should(Succeed())
				online := retainedState()
				Expect(online.Online).To(BeTrue())
				Expect(online.Timestamp).To(And(
					BeNumerically(">=", before),
					BeNumerically("<=", int64(telem.Now()/telem.MillisecondTS)),
				))
				Expect(configured.Stop(false)).To(Succeed())
				offline := retainedState()
				Expect(offline.Online).To(BeFalse())
				Expect(offline.Timestamp).To(BeNumerically(">=", online.Timestamp))
				Eventually(broker.clientCount).Should(BeZero())
				// A clean stop sends no last will over the offline message.
				Expect(retainedState()).To(Equal(offline))
			},
		)

		It("Should publish no STATE message with no host ID", func(ctx SpecContext) {
			dev := createBrokerDevice(ctx, rackKey, broker.port, nil)
			_, data := createIndexed(ctx, telem.Float64T)
			configured, _ := run(ctx, readConfig(
				dev, plainEntry("plant/state", field("v", data[0], "")),
			))
			Eventually(broker.subscriptions).Should(Equal(1))
			Expect(broker.retained("spBv1.0/STATE/+")).To(BeEmpty())
			Expect(configured.Stop(false)).To(Succeed())
			Eventually(broker.clientCount).Should(BeZero())
			Expect(broker.retained("spBv1.0/STATE/+")).To(BeEmpty())
		})

		It(
			"Should reject a host ID that is not one topic level",
			func(ctx SpecContext) {
				dev := createBrokerDevice(
					ctx, rackKey, broker.port, hostProperties("syn/nax"),
				)
				_, data := createIndexed(ctx, telem.Float64T)
				t := newTask(rackKey, mqtt.ReadTaskType, readConfig(
					dev, plainEntry("plant/state", field("v", data[0], "")),
				))
				message := "invalid broker properties: sparkplug.host_id: syn/nax " +
					"must not hold /, +, or #"
				Expect(factory.ConfigureTask(ctx, t, "configure")).Error().To(And(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(message)),
				))
				stat := taskStatus(ctx, t)
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Message).To(ContainSubstring(message))
			},
		)

		It("Should leave an offline last will and come back online after a drop",
			func(ctx SpecContext) {
				broker.collect(stateTopic)
				states := func() []hostState {
					GinkgoHelper()
					var out []hostState
					for _, m := range broker.collected() {
						out = append(out, parseHostState(m.payload))
					}
					return out
				}
				dev := createBrokerDevice(
					ctx, rackKey, broker.port, hostProperties("synnax"),
				)
				_, data := createIndexed(ctx, telem.Float64T)
				_, t := run(ctx, readConfig(
					dev, plainEntry("plant/state", field("v", data[0], "")),
				))
				Eventually(states).Should(HaveLen(1))
				broker.dropClients()
				Eventually(states, "10s").Should(HaveLen(3))
				got := states()
				Expect(got[0].Online).To(BeTrue())
				// The last will carries the timestamp of the online message.
				Expect(got[1]).To(Equal(hostState{Timestamp: got[0].Timestamp}))
				Expect(got[2].Online).To(BeTrue())
				Expect(got[2].Timestamp).To(BeNumerically(">", got[0].Timestamp))
				Expect(retainedState()).To(Equal(got[2]))
				Eventually(func() status.Variant { return taskStatus(ctx, t).Variant }).
					Should(Equal(status.VariantSuccess))
				broker.publish("plant/state", "6", false)
				Eventually(func() int64 { return stored(ctx, data[0]).Len() }).
					Should(BeEquivalentTo(1))
			},
		)
	})

	Describe("Rebirth requests", func() {
		var (
			dev  device.Device
			node *testEdgeNode
		)

		BeforeEach(func(ctx SpecContext) {
			dev = createBrokerDevice(ctx, rackKey, broker.port, nil)
			node = startEdgeNode(broker, "plant", "line1")
		})

		It("Should share one connection and ask for a birth for each new task",
			func(ctx SpecContext) {
				_, a := createIndexed(ctx, telem.Float64T)
				_, b := createIndexed(ctx, telem.Float64T)
				node.setTags(
					"",
					tagValue{
						name:      "temperature",
						alias:     1,
						value:     1.5,
						timestamp: 1700000000000,
					},
					tagValue{
						name:      "pressure",
						alias:     2,
						value:     2.5,
						timestamp: 1700000000000,
					},
				)
				_, first := run(ctx, readConfig(
					dev, tagEntry("plant", "line1", "", "temperature", a[0]),
				))
				Eventually(node.births).Should(Equal(1))
				_, second := run(ctx, readConfig(
					dev, tagEntry("plant", "line1", "", "pressure", b[0]),
				))
				Eventually(node.births).Should(Equal(2))
				Expect(broker.clientCount()).To(Equal(1))
				node.data(
					"",
					tagValue{alias: 1, value: 3.5, timestamp: 1700000001000},
					tagValue{alias: 2, value: 4.5, timestamp: 1700000001000},
				)
				Eventually(func() int64 { return stored(ctx, a[0]).Len() }).
					Should(BeEquivalentTo(2))
				Eventually(func() int64 { return stored(ctx, b[0]).Len() }).
					Should(BeEquivalentTo(2))
				Expect(stored(ctx, a[0])).
					To(telem.MatchSeriesData(telem.NewSeriesV(1.5, 3.5)))
				Expect(stored(ctx, b[0])).
					To(telem.MatchSeriesData(telem.NewSeriesV(2.5, 4.5)))
				Expect(taskStatus(ctx, first).Variant).To(Equal(status.VariantSuccess))
				Expect(taskStatus(ctx, second).Variant).To(Equal(status.VariantSuccess))
				Expect(node.births()).To(Equal(2))
			},
		)

		It("Should hold a rebirth request until the rebirth interval has passed",
			func(ctx SpecContext) {
				factory = newTimedFactory(300*time.Millisecond, time.Second)
				_, a := createIndexed(ctx, telem.Float64T)
				_, b := createIndexed(ctx, telem.Float64T)
				node.setTags("", tagValue{
					name: "temperature", alias: 1, value: 1.5, timestamp: 1700000000000,
				})
				first := configure(ctx, factory, newTask(
					rackKey, mqtt.ReadTaskType, readConfig(
						dev, tagEntry("plant", "line1", "", "temperature", a[0]),
					),
				))
				secondTask := newTask(rackKey, mqtt.ReadTaskType, readConfig(
					dev, tagEntry("plant", "line1", "", "temperature", b[0]),
				))
				second := configure(ctx, factory, secondTask)
				Expect(first.Exec(ctx, task.Command{Type: "start", Key: "start"})).
					To(Succeed())
				Eventually(node.births).Should(Equal(1))
				Expect(second.Exec(ctx, task.Command{Type: "start", Key: "start"})).
					To(Succeed())
				Consistently(node.births, "150ms", "10ms").Should(Equal(1))
				Eventually(node.births, "2s").Should(Equal(2))
				requests := node.rebirthRequests()
				Expect(requests).To(HaveLen(2))
				sent := func(c command) uint64 { return c.payload.GetTimestamp() }
				Expect(sent(requests[1]) - sent(requests[0])).
					To(BeNumerically(">=", 250))
				Eventually(func() int64 { return stored(ctx, b[0]).Len() }).
					Should(BeEquivalentTo(1))
				// The held request is no missing birth.
				Expect(taskStatus(ctx, secondTask).Variant).
					To(Equal(status.VariantSuccess))
			},
		)

		It("Should ask for a rebirth again after the broker returns",
			func(ctx SpecContext) {
				_, data := createIndexed(ctx, telem.Float64T)
				node.setTags("", tagValue{
					name: "temperature", alias: 1, value: 1.0, timestamp: 1700000000000,
				})
				_, t := run(ctx, readConfig(
					dev, tagEntry("plant", "line1", "", "temperature", data[0]),
				))
				Eventually(func() int64 { return stored(ctx, data[0]).Len() }).
					Should(BeEquivalentTo(1))
				port := broker.port
				broker.stop()
				Eventually(func(g Gomega) {
					stat := taskStatus(ctx, t)
					g.Expect(stat.Variant).To(Equal(status.VariantWarning))
					g.Expect(stat.Message).
						To(ContainSubstring("broker is not connected"))
				}, "5s").Should(Succeed())
				broker = startBroker(port)
				// An edge node belongs to one broker, so a new one takes its place.
				restarted := startEdgeNode(broker, "plant", "line1")
				restarted.setTags("", tagValue{
					name: "temperature", alias: 1, value: 2.0, timestamp: 1700000001000,
				})
				Eventually(restarted.births, "10s").Should(Equal(1))
				Expect(restarted.rebirthRequests()).To(HaveLen(1))
				Eventually(func() int64 { return stored(ctx, data[0]).Len() }).
					Should(BeEquivalentTo(2))
				Expect(stored(ctx, data[0])).
					To(telem.MatchSeriesData(telem.NewSeriesV(1.0, 2.0)))
				Eventually(func() status.Variant { return taskStatus(ctx, t).Variant }).
					Should(Equal(status.VariantSuccess))
			},
		)

		It("Should follow an edge node again after its last task stopped",
			func(ctx SpecContext) {
				_, plain := createIndexed(ctx, telem.Float64T)
				_, a := createIndexed(ctx, telem.Float64T)
				_, b := createIndexed(ctx, telem.Float64T)
				// A plain task keeps the connection open while no task follows the edge
				// node.
				run(ctx, readConfig(
					dev, plainEntry("plant/keeper", field("v", plain[0], "")),
				))
				node.setTags("", tagValue{
					name: "temperature", alias: 1, value: 1.0, timestamp: 1700000000000,
				})
				first, _ := run(ctx, readConfig(
					dev, tagEntry("plant", "line1", "", "temperature", a[0]),
				))
				Eventually(func() int64 { return stored(ctx, a[0]).Len() }).
					Should(BeEquivalentTo(1))
				Expect(first.Stop(false)).To(Succeed())
				Expect(broker.clientCount()).To(Equal(1))
				node.setTags("", tagValue{
					name: "temperature", alias: 1, value: 2.0, timestamp: 1700000001000,
				})
				_, second := run(ctx, readConfig(
					dev, tagEntry("plant", "line1", "", "temperature", b[0]),
				))
				Eventually(node.births).Should(Equal(2))
				Eventually(func() int64 { return stored(ctx, b[0]).Len() }).
					Should(BeEquivalentTo(1))
				Expect(stored(ctx, b[0])).
					To(telem.MatchSeriesData(telem.NewSeriesV(2.0)))
				node.data("", tagValue{alias: 1, value: 3.0, timestamp: 1700000002000})
				Eventually(func() int64 { return stored(ctx, b[0]).Len() }).
					Should(BeEquivalentTo(2))
				Expect(stored(ctx, a[0]).Len()).To(BeEquivalentTo(1))
				Expect(taskStatus(ctx, second).Variant).To(Equal(status.VariantSuccess))
				Expect(broker.clientCount()).To(Equal(1))
			},
		)
	})

	Describe("Entries", func() {
		It("Should write a plain entry and a Sparkplug B entry of one task",
			func(ctx SpecContext) {
				dev := createBrokerDevice(ctx, rackKey, broker.port, nil)
				node := startEdgeNode(broker, "plant", "line1")
				_, plain := createIndexed(ctx, telem.Float64T)
				_, tag := createIndexed(ctx, telem.Float64T)
				node.setTags("", tagValue{
					name:      "temperature",
					alias:     1,
					value:     23.5,
					timestamp: 1700000000000,
				})
				_, t := run(ctx, readConfig(
					dev,
					plainEntry("plant/mixed", field("v", plain[0], "")),
					tagEntry("plant", "line1", "", "temperature", tag[0]),
				))
				Eventually(func() int64 { return stored(ctx, tag[0]).Len() }).
					Should(BeEquivalentTo(1))
				broker.publish("plant/mixed", "7", false)
				Eventually(func() int64 { return stored(ctx, plain[0]).Len() }).
					Should(BeEquivalentTo(1))
				Expect(stored(ctx, tag[0])).
					To(telem.MatchSeriesData(telem.NewSeriesV(23.5)))
				Expect(stored(ctx, plain[0])).
					To(telem.MatchSeriesData(telem.NewSeriesV(7.0)))
				Expect(broker.clientCount()).To(Equal(1))
				Expect(taskStatus(ctx, t).Variant).To(Equal(status.VariantSuccess))
			},
		)
	})

	Describe("NewFactory", func() {
		DescribeTable("Should reject timing that is not valid",
			func(rebirthInterval, birthGrace time.Duration, message string) {
				Expect(mqtt.NewFactory(mqtt.FactoryConfig{
					Device:          deviceSvc,
					Channel:         channelSvc,
					Framer:          framerSvc,
					Status:          statusSvc,
					RebirthInterval: rebirthInterval,
					BirthGrace:      birthGrace,
				})).Error().To(MatchError(ContainSubstring(message)))
			},
			Entry("a birth grace equal to the rebirth interval",
				time.Second, time.Second,
				"birth_grace: must be longer than rebirth_interval",
			),
			Entry("a birth grace shorter than the rebirth interval",
				2*time.Second, time.Second,
				"birth_grace: must be longer than rebirth_interval",
			),
			Entry("a rebirth interval longer than the default birth grace",
				time.Minute, time.Duration(0),
				"birth_grace: must be longer than rebirth_interval",
			),
			Entry("a negative rebirth interval",
				-time.Second, time.Duration(0), "rebirth_interval: must be positive",
			),
		)
	})
})
