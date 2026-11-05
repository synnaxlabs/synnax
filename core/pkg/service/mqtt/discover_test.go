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
	"fmt"
	"net"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// stages returns the stage and variant of each step.
func stages(steps []driver.TestStep) []string {
	out := make([]string, len(steps))
	for i, s := range steps {
		out[i] = s.Stage + ":" + string(s.Variant)
	}
	return out
}

// publishUntilDone publishes payload to topic every 10 ms until the spec ends.
func publishUntilDone(b *testBroker, topic, payload string) {
	GinkgoHelper()
	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer GinkgoRecover()
		defer close(stopped)
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				Expect(b.server.Publish(topic, []byte(payload), false, 0)).To(Succeed())
			}
		}
	}()
	DeferCleanup(func() {
		close(done)
		Eventually(stopped).Should(BeClosed())
	})
}

var _ = Describe("Discover", func() {
	var (
		factory *mqtt.Factory
		dev     device.Device
		broker  *testBroker
	)

	BeforeEach(func(ctx SpecContext) {
		factory = MustOpen(mqtt.OpenFactory(mqtt.FactoryConfig{
			Device:  deviceSvc,
			Channel: channelSvc,
			Framer:  framerSvc,
			DB:      db,
			Status:  statusSvc,
		}))
		broker = startBroker(0)
		dev = createBrokerDevice(ctx, createRack(ctx), broker.port, nil)
	})

	Describe("TestConnection", func() {
		test := func(ctx SpecContext, props msgpack.EncodedJSON) []driver.TestStep {
			GinkgoHelper()
			return MustSucceed(factory.TestConnection(ctx, mqtt.TestRequest{
				Location:   "127.0.0.1",
				Properties: props,
			}))
		}

		It("Should pass every stage of a broker that is not a device yet",
			func(ctx SpecContext) {
				steps := test(ctx, msgpack.EncodedJSON{"port": broker.port})
				Expect(
					stages(steps),
				).To(Equal([]string{"dial:success", "connect:success"}))
				Expect(
					steps[0].Address,
				).To(Equal(fmt.Sprintf("127.0.0.1:%d", broker.port)))
			},
		)

		It("Should stop at a broker that does not answer", func(ctx SpecContext) {
			l := MustSucceed(net.Listen("tcp", "127.0.0.1:0"))
			port := l.Addr().(*net.TCPAddr).Port
			Expect(l.Close()).To(Succeed())
			steps := test(ctx, msgpack.EncodedJSON{"port": port})
			Expect(stages(steps)).To(Equal([]string{"dial:error"}))
			Expect(steps[0].Message).To(Equal("Failed to reach broker"))
		})

		It(
			"Should fail the connect stage with a wrong password",
			func(ctx SpecContext) {
				secured := startBrokerWith(
					brokerOptions{username: "user", password: "pass"},
				)
				steps := test(ctx, msgpack.EncodedJSON{
					"port": secured.port, "username": "user", "password": "wrong",
				})
				Expect(
					stages(steps),
				).To(Equal([]string{"dial:success", "connect:error"}))
				Expect(steps[1].Message).To(Equal("Authentication failed"))
			},
		)

		It("Should pass the TLS stage of a broker that its CA file verifies",
			func(ctx SpecContext) {
				tlsCfg, caFile := createCerts()
				secured := startBrokerWith(brokerOptions{tls: tlsCfg})
				steps := test(ctx, msgpack.EncodedJSON{
					"port": secured.port, "secure": true, "ca_file": caFile,
				})
				Expect(stages(steps)).To(Equal([]string{
					"dial:success", "tls:success", "connect:success",
				}))
			},
		)

		It("Should fail the TLS stage of a broker that its roots do not verify",
			func(ctx SpecContext) {
				tlsCfg, _ := createCerts()
				secured := startBrokerWith(brokerOptions{tls: tlsCfg})
				steps := test(
					ctx,
					msgpack.EncodedJSON{"port": secured.port, "secure": true},
				)
				Expect(stages(steps)).To(Equal([]string{"dial:success", "tls:error"}))
				Expect(steps[1].Message).To(Equal("Certificate not trusted"))
			},
		)

		It("Should reject properties that are not valid", func(ctx SpecContext) {
			Expect(factory.TestConnection(ctx, mqtt.TestRequest{
				Location:   "127.0.0.1",
				Properties: msgpack.EncodedJSON{"port": 0},
			})).Error().To(MatchError(validate.ErrValidation))
		})
	})

	Describe("Browse", func() {
		It("Should count the messages of each topic", func(ctx SpecContext) {
			broker.publish("plant/a", `{"v":1}`, true)
			publishUntilDone(broker, "plant/b", "ON")
			res := MustSucceed(factory.Browse(ctx, mqtt.BrowseRequest{
				Device:   dev.Key,
				Filter:   "plant/#",
				Duration: 300,
			}))
			Expect(res.Truncated).To(BeFalse())
			Expect(res.Topics).To(HaveLen(2))
			a, b := res.Topics[0], res.Topics[1]
			Expect(a.Topic).To(Equal("plant/a"))
			Expect(a.Count).To(Equal(1))
			Expect(a.Rate).To(BeZero())
			Expect(a.Retained).To(BeTrue())
			Expect(a.Payload).To(Equal([]byte(`{"v":1}`)))
			Expect(a.Format).To(Equal(driver.FormatJSON))
			Expect(a.LastSeen).ToNot(BeZero())
			Expect(b.Topic).To(Equal("plant/b"))
			Expect(b.Count).To(BeNumerically(">", 1))
			Expect(b.Rate).To(BeNumerically(">", 0))
			Expect(b.Format).To(Equal(driver.FormatText))
		})
	})

	Describe("Sample", func() {
		It("Should end once each exact topic holds its count", func(ctx SpecContext) {
			now := time.Now().UnixMilli()
			broker.publish(
				"rig/1/temp",
				fmt.Sprintf(`{"value":1.5,"time":%d}`, now),
				true,
			)
			broker.publish(
				"rig/2/temp",
				fmt.Sprintf(`{"value":2.5,"time":%d}`, now),
				true,
			)
			start := time.Now()
			res := MustSucceed(factory.Sample(ctx, mqtt.SampleRequest{
				Device: dev.Key,
				Topics: []string{"rig/1/temp", "rig/2/temp"},
				Count:  1,
			}))
			Expect(time.Since(start)).To(BeNumerically("<", 2*time.Second))
			Expect(res.Topics).To(HaveLen(2))
			Expect(res.Topics[0].Topic).To(Equal("rig/1/temp"))
			Expect(res.Topics[0].Messages[0].Retained).To(BeTrue())
			Expect(res.Shapes).To(HaveLen(1))
			shape := res.Shapes[0]
			Expect(shape.Sources).To(Equal([]string{"rig/1/temp", "rig/2/temp"}))
			Expect(shape.Fields[shape.Index].Pointer.String()).To(Equal("/time"))
		})

		It("Should keep at most the count of each topic of a wildcard",
			func(ctx SpecContext) {
				publishUntilDone(broker, "line/a", `{"v":1}`)
				res := MustSucceed(factory.Sample(ctx, mqtt.SampleRequest{
					Device:   dev.Key,
					Topics:   []string{"line/#"},
					Count:    3,
					Duration: 300,
				}))
				Expect(res.Topics).To(HaveLen(1))
				Expect(res.Topics[0].Messages).To(HaveLen(3))
			},
		)

		It("Should stop keeping topics at the most that a sample keeps",
			func(ctx SpecContext) {
				for i := range 501 {
					broker.publish(fmt.Sprintf("many/%03d", i), "1", true)
				}
				res := MustSucceed(factory.Sample(ctx, mqtt.SampleRequest{
					Device:   dev.Key,
					Topics:   []string{"many/#"},
					Count:    1,
					Duration: 500,
				}))
				Expect(res.Truncated).To(BeTrue())
				Expect(res.Topics).To(HaveLen(500))
			},
		)

		It("Should reject a sample with no topics", func(ctx SpecContext) {
			Expect(factory.Sample(ctx, mqtt.SampleRequest{Device: dev.Key})).Error().
				To(MatchError(validate.ErrValidation))
		})
	})
})
