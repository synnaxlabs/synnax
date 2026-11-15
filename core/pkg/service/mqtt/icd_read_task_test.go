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
	base := icd.JSONBaseField{
		BaseField: icd.BaseField{Key: uuid.New(), Name: name, Scale: 1},
		Pointer:   pointer,
	}
	switch f := v.(type) {
	case icd.FloatJSONField:
		f.JSONBaseField = base
		return icd.JSONField{Variant: f}
	case icd.StringJSONField:
		f.JSONBaseField = base
		return icd.JSONField{Variant: f}
	case icd.TimestampJSONField:
		f.JSONBaseField = base
		return icd.JSONField{Variant: f}
	}
	Fail("unknown JSON field variant")
	return icd.JSONField{}
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

func fieldKey(m icd.Message, i int) icd.FieldKey {
	return m.Payload.Variant.(icd.JSONPayload).Fields[i].Base().Key
}

// binding returns the config that writes field i of m to ch.
func binding(m icd.Message, i int, ch channel.Channel) map[string]any {
	return map[string]any{"field": fieldKey(m, i), "channel": ch.Key()}
}

func icdEntry(topic string, ic icd.ICD, messages ...map[string]any) map[string]any {
	list := make([]any, len(messages))
	for i, m := range messages {
		list[i] = m
	}
	return map[string]any{
		"key":      topic,
		"topic":    topic,
		"icd":      ic.Key,
		"messages": list,
	}
}

func readMessage(m icd.Message, bindings ...map[string]any) map[string]any {
	list := make([]any, len(bindings))
	for i, b := range bindings {
		list[i] = b
	}
	return map[string]any{"message": m.Key, "bindings": list}
}

var _ = Describe("ICD read task", func() {
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
		t := newTask(ctx, rackKey, mqtt.ICDReadTaskType, cfg)
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
			t := start(ctx, readConfig(dev, icdEntry(
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
				&icd.JSONMatch{Pointer: "/type", Value: "s"},
				jsonField("ts", "/ts", icd.TimestampJSONField{
					TimeFormat: payload.TimeFormatUnixSec,
				}),
				jsonField("temp", "/temp", icd.FloatJSONField{}),
			)
			event := jsonMessage("Event", &icd.JSONMatch{Pointer: "/type", Value: "e"},
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
				readConfig(dev, icdEntry("plant/all", ic, statusRead, eventRead)),
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

	It("Should use a field's default when a document has no value",
		func(ctx SpecContext) {
			fallback := payload.Float(-1)
			m := jsonMessage("Status", nil,
				jsonField("temp", "/temp", icd.FloatJSONField{Default: &fallback}),
			)
			ic := createICD(ctx, m)
			_, data := createIndexed(ctx, telem.Float64T)
			start(ctx, readConfig(dev, icdEntry(
				"plant/status", ic, readMessage(m, binding(m, 0, data[0])),
			)))
			broker.publish("plant/status", `{}`, false)
			Eventually(func() int64 { return stored(ctx, data[0]).Len() }).
				Should(BeEquivalentTo(1))
			Expect(stored(ctx, data[0])).
				To(telem.MatchSeriesData(telem.NewSeriesV(-1.0)))
		},
	)

	Describe("Configure", func() {
		configureErr := func(ctx context.Context, cfg msgpack.EncodedJSON) error {
			t := newTask(ctx, rackKey, mqtt.ICDReadTaskType, cfg)
			_, err := factory.ConfigureTask(ctx, t, "configure")
			return err
		}

		It("Should reject an ICD that does not exist", func(ctx SpecContext) {
			m := jsonMessage("Status", nil, jsonField("t", "/t", icd.FloatJSONField{}))
			_, data := createIndexed(ctx, telem.Float64T)
			entry := icdEntry(
				"plant/status", icd.ICD{Key: uuid.New()},
				readMessage(m, binding(m, 0, data[0])),
			)
			Expect(configureErr(ctx, readConfig(dev, entry))).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("does not exist")),
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
				Expect(configureErr(ctx, readConfig(dev, entry))).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(
						"a string field cannot write to a float64 channel",
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
			Expect(configureErr(ctx, readConfig(dev, entry))).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("message Frame is not a JSON message")),
			))
		})
	})
})
