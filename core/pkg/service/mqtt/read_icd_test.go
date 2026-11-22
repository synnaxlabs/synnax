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
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// jsonField returns a JSON field of an ICD message.
func jsonField(name, pointer string, v icd.JSONFieldVariant) icd.JSONField {
	base := icd.JSONBaseField{Key: uuid.New(), Name: name, Scale: 1, Pointer: pointer}
	f := icd.JSONField{Variant: v}
	f.SetBase(base)
	return f
}

func jsonMessage(
	name string,
	match *icd.JSONMatch,
	fields ...icd.JSONField,
) icd.Message {
	return icd.Message{
		Key:  uuid.New(),
		Name: name,
		Payload: icd.Payload{
			Variant: icd.JSONPayload{Match: match, Fields: fields},
		},
	}
}

func createICD(ctx context.Context, messages ...icd.Message) icd.ICD {
	GinkgoHelper()
	ic := icd.ICD{Name: "Plant", Messages: messages}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return icdSvc.NewWriter(tx).Create(ctx, &ic)
	})).To(Succeed())
	return ic
}

// jsonFields returns the fields of m, which must have a JSON payload.
func jsonFields(m icd.Message) []icd.JSONField {
	GinkgoHelper()
	p, ok := m.Payload.Variant.(icd.JSONPayload)
	Expect(ok).To(BeTrue())
	return p.Fields
}

func fieldKey(m icd.Message, i int) icd.FieldKey {
	GinkgoHelper()
	return jsonFields(m)[i].Base().Key
}

// binding returns the config that writes field i of m to ch.
func binding(m icd.Message, i int, ch channel.Channel) map[string]any {
	return map[string]any{"field": fieldKey(m, i), "channel": ch.Key()}
}

// icdEntry returns the config of a plain read entry that reads messages of ic.
func icdEntry(topic string, ic icd.ICD, messages ...map[string]any) map[string]any {
	list := make([]any, len(messages))
	for i, m := range messages {
		list[i] = m
	}
	return map[string]any{
		"key":      topic,
		"type":     "plain",
		"topic":    topic,
		"icd":      ic.Key,
		"messages": list,
	}
}

// readMessage returns the config that reads m through bindings.
func readMessage(m icd.Message, bindings ...map[string]any) map[string]any {
	list := make([]any, len(bindings))
	for i, b := range bindings {
		list[i] = b
	}
	return map[string]any{"message": m.Key, "bindings": list}
}

var _ = Describe("Read task ICD messages", func() {
	var (
		broker  *testBroker
		rackKey rack.Key
		dev     device.Device
		factory driver.Factory
	)

	BeforeEach(func(ctx SpecContext) {
		broker = startBroker(0)
		rackKey = createRack(ctx)
		dev = createBrokerDevice(ctx, rackKey, broker.port, nil)
		factory = newFactory(64)
	})

	start := func(ctx context.Context, cfg msgpack.EncodedJSON) task.Task {
		GinkgoHelper()
		t := newTask(ctx, rackKey, mqtt.ReadTaskType, cfg)
		configured := configure(ctx, factory, t)
		Expect(configured.Exec(ctx, task.Command{Type: "start", Key: "start"})).
			To(Succeed())
		Eventually(broker.subscriptions).Should(BeNumerically(">", 0))
		return t
	}

	It("Should write the fields of a message with its arrival time",
		func(ctx SpecContext) {
			m := jsonMessage("Status", nil,
				jsonField("temp", "/temp", icd.FloatJSONField{}),
				jsonField("state", "/state", icd.StringJSONField{}),
			)
			ic := createICD(ctx, m)
			idx, data := createIndexed(ctx, telem.Float64T, telem.StringT)
			before := telem.Now()
			t := start(ctx, readConfig(ctx, dev, icdEntry(
				"plant/status", ic,
				readMessage(m, binding(m, 0, data[0]), binding(m, 1, data[1])),
			)))
			broker.publish("plant/status", `{"temp":21.5,"state":"run"}`, false)
			Eventually(func() int64 { return stored(ctx, data[0]).Len() }).
				Should(BeEquivalentTo(1))
			Expect(stored(ctx, data[0])).
				To(telem.MatchSeriesData(telem.NewSeriesV(21.5)))
			Expect(stored(ctx, data[1])).
				To(telem.MatchSeriesData(telem.NewSeriesV("run")))
			stamp := stored(ctx, idx).ValueAt[telem.TimeStamp](0)
			Expect(stamp).To(BeNumerically(">=", before))
			Expect(taskStatus(ctx, t).Variant).To(Equal(status.VariantSuccess))
		},
	)

	It("Should read each document as the message whose match rule it meets",
		func(ctx SpecContext) {
			statusMsg := jsonMessage(
				"Status",
				&icd.JSONMatch{
					Variant: icd.StringJSONMatch{Pointer: "/type", Value: "s"},
				},
				jsonField("ts", "/ts", icd.TimestampJSONField{
					TimeFormat: payload.TimeFormatUnixSec,
				}),
				jsonField("temp", "/temp", icd.FloatJSONField{}),
			)
			event := jsonMessage(
				"Event",
				&icd.JSONMatch{
					Variant: icd.StringJSONMatch{Pointer: "/type", Value: "e"},
				},
				jsonField("ts", "/ts", icd.TimestampJSONField{
					TimeFormat: payload.TimeFormatUnixSec,
				}),
				jsonField("code", "/code", icd.FloatJSONField{}),
			)
			ic := createICD(ctx, statusMsg, event)
			statusIdx, statusData := createIndexed(ctx, telem.Float64T)
			eventIdx, eventData := createIndexed(ctx, telem.Float64T)
			statusIndex, eventIndex := fieldKey(statusMsg, 0), fieldKey(event, 0)
			statusRead := readMessage(
				statusMsg,
				binding(statusMsg, 0, statusIdx),
				binding(statusMsg, 1, statusData[0]),
			)
			statusRead["index"] = statusIndex
			eventRead := readMessage(
				event, binding(event, 0, eventIdx), binding(event, 1, eventData[0]),
			)
			eventRead["index"] = eventIndex
			start(
				ctx,
				readConfig(ctx, dev, icdEntry("plant/all", ic, statusRead, eventRead)),
			)
			broker.publish("plant/all", `{"type":"s","ts":1,"temp":20}`, false)
			broker.publish("plant/all", `{"type":"e","ts":2,"code":7}`, false)
			Eventually(func() int64 { return stored(ctx, eventData[0]).Len() }).
				Should(BeEquivalentTo(1))
			Expect(stored(ctx, statusData[0])).
				To(telem.MatchSeriesData(telem.NewSeriesV(20.0)))
			Expect(stored(ctx, eventData[0])).
				To(telem.MatchSeriesData(telem.NewSeriesV(7.0)))
			Expect(stored(ctx, statusIdx)).
				To(telem.MatchSeriesData(telem.NewSeriesV(telem.SecondTS)))
			Expect(stored(ctx, eventIdx)).
				To(telem.MatchSeriesData(telem.NewSeriesV(2 * telem.SecondTS)))
		},
	)

	It("Should write the messages of one topic on their own indexes in one frame",
		func(ctx SpecContext) {
			factory = newFactory(1024)
			match := func(value string) *icd.JSONMatch {
				return &icd.JSONMatch{
					Variant: icd.StringJSONMatch{Pointer: "/type", Value: value},
				}
			}
			statusMsg := jsonMessage(
				"Status", match("s"), jsonField("temp", "/v", icd.FloatJSONField{}),
			)
			event := jsonMessage(
				"Event", match("e"), jsonField("code", "/v", icd.FloatJSONField{}),
			)
			ic := createICD(ctx, statusMsg, event)
			statusIdx, statusData := createIndexed(ctx, telem.Float64T)
			eventIdx, eventData := createIndexed(ctx, telem.Float64T)
			streamer := stream(
				ctx,
				channel.Keys{statusData[0].Key(), eventData[0].Key()},
			)
			t := start(ctx, readConfig(ctx, dev, icdEntry(
				"plant/all", ic,
				readMessage(statusMsg, binding(statusMsg, 0, statusData[0])),
				readMessage(event, binding(event, 0, eventData[0])),
			)))
			const documents = 200
			for i := range documents {
				broker.publish(
					"plant/all",
					fmt.Sprintf(`{"type":%q,"v":%d}`, []string{"s", "e"}[i%2], i),
					false,
				)
			}
			var samples, mixed int
			for samples < documents {
				fr := streamer.receive()
				statusLen := fr.Get(statusData[0].Key()).Len()
				eventLen := fr.Get(eventData[0].Key()).Len()
				if statusLen > 0 && eventLen > 0 {
					mixed++
				}
				samples += int(statusLen + eventLen)
			}
			Expect(mixed).To(BeNumerically(">", 0))
			for _, ch := range []channel.Channel{
				statusIdx, statusData[0], eventIdx, eventData[0],
			} {
				Expect(stored(ctx, ch).Len()).To(BeEquivalentTo(documents / 2))
			}
			Expect(taskStatus(ctx, t).Variant).To(Equal(status.VariantSuccess))
		},
	)

	It("Should use a field's default when a document has no value",
		func(ctx SpecContext) {
			fallback := payload.Float(-1)
			m := jsonMessage("Status", nil,
				jsonField("temp", "/temp", icd.FloatJSONField{Default: &fallback}),
			)
			ic := createICD(ctx, m)
			_, data := createIndexed(ctx, telem.Float64T)
			start(ctx, readConfig(ctx, dev, icdEntry(
				"plant/status", ic, readMessage(m, binding(m, 0, data[0])),
			)))
			broker.publish("plant/status", `{}`, false)
			Eventually(func() int64 { return stored(ctx, data[0]).Len() }).
				Should(BeEquivalentTo(1))
			Expect(stored(ctx, data[0])).
				To(telem.MatchSeriesData(telem.NewSeriesV(-1.0)))
		},
	)

	Describe("Rejected messages", func() {
		warned := func(ctx context.Context, t task.Task, description string) {
			GinkgoHelper()
			Eventually(func(g Gomega) {
				stat := taskStatus(ctx, t)
				g.Expect(stat.Variant).To(Equal(status.VariantWarning))
				g.Expect(stat.Message).To(Equal("Skipped 1 of 1 message"))
				g.Expect(stat.Description).To(Equal(description))
				g.Expect(stat.Details.Running).To(BeTrue())
			}).Should(Succeed())
		}

		It("Should warn and write nothing for a document that matches no message",
			func(ctx SpecContext) {
				m := jsonMessage(
					"Status",
					&icd.JSONMatch{
						Variant: icd.StringJSONMatch{Pointer: "/type", Value: "s"},
					},
					jsonField("temp", "/temp", icd.FloatJSONField{}),
				)
				ic := createICD(ctx, m)
				_, data := createIndexed(ctx, telem.Float64T)
				t := start(ctx, readConfig(ctx, dev, icdEntry(
					"plant/status", ic, readMessage(m, binding(m, 0, data[0])),
				)))
				broker.publish("plant/status", `{"type":"e","temp":1}`, false)
				warned(ctx, t,
					"Message on topic plant/status: the payload matches no message",
				)
				broker.publish("plant/status", `{"type":"s","temp":2}`, false)
				Eventually(func() int64 { return stored(ctx, data[0]).Len() }).
					Should(BeEquivalentTo(1))
				Expect(stored(ctx, data[0])).
					To(telem.MatchSeriesData(telem.NewSeriesV(2.0)))
			},
		)

		It("Should warn and write nothing for a value that does not decode",
			func(ctx SpecContext) {
				m := jsonMessage("Status", nil,
					jsonField("temp", "/temp", icd.FloatJSONField{}),
				)
				ic := createICD(ctx, m)
				_, data := createIndexed(ctx, telem.Float64T)
				t := start(ctx, readConfig(ctx, dev, icdEntry(
					"plant/status", ic, readMessage(m, binding(m, 0, data[0])),
				)))
				broker.publish("plant/status", `{"temp":"hot"}`, false)
				warned(ctx, t, "Message on topic plant/status: field temp: "+
					`unknown enum label "hot": validation error`,
				)
				Expect(stored(ctx, data[0]).Len()).To(BeZero())
			},
		)
	})

	Describe("Configure", func() {
		configureErr := func(ctx context.Context, cfg msgpack.EncodedJSON) error {
			t := newTask(ctx, rackKey, mqtt.ReadTaskType, cfg)
			_, err := factory.ConfigureTask(ctx, t, "configure")
			return err
		}

		// format is an ICD of a Status message with a timestamp and two floats and an
		// Event message with one float, and channels on two indexes to bind them to.
		type format struct {
			m, n       icd.Message
			ic         icd.ICD
			idx, other channel.Channel
			data       []channel.Channel
		}

		createFormat := func(ctx context.Context) format {
			m := jsonMessage("Status", nil,
				jsonField("ts", "/ts", icd.TimestampJSONField{
					TimeFormat: payload.TimeFormatUnixSec,
				}),
				jsonField("temp", "/temp", icd.FloatJSONField{}),
				jsonField("pressure", "/pressure", icd.FloatJSONField{}),
			)
			n := jsonMessage(
				"Event",
				&icd.JSONMatch{
					Variant: icd.StringJSONMatch{Pointer: "/type", Value: "event"},
				},
				jsonField("code", "/code", icd.FloatJSONField{}),
			)
			idx, data := createIndexed(ctx, telem.Float64T, telem.Float64T)
			_, other := createIndexed(ctx, telem.Float64T)
			return format{
				m:     m,
				n:     n,
				ic:    createICD(ctx, m, n),
				idx:   idx,
				other: other[0],
				data:  data,
			}
		}

		type entries = func(f format) []map[string]any

		// status reads the message of f on plant/status.
		status := func(f format, read map[string]any) []map[string]any {
			return []map[string]any{icdEntry("plant/status", f.ic, read)}
		}

		DescribeTable("Should reject an invalid entry",
			func(ctx SpecContext, build entries, msg string) {
				cfg := readConfig(ctx, dev, build(createFormat(ctx))...)
				Expect(configureErr(ctx, cfg)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(msg)),
				))
			},
			Entry("a topic with a wildcard", func(f format) []map[string]any {
				read := readMessage(f.m, binding(f.m, 1, f.data[0]))
				return []map[string]any{icdEntry("plant/+", f.ic, read)}
			}, "must not hold the wildcards + or #"),
			Entry("a topic that reads no message", func(f format) []map[string]any {
				return []map[string]any{icdEntry("plant/status", f.ic)}
			}, "topic plant/status reads no messages"),
			Entry("a message that the ICD lacks", func(f format) []map[string]any {
				other := jsonMessage(
					"Other", nil, jsonField("v", "/v", icd.FloatJSONField{}),
				)
				return status(f, readMessage(other, binding(other, 0, f.data[0])))
			}, "does not exist in ICD Plant"),
			Entry("a message that binds no field", func(f format) []map[string]any {
				return status(f, readMessage(f.m))
			}, "message Status binds no fields"),
			Entry("a channel that does not exist", func(f format) []map[string]any {
				missing := map[string]any{"field": fieldKey(f.m, 1), "channel": 1 << 30}
				return status(f, readMessage(f.m, missing))
			}, "message Status: channel 1073741824 does not exist"),
			Entry("channels on two indexes", func(f format) []map[string]any {
				return status(f, readMessage(
					f.m, binding(f.m, 1, f.data[0]), binding(f.m, 2, f.other),
				))
			}, "every channel must share one index channel"),
			Entry("an index field on a data channel", func(f format) []map[string]any {
				read := readMessage(f.m, binding(f.m, 0, f.data[0]))
				read["index"] = fieldKey(f.m, 0)
				return status(f, read)
			}, "the index field must write to an index channel"),
			Entry("a data field on an index channel", func(f format) []map[string]any {
				return status(f, readMessage(f.m, binding(f.m, 1, f.idx)))
			}, "only the index field can write to an index channel"),
			Entry("an index field with no channel", func(f format) []map[string]any {
				read := readMessage(f.m, binding(f.m, 1, f.data[0]))
				read["index"] = fieldKey(f.m, 0)
				return status(f, read)
			}, "the index field has no channel"),
			Entry("only disabled entries", func(f format) []map[string]any {
				list := status(f, readMessage(f.m, binding(f.m, 1, f.data[0])))
				list[0]["disabled"] = true
				return list
			}, "the task has no enabled entries"),
			Entry("a topic read twice", func(f format) []map[string]any {
				read := readMessage(f.m, binding(f.m, 1, f.data[0]))
				second := icdEntry("plant/status", f.ic, read)
				second["key"] = "second"
				return append(status(f, read), second)
			}, "topic plant/status appears more than once"),
			Entry("two topics on one channel", func(f format) []map[string]any {
				read := readMessage(f.m, binding(f.m, 1, f.data[0]))
				return []map[string]any{
					icdEntry("plant/a", f.ic, read), icdEntry("plant/b", f.ic, read),
				}
			}, "write to the same channel"),
			Entry("two messages on one index", func(f format) []map[string]any {
				return []map[string]any{icdEntry(
					"plant/status", f.ic,
					readMessage(f.m, binding(f.m, 1, f.data[0])),
					readMessage(f.n, binding(f.n, 0, f.data[1])),
				)}
			}, "write to the same channel"),
		)

		It("Should reject an ICD that does not exist", func(ctx SpecContext) {
			m := jsonMessage("Status", nil, jsonField("t", "/t", icd.FloatJSONField{}))
			_, data := createIndexed(ctx, telem.Float64T)
			missing := icd.ICD{Key: uuid.New()}
			entry := icdEntry(
				"plant/status", missing, readMessage(m, binding(m, 0, data[0])),
			)
			Expect(configureErr(ctx, readConfig(ctx, dev, entry))).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"icd: ICD %s does not exist", missing.Key,
				)),
			))
		})

		It("Should reject a field whose kind does not convert to its channel",
			func(ctx SpecContext) {
				m := jsonMessage("Status", nil,
					jsonField("state", "/state", icd.StringJSONField{}),
				)
				ic := createICD(ctx, m)
				_, data := createIndexed(ctx, telem.Float64T)
				entry := icdEntry(
					"plant/status", ic, readMessage(m, binding(m, 0, data[0])),
				)
				Expect(configureErr(ctx, readConfig(ctx, dev, entry))).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(
						"cannot write string values to float64 channels",
					)),
				))
			},
		)

		It("Should reject a message that is not JSON", func(ctx SpecContext) {
			m := icd.Message{
				Key:     uuid.New(),
				Name:    "Frame",
				Payload: icd.Payload{Variant: icd.BinaryPayload{}},
			}
			ic := createICD(ctx, m)
			_, data := createIndexed(ctx, telem.Float64T)
			entry := icdEntry("plant/status", ic, map[string]any{
				"message": m.Key,
				"bindings": []any{
					map[string]any{"field": uuid.New(), "channel": data[0].Key()},
				},
			})
			Expect(configureErr(ctx, readConfig(ctx, dev, entry))).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("message Frame is not a JSON message")),
			))
		})
	})
})
