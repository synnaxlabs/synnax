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
	"fmt"
	"strings"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// browsedTopic is one topic of the data that answers a browse.
type browsedTopic struct {
	Name       string          `json:"name"`
	Partitions int             `json:"partitions"`
	Records    int64           `json:"records"`
	Compacted  bool            `json:"compacted"`
	Rate       float64         `json:"rate"`
	Newest     telem.TimeStamp `json:"newest"`
	Format     driver.Format   `json:"format"`
	Readable   bool            `json:"readable"`
}

// sampledTopic is one topic of the data that answers a sample.
type sampledTopic struct {
	Topic   string `json:"topic"`
	Records []struct {
		Offset  int64               `json:"offset"`
		Key     string              `json:"key"`
		Headers []map[string]string `json:"headers"`
		Format  driver.Format       `json:"format"`
		Payload string              `json:"payload"`
	} `json:"records"`
	Shapes []xjson.Shape `json:"shapes"`
	Error  string        `json:"error"`
}

var _ = Describe("Discover", func() {
	var (
		dt driver.Task
		t  task.Task
	)

	BeforeEach(func(ctx SpecContext) {
		t = newTask(kafka.ScanTaskType, map[string]any{"rate": 20})
		dt = MustSucceed(newFactory().ConfigureTask(ctx, t, driver.NoCommand))
		DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
	})

	// command runs a command of typ with args, decodes the data that answers it into
	// res, and returns the status that answers it.
	command := func(ctx context.Context, typ string, args, res any) task.Status {
		GinkgoHelper()
		key := uuid.New().String()
		Expect(dt.Exec(ctx, task.Command{
			Task: t.Key, Type: typ, Key: key, Args: toJSON(args),
		})).To(Succeed())
		var stat task.Status
		Eventually(func(g Gomega) {
			stat = taskStatus(ctx, t)
			g.Expect(stat.Details.Cmd).To(Equal(key))
		}).WithTimeout(15 * time.Second).Should(Succeed())
		if stat.Variant == status.VariantSuccess {
			Expect(stat.Details.Data.Unmarshal(res)).To(Succeed())
		}
		return stat
	}

	Describe("Browse", func() {
		// browse returns the topics of the cluster of props.
		browse := func(ctx context.Context, props kafka.Properties) []browsedTopic {
			GinkgoHelper()
			var res struct {
				Topics []browsedTopic `json:"topics"`
			}
			dev := createDevice(ctx, props)
			stat := command(ctx, "browse", map[string]any{"device": dev.Key}, &res)
			Expect(stat.Variant).To(Equal(status.VariantSuccess))
			return res.Topics
		}

		// named returns the topic of topics named name.
		named := func(topics []browsedTopic, name string) browsedTopic {
			GinkgoHelper()
			for _, t := range topics {
				if t.Name == name {
					return t
				}
			}
			Fail("no topic named " + name)
			return browsedTopic{}
		}

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
			topics := browse(ctx, clusterProperties())

			j := named(topics, jsonTopic)
			Expect(j.Partitions).To(Equal(2))
			Expect(j.Records).To(Equal(int64(3)))
			Expect(j.Rate).To(BeNumerically("~", 3.0/60, 1e-9))
			Expect(j.Format).To(Equal(driver.FormatJSON))
			Expect(j.Readable).To(BeTrue())
			Expect(
				j.Newest,
			).To(BeNumerically(">=", before-telem.TimeStamp(telem.Millisecond)))
			Expect(named(topics, srTopic).Format).
				To(Equal(driver.Format("schema_registry")))
			empty := named(topics, emptyTopic)
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
			topics := browse(ctx, kafka.Properties{
				Brokers: aclCluster.ListenAddrs(),
				Sasl: kafka.SASL{
					Variant: kafka.PlainSASL{Username: "reader", Password: "pass"},
				},
			})
			Expect(named(topics, readable).Readable).To(BeTrue())
			Expect(named(topics, hidden).Readable).To(BeFalse())
		})

		It("Should report a device that does not exist", func(ctx SpecContext) {
			stat := command(ctx, "browse", map[string]any{"device": "missing"}, nil)
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Message).To(HavePrefix("retrieving device missing: "))
		})
	})

	Describe("Sample", func() {
		// sample samples topics of the suite's cluster and returns the data that
		// answers it.
		sample := func(
			ctx context.Context,
			topics []string,
			count int,
		) ([]sampledTopic, bool) {
			GinkgoHelper()
			var res struct {
				Topics    []sampledTopic `json:"topics"`
				Truncated bool           `json:"truncated"`
			}
			stat := command(ctx, "sample", map[string]any{
				"device": createDevice(ctx, clusterProperties()).Key,
				"topics": topics,
				"count":  count,
			}, &res)
			Expect(stat.Variant).To(Equal(status.VariantSuccess))
			return res.Topics, res.Truncated
		}

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
				Value:   fmt.Appendf(nil, `{"ts":%d,"psi":4}`, base+2),
				Headers: []kgo.RecordHeader{{Key: "source", Value: []byte("rig")}},
			}).FirstErr()).To(Succeed())
			samples, truncated := sample(ctx, []string{topic}, 0)
			Expect(truncated).To(BeFalse())
			Expect(samples).To(HaveLen(1))
			Expect(samples[0].Error).To(BeEmpty())
			Expect(samples[0].Records).To(HaveLen(7))
			last := samples[0].Records[6]
			Expect(last.Offset).To(Equal(int64(6)))
			Expect(last.Key).To(Equal("tank-1"))
			Expect(last.Format).To(Equal(driver.FormatJSON))
			Expect(last.Payload).To(Equal(fmt.Sprintf(`{"ts":%d,"psi":4}`, base+2)))
			Expect(last.Headers).
				To(Equal([]map[string]string{{"key": "source", "value": "rig"}}))
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
				samples, _ := sample(ctx, []string{topic}, count)
				offsets := make([]int64, len(samples[0].Records))
				for i, r := range samples[0].Records {
					offsets[i] = r.Offset
				}
				Expect(offsets).To(Equal(expected))
			},
			Entry("a count of one", 1, []int64{49}),
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
				samples, _ := sample(ctx, []string{"missing-" + topic, topic}, 0)
				Expect(samples[0].Error).To(Equal("The topic does not exist."))
				Expect(samples[1].Records).To(HaveLen(1))
			},
		)

		It("Should infer shapes from records that do not fit in the answer",
			func(ctx SpecContext) {
				topic := uniqueTopic()
				Expect(cluster.CreateTopic(topic, 1, nil)).To(Succeed())
				cl := newKafkaClient()
				pad := strings.Repeat("x", 15<<10)
				for i := range 80 {
					produceJSON(ctx, cl, topic, "", map[string]any{"v": i, "pad": pad})
				}
				samples, truncated := sample(ctx, []string{topic}, 80)
				Expect(truncated).To(BeTrue())
				Expect(len(samples[0].Records)).To(BeNumerically("<", 80))
				Expect(samples[0].Shapes).To(HaveLen(1))
				Expect(samples[0].Shapes[0].Count).To(Equal(80))
			},
		)
	})
})
