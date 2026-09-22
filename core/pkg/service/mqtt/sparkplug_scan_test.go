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
	"maps"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Sparkplug B browse", func() {
	var (
		broker  *testBroker
		rackKey rack.Key
		factory driver.Factory
	)

	BeforeEach(func(ctx SpecContext) {
		broker = startBroker(0)
		rackKey = createRack(ctx)
		factory = newFactory(64)
	})

	// scan configures a disabled scan task. It runs commands but checks no device, so
	// the only client on the broker is the client of a browse.
	scan := func(ctx context.Context) (driver.Task, task.Task) {
		GinkgoHelper()
		t := newTask(rackKey, mqtt.ScanTaskType, msgpack.EncodedJSON{"disabled": true})
		return configure(ctx, factory, t), t
	}

	browseCommand := func(dev device.Device, args msgpack.EncodedJSON) task.Command {
		// A table entry shares its arguments between runs.
		args = maps.Clone(args)
		args["device"] = dev.Key
		return task.Command{Type: "browse_sparkplug", Key: "browse", Args: args}
	}

	// browse runs a browse that succeeds and returns the data of its result.
	browse := func(
		ctx context.Context,
		dev device.Device,
		args msgpack.EncodedJSON,
	) msgpack.EncodedJSON {
		GinkgoHelper()
		configured, t := scan(ctx)
		Expect(configured.Exec(ctx, browseCommand(dev, args))).To(Succeed())
		stat := taskStatus(ctx, t)
		Expect(stat.Variant).To(Equal(status.VariantSuccess))
		Expect(stat.Details.Cmd).To(Equal("browse"))
		return stat.Details.Data
	}

	// expectFailed runs a browse that fails with an error that matches matcher.
	expectFailed := func(
		ctx context.Context,
		cmd task.Command,
		message string,
		matcher types.GomegaMatcher,
	) {
		GinkgoHelper()
		configured, t := scan(ctx)
		Expect(configured.Exec(ctx, cmd)).
			To(And(matcher, MatchError(ContainSubstring(message))))
		stat := taskStatus(ctx, t)
		Expect(stat.Variant).To(Equal(status.VariantError))
		Expect(stat.Message).To(ContainSubstring(message))
		Expect(stat.Details.Cmd).To(Equal("browse"))
	}

	Describe("Edge nodes", func() {
		DescribeTable("Should list the edge nodes that publish during the browse",
			func(
				ctx SpecContext,
				groups []any,
				group string,
				filters int,
				expected []any,
			) {
				dev := createBrokerDevice(
					ctx,
					rackKey,
					broker.port,
					msgpack.EncodedJSON{
						"sparkplug": map[string]any{"groups": groups},
					},
				)
				bench := startEdgeNode(broker, "lab", "bench")
				line2 := startEdgeNode(broker, "plant", "line2")
				line1 := startEdgeNode(broker, "plant", "line1")
				line1.setTags("pump", tagValue{name: "speed", alias: 1, value: 1.0})
				line1.setTags("fan", tagValue{name: "speed", alias: 1, value: 2.0})
				configured, t := scan(ctx)
				cmd := browseCommand(dev, msgpack.EncodedJSON{
					"group": group, "duration": 300,
				})
				result := make(chan error, 1)
				go func() {
					defer GinkgoRecover()
					result <- configured.Exec(ctx, cmd)
				}()
				// An edge node reports by exception, so it must publish after the
				// browse has subscribed.
				Eventually(broker.subscriptions).Should(Equal(filters))
				broker.publish("spBv1.0/STATE/scada", `{"online":true}`, false)
				bench.birth()
				line2.birth()
				line1.birth()
				Eventually(result, "5s").Should(Receive(BeNil()))
				stat := taskStatus(ctx, t)
				Expect(stat.Variant).To(Equal(status.VariantSuccess))
				Expect(stat.Details.Data).To(HaveKeyWithValue("tags", BeEmpty()))
				Expect(stat.Details.Data["nodes"]).To(Equal(expected))
			},
			Entry("every group", nil, "", 1, []any{
				map[string]any{
					"group": "lab", "edge_node": "bench", "devices": []any{},
				},
				map[string]any{
					"group": "plant", "edge_node": "line1",
					"devices": []any{"fan", "pump"},
				},
				map[string]any{
					"group": "plant", "edge_node": "line2", "devices": []any{},
				},
			}),
			Entry("the group of the command", nil, "plant", 1, []any{
				map[string]any{
					"group": "plant", "edge_node": "line1",
					"devices": []any{"fan", "pump"},
				},
				map[string]any{
					"group": "plant", "edge_node": "line2", "devices": []any{},
				},
			}),
			Entry("the groups of the broker device", []any{"lab", "field"}, "", 2,
				[]any{map[string]any{
					"group": "lab", "edge_node": "bench", "devices": []any{},
				}},
			),
			Entry("the group of the command over the groups of the broker device",
				[]any{"lab"}, "plant", 1,
				[]any{
					map[string]any{
						"group": "plant", "edge_node": "line1",
						"devices": []any{"fan", "pump"},
					},
					map[string]any{
						"group": "plant", "edge_node": "line2", "devices": []any{},
					},
				},
			),
		)

		It("Should list no edge node when none publishes", func(ctx SpecContext) {
			dev := createBrokerDevice(ctx, rackKey, broker.port, nil)
			startEdgeNode(broker, "plant", "line1")
			data := browse(ctx, dev, msgpack.EncodedJSON{"duration": 100})
			Expect(data).To(HaveKeyWithValue("nodes", BeEmpty()))
		})
	})

	Describe("Tags", func() {
		var (
			dev  device.Device
			node *testEdgeNode
		)

		BeforeEach(func(ctx SpecContext) {
			dev = createBrokerDevice(ctx, rackKey, broker.port, nil)
			node = startEdgeNode(broker, "plant", "line1")
		})

		It("Should ask for a rebirth and list the tags of the births, in order",
			func(ctx SpecContext) {
				node.setTags(
					"",
					tagValue{name: "temperature", alias: 1, value: 23.5},
					tagValue{name: "mode", alias: 2, value: "auto"},
					tagValue{name: "count", alias: 3, value: int32(-4)},
					tagValue{name: sparkplug.RebirthMetric, alias: 4, value: false},
				)
				node.setTags(
					"pump",
					tagValue{name: "speed", alias: 1, value: 1200.5},
					tagValue{name: "running", alias: 2, value: true},
				)
				data := browse(ctx, dev, msgpack.EncodedJSON{
					"group": "plant", "edge_node": "line1", "duration": 2000,
				})
				Expect(node.births()).To(Equal(1))
				Expect(data).To(HaveKeyWithValue("nodes", BeEmpty()))
				tag := func(dev, name, dataType, value string) map[string]any {
					return map[string]any{
						"device":    dev,
						"name":      name,
						"data_type": dataType,
						"value":     value,
						"supported": true,
					}
				}
				Expect(data["tags"]).To(Equal([]any{
					tag("", "Node Control/Rebirth", "boolean", "false"),
					tag("", "count", "int32", "-4"),
					tag("", "mode", "string", "auto"),
					tag("", "temperature", "double", "23.5"),
					tag("pump", "running", "boolean", "true"),
					tag("pump", "speed", "double", "1200.5"),
				}))
			},
		)

		It("Should report an edge node that does not answer", func(ctx SpecContext) {
			node.setSilent(true)
			expectFailed(
				ctx,
				browseCommand(dev, msgpack.EncodedJSON{
					"group": "plant", "edge_node": "line1", "duration": 200,
				}),
				"edge node plant/line1 did not answer the rebirth request",
				Not(MatchError(validate.ErrValidation)),
			)
			Expect(node.births()).To(BeZero())
		})

		DescribeTable("Should reject an ID that is not valid",
			func(ctx SpecContext, group, edgeNode, message string) {
				expectFailed(
					ctx,
					browseCommand(dev, msgpack.EncodedJSON{
						"group": group, "edge_node": edgeNode, "duration": 200,
					}),
					message,
					MatchError(validate.ErrValidation),
				)
			},
			Entry("an edge node with a topic separator",
				"plant", "line/1", "edge_node: line/1 must not hold /, +, or #",
			),
			Entry("an edge node with a wildcard",
				"plant", "+", "edge_node: + must not hold /, +, or #",
			),
			Entry("no group", "", "line1", "group: required"),
		)
	})

	Describe("Arguments", func() {
		It("Should reject arguments of the wrong type", func(ctx SpecContext) {
			dev := createBrokerDevice(ctx, rackKey, broker.port, nil)
			expectFailed(
				ctx,
				browseCommand(dev, msgpack.EncodedJSON{"duration": "long"}),
				"invalid arguments",
				MatchError(validate.ErrValidation),
			)
		})

		It("Should report a device that does not exist", func(ctx SpecContext) {
			configured, t := scan(ctx)
			Expect(configured.Exec(ctx, task.Command{
				Type: "browse_sparkplug", Key: "browse",
				Args: msgpack.EncodedJSON{"device": "missing"},
			})).To(MatchError(query.ErrNotFound))
			Expect(taskStatus(ctx, t).Variant).To(Equal(status.VariantError))
		})
	})

	Describe("Client", func() {
		DescribeTable("Should leave the broker when the browse ends",
			func(
				ctx SpecContext,
				silent bool,
				args msgpack.EncodedJSON,
				result types.GomegaMatcher,
			) {
				dev := createBrokerDevice(ctx, rackKey, broker.port, nil)
				startEdgeNode(broker, "plant", "line1").setSilent(silent)
				configured, _ := scan(ctx)
				Expect(configured.Exec(ctx, browseCommand(dev, args))).To(result)
				Eventually(broker.clientCount).Should(BeZero())
				Consistently(broker.clientCount, "100ms").Should(BeZero())
			},
			Entry("a browse for edge nodes",
				false, msgpack.EncodedJSON{"duration": 50}, Succeed(),
			),
			Entry("a browse for tags",
				false,
				msgpack.EncodedJSON{
					"group": "plant", "edge_node": "line1", "duration": 2000,
				},
				Succeed(),
			),
			Entry("a browse for tags with no answer",
				true,
				msgpack.EncodedJSON{
					"group": "plant", "edge_node": "line1", "duration": 50,
				},
				MatchError(ContainSubstring("did not answer the rebirth request")),
			),
		)
	})
})
