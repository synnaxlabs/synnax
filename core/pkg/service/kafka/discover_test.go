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
	"crypto/tls"
	"fmt"
	"net"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/security/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/address"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// stages returns the stage and variant of each step.
func stages(steps []driver.TestStep) []string {
	out := make([]string, len(steps))
	for i, s := range steps {
		out[i] = s.Stage + ":" + string(s.Variant)
	}
	return out
}

// closedAddress returns a local address that refuses connections.
func closedAddress() string {
	GinkgoHelper()
	l := MustSucceed(net.Listen("tcp", "127.0.0.1:0"))
	addr := l.Addr().String()
	Expect(l.Close()).To(Succeed())
	return addr
}

// topicNamed returns the topic of topics named name.
func topicNamed(topics []kafka.Topic, name string) kafka.Topic {
	GinkgoHelper()
	for _, t := range topics {
		if t.Name == name {
			return t
		}
	}
	Fail("no topic named " + name)
	return kafka.Topic{}
}

var _ = Describe("Discover", func() {
	Describe("TestConnection", func() {
		It("Should pass every stage of a reachable cluster", func(ctx SpecContext) {
			steps := MustSucceed(kafka.TestConnection(ctx, clusterProperties()))
			Expect(stages(steps)).To(Equal([]string{
				"dial:success", "metadata:success", "broker:success",
			}))
			Expect(steps[0].Address).To(Equal(cluster.ListenAddrs()[0]))
		})

		It("Should stop at a broker that does not answer", func(ctx SpecContext) {
			addr := closedAddress()
			steps := MustSucceed(kafka.TestConnection(
				ctx, kafka.Properties{Brokers: []string{addr}},
			))
			Expect(stages(steps)).To(Equal([]string{"dial:error"}))
			Expect(steps[0].Message).To(Equal("Failed to reach broker"))
			Expect(steps[0].Description).To(HavePrefix("Could not reach " + addr))
		})

		It(
			"Should warn about a seed broker that does not answer",
			func(ctx SpecContext) {
				steps := MustSucceed(kafka.TestConnection(ctx, kafka.Properties{
					Brokers: append(
						[]string{closedAddress()},
						cluster.ListenAddrs()...,
					),
				}))
				Expect(stages(steps)).To(Equal([]string{
					"dial:warning", "metadata:success", "broker:success",
				}))
			},
		)

		It("Should fail the TLS stage of a broker without TLS", func(ctx SpecContext) {
			props := clusterProperties()
			props.TLS = true
			steps := MustSucceed(kafka.TestConnection(ctx, props))
			Expect(stages(steps)).To(Equal([]string{"dial:success", "tls:error"}))
		})

		It("Should fail the TLS stage of a broker with an untrusted certificate",
			func(ctx SpecContext) {
				certs := MustSucceed(cert.NewFactory(cert.FactoryConfig{
					FS:      xfs.NewMem(),
					KeySize: testutil.SmallKeySize,
				}))
				Expect(certs.CreateCAPair()).To(Succeed())
				leaf := MustSucceed(certs.SignNodeCert([]address.Address{"127.0.0.1"}))
				tlsCluster := MustSucceed(kfake.NewCluster(
					kfake.NumBrokers(1),
					kfake.TLS(&tls.Config{Certificates: []tls.Certificate{*leaf}}),
				))
				DeferCleanup(tlsCluster.Close)
				steps := MustSucceed(kafka.TestConnection(ctx, kafka.Properties{
					Brokers: tlsCluster.ListenAddrs(),
					TLS:     true,
				}))
				Expect(stages(steps)).To(Equal([]string{"dial:success", "tls:error"}))
				Expect(steps[1].Message).To(Equal("Certificate not trusted"))
			},
		)

		Describe("SASL", func() {
			var brokers []string
			BeforeEach(func() {
				saslCluster := MustSucceed(kfake.NewCluster(
					kfake.NumBrokers(1),
					kfake.EnableSASL(),
					kfake.Superuser("PLAIN", "user", "pass"),
				))
				DeferCleanup(saslCluster.Close)
				brokers = saslCluster.ListenAddrs()
			})

			It(
				"Should pass the SASL stage with the right password",
				func(ctx SpecContext) {
					steps := MustSucceed(kafka.TestConnection(ctx, kafka.Properties{
						Brokers: brokers,
						Sasl: kafka.SASL{
							Variant: kafka.PlainSASL{
								Username: "user",
								Password: "pass",
							},
						},
					}))
					Expect(stages(steps)).To(Equal([]string{
						"dial:success",
						"sasl:success",
						"metadata:success",
						"broker:success",
					}))
				},
			)

			It(
				"Should fail the SASL stage with a wrong password",
				func(ctx SpecContext) {
					steps := MustSucceed(kafka.TestConnection(ctx, kafka.Properties{
						Brokers: brokers,
						Sasl: kafka.SASL{
							Variant: kafka.PlainSASL{
								Username: "user",
								Password: "wrong",
							},
						},
					}))
					Expect(
						stages(steps),
					).To(Equal([]string{"dial:success", "sasl:error"}))
					Expect(steps[1].Message).To(Equal("Authentication failed"))
					Expect(steps[1].Variant).To(Equal(status.VariantError))
				},
			)
		})

		It("Should reject properties with no brokers", func(ctx SpecContext) {
			Expect(kafka.TestConnection(ctx, kafka.Properties{})).Error().
				To(MatchError(ContainSubstring("brokers: must be non-empty")))
		})
	})

	Describe("Browse", func() {
		It("Should describe each topic", func(ctx SpecContext) {
			jsonTopic, srTopic, emptyTopic := uniqueTopic(), uniqueTopic(), uniqueTopic()
			Expect(cluster.CreateTopic(jsonTopic, 2, nil)).To(Succeed())
			Expect(cluster.CreateTopic(srTopic, 1, nil)).To(Succeed())
			Expect(cluster.CreateTopic(
				emptyTopic, 1, map[string]string{"cleanup.policy": "compact"},
			)).To(Succeed())
			cl := newKafkaClient()
			before := telem.Now()
			for i := range 3 {
				produceJSON(ctx, cl, jsonTopic, fmt.Sprint(i), map[string]int{"v": i})
			}
			Expect(cl.ProduceSync(ctx, &kgo.Record{
				Topic: srTopic,
				Value: []byte{0, 0, 0, 0, 7, 2, 4},
			}).FirstErr()).To(Succeed())
			topics := MustSucceed(kafka.Browse(ctx, clusterProperties()))

			j := topicNamed(topics, jsonTopic)
			Expect(j.Partitions).To(Equal(2))
			Expect(j.Records).To(Equal(int64(3)))
			Expect(j.Rate).To(BeNumerically("~", 3.0/60, 1e-9))
			Expect(j.Format).To(Equal(driver.FormatJSON))
			Expect(j.Readable).To(BeTrue())
			Expect(
				j.Newest,
			).To(BeNumerically(">=", before-telem.TimeStamp(telem.Millisecond)))
			Expect(topicNamed(topics, srTopic).Format).
				To(Equal(driver.FormatSchemaRegistry))
			empty := topicNamed(topics, emptyTopic)
			Expect(empty.Records).To(BeZero())
			Expect(empty.Format).To(BeEmpty())
			Expect(empty.Compacted).To(BeTrue())
		})

		It("Should mark a topic that the user cannot read", func(ctx SpecContext) {
			readable, hidden := uniqueTopic(), uniqueTopic()
			aclCluster := MustSucceed(kfake.NewCluster(
				kfake.NumBrokers(1),
				kfake.EnableSASL(),
				kfake.EnableACLs(),
				kfake.SeedTopics(1, readable, hidden),
				kfake.User("PLAIN", "reader", "pass",
					kfake.ACL{
						Resource:  kmsg.ACLResourceTypeTopic,
						Name:      readable,
						Pattern:   kmsg.ACLResourcePatternTypeLiteral,
						Operation: kmsg.ACLOperationRead,
						Allow:     true,
					},
					kfake.ACL{
						Resource:  kmsg.ACLResourceTypeTopic,
						Name:      readable,
						Pattern:   kmsg.ACLResourcePatternTypeLiteral,
						Operation: kmsg.ACLOperationDescribe,
						Allow:     true,
					},
					kfake.ACL{
						Resource:  kmsg.ACLResourceTypeTopic,
						Name:      hidden,
						Pattern:   kmsg.ACLResourcePatternTypeLiteral,
						Operation: kmsg.ACLOperationDescribe,
						Allow:     true,
					},
				),
			))
			DeferCleanup(aclCluster.Close)
			topics := MustSucceed(kafka.Browse(ctx, kafka.Properties{
				Brokers: aclCluster.ListenAddrs(),
				Sasl: kafka.SASL{
					Variant: kafka.PlainSASL{Username: "reader", Password: "pass"},
				},
			}))
			Expect(topicNamed(topics, readable).Readable).To(BeTrue())
			Expect(topicNamed(topics, hidden).Readable).To(BeFalse())
		})
	})

	Describe("Sample", func() {
		It("Should infer one shape for each set of keys", func(ctx SpecContext) {
			topic := uniqueTopic()
			Expect(cluster.CreateTopic(topic, 1, nil)).To(Succeed())
			cl := newKafkaClient()
			base := time.Now().UnixMilli()
			for i := range 2 {
				ts := base + int64(i)
				produceJSON(
					ctx,
					cl,
					topic,
					"tank-1",
					map[string]int64{"ts": ts, "psi": 1},
				)
				produceJSON(
					ctx,
					cl,
					topic,
					"pump-1",
					map[string]int64{"ts": ts, "rpm": 2},
				)
				produceJSON(
					ctx,
					cl,
					topic,
					"tank-2",
					map[string]int64{"ts": ts, "psi": 3},
				)
			}
			Expect(cl.ProduceSync(ctx, &kgo.Record{
				Topic:   topic,
				Key:     []byte("tank-1"),
				Value:   []byte(fmt.Sprintf(`{"ts":%d,"psi":4}`, base+2)),
				Headers: []kgo.RecordHeader{{Key: "source", Value: []byte("rig")}},
			}).FirstErr()).To(Succeed())
			samples := MustSucceed(
				kafka.Sample(ctx, clusterProperties(), kafka.SampleRequest{
					Topics: []string{topic},
				}),
			)
			Expect(samples).To(HaveLen(1))
			Expect(samples[0].Error).To(BeEmpty())
			Expect(samples[0].Records).To(HaveLen(7))
			last := samples[0].Records[6]
			Expect(last.Offset).To(Equal(int64(6)))
			Expect(last.Key).To(Equal("tank-1"))
			Expect(
				last.Headers,
			).To(Equal([]kafka.Header{{Key: "source", Value: "rig"}}))
			shapes := samples[0].Shapes
			Expect(shapes).To(HaveLen(2))
			Expect(shapes[0].Sources).To(Equal([]string{"tank-1", "tank-2"}))
			Expect(shapes[0].Fields[shapes[0].Index].Pointer.String()).To(Equal("/ts"))
			Expect(shapes[1].Sources).To(Equal([]string{"pump-1"}))
		})

		DescribeTable("Should read windows spread back in time",
			func(ctx SpecContext, count int, expected []int64) {
				topic := uniqueTopic()
				Expect(cluster.CreateTopic(topic, 1, nil)).To(Succeed())
				cl := newKafkaClient()
				for i := range 50 {
					produceJSON(ctx, cl, topic, "", map[string]int{"v": i})
				}
				samples := MustSucceed(
					kafka.Sample(ctx, clusterProperties(), kafka.SampleRequest{
						Topics: []string{topic},
						Count:  count,
					}),
				)
				offsets := make([]int64, len(samples[0].Records))
				for i, r := range samples[0].Records {
					offsets[i] = r.Offset
				}
				Expect(offsets).To(Equal(expected))
			},
			Entry("a count that splits evenly", 9,
				[]int64{0, 1, 2, 23, 24, 25, 47, 48, 49}),
			Entry("a count that does not split evenly", 10,
				[]int64{0, 1, 2, 3, 24, 25, 26, 47, 48, 49}),
		)

		It("Should report a topic that does not exist and read the rest",
			func(ctx SpecContext) {
				topic := uniqueTopic()
				Expect(cluster.CreateTopic(topic, 1, nil)).To(Succeed())
				produceJSON(ctx, newKafkaClient(), topic, "", map[string]int{"v": 1})
				samples := MustSucceed(kafka.Sample(
					ctx,
					clusterProperties(),
					kafka.SampleRequest{Topics: []string{"missing-" + topic, topic}},
				))
				Expect(samples[0].Error).To(Equal("The topic does not exist."))
				Expect(samples[1].Records).To(HaveLen(1))
			},
		)
	})
})
