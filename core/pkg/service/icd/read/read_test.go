// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package read_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/read"
	"github.com/synnaxlabs/x/content"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

func jsonField(name, pointer string, v icd.JSONFieldVariant) icd.JSONField {
	f := icd.JSONField{Variant: v}
	f.SetBase(
		icd.JSONBaseField{Key: uuid.New(), Name: name, Scale: 1, Pointer: pointer},
	)
	return f
}

func jsonMessage(name string, fields ...icd.JSONField) icd.Message {
	return icd.Message{
		Key:     uuid.New(),
		Name:    name,
		Payload: icd.Payload{Variant: icd.JSONPayload{Fields: fields}},
	}
}

var (
	ts = jsonField(
		"ts",
		"/ts",
		icd.TimestampJSONField{TimeFormat: payload.TimeFormatUnixSec},
	)
	temp  = jsonField("temp", "/temp", icd.FloatJSONField{})
	state = jsonField("state", "/state", icd.StringJSONField{})
	raw   = jsonField("raw", "", icd.BytesJSONField{})
	stamp = telem.SecondTS
	// defaulted is a timestamp field that takes a default.
	defaulted = jsonField("ts", "/ts", icd.TimestampJSONField{
		TimeFormat: payload.TimeFormatUnixSec,
		Default:    &stamp,
	})
	status  = jsonMessage("Status", ts, temp, state, raw)
	stamped = jsonMessage("Stamped", defaulted)
	frame   = icd.Message{
		Key:     uuid.New(),
		Name:    "Frame",
		Payload: icd.Payload{Variant: icd.BinaryPayload{}},
	}
	plant = icd.ICD{Name: "Plant", Messages: []icd.Message{status, stamped, frame}}
)

const (
	timeKey  channel.Key = 1
	valueKey channel.Key = 2
	stampKey channel.Key = 3
)

func newChecker() read.Checker {
	return read.NewChecker(map[channel.Key]read.Channel{
		timeKey:  {Name: "time", DataType: telem.TimestampT, Index: true},
		valueKey: {Name: "value", DataType: telem.Float64T},
		stampKey: {Name: "stamp", DataType: telem.TimestampT},
	})
}

func bind(f icd.JSONField, key channel.Key) icd.ReadBinding {
	return icd.ReadBinding{Field: f.Base().Key, Channel: key}
}

func readOf(m icd.Message, bindings ...icd.ReadBinding) icd.ReadMessage {
	return icd.ReadMessage{Message: m.Key, Bindings: bindings}
}

func indexed(f icd.JSONField, rm icd.ReadMessage) icd.ReadMessage {
	key := f.Base().Key
	rm.Index = &key
	return rm
}

var _ = Describe("Read", func() {
	Describe("Checker", func() {
		Describe("Message", func() {
			It("Should accept a read whose fields suit their channels", func() {
				rm := indexed(
					ts,
					readOf(status, bind(ts, timeKey), bind(temp, valueKey)),
				)
				Expect(newChecker().Message(rm, plant, []string{"reads", "0"})).
					To(Succeed())
			})

			It("Should accept a binding to a channel that the checker does not hold",
				func() {
					rm := readOf(status, bind(state, 99))
					Expect(newChecker().Message(rm, plant, []string{"reads", "0"})).
						To(Succeed())
				},
			)

			It("Should accept a data type that the field converts to", func() {
				rm := readOf(status, icd.ReadBinding{
					Field: temp.Base().Key, DataType: telem.Float32T,
				})
				Expect(newChecker().Message(rm, plant, []string{"reads", "0"})).
					To(Succeed())
			})

			DescribeTable("Should reject a read with the path of the problem",
				func(rm icd.ReadMessage, path, msg string) {
					err := newChecker().Message(rm, plant, []string{"reads", "0"})
					Expect(err).To(MatchError(validate.ErrValidation))
					Expect(err).To(MatchError(HavePrefix(path + ": ")))
					Expect(err).To(MatchError(ContainSubstring(msg)))
				},
				Entry("a message that the ICD lacks",
					icd.ReadMessage{Message: uuid.New()},
					"reads.0.message", "does not exist in ICD Plant",
				),
				Entry("a message that is not JSON",
					readOf(frame),
					"reads.0.message", "message Frame is not a JSON message",
				),
				Entry("a field that the message lacks",
					readOf(status, icd.ReadBinding{Field: uuid.New()}),
					"reads.0.bindings.0.field", "does not exist in message Status",
				),
				Entry(
					"a bytes field at the root",
					readOf(status, icd.ReadBinding{Field: raw.Base().Key}),
					"reads.0.bindings.0.field",
					"bytes field raw requires a JSON pointer",
				),
				Entry("a data type that the field kind does not suit",
					readOf(status, icd.ReadBinding{
						Field: state.Base().Key, DataType: telem.Float64T,
					}),
					"reads.0.bindings.0.data_type",
					"field state of kind string cannot read into data type float64",
				),
				Entry("a channel that the field kind does not convert to",
					readOf(status, bind(state, valueKey)),
					"reads.0.bindings.0.channel",
					"field state of kind string cannot read into channel value of "+
						"data type float64",
				),
				Entry("an index field that is not a timestamp",
					indexed(temp, readOf(status, bind(temp, valueKey))),
					"reads.0.index", "the index field temp must be a timestamp field",
				),
				Entry("an index field with a default",
					indexed(defaulted, readOf(stamped, bind(defaulted, timeKey))),
					"reads.0.index", "the index field ts cannot take a default",
				),
				Entry("an index field with no binding",
					indexed(ts, readOf(status, bind(temp, valueKey))),
					"reads.0.index", "the index field ts has no binding",
				),
				Entry("an index that names no field",
					indexed(jsonField("gone", "/gone", icd.TimestampJSONField{}),
						readOf(status, bind(temp, valueKey))),
					"reads.0.index", "does not exist in message Status",
				),
				Entry("an index field on a data channel",
					indexed(ts, readOf(status, bind(ts, stampKey))),
					"reads.0.bindings.0.channel",
					"the index field ts must write to an index channel",
				),
				Entry("a data field on an index channel",
					readOf(status, bind(ts, timeKey)),
					"reads.0.bindings.0.channel",
					"only the index field can write to index channel time",
				),
			)

			It("Should reject a channel that an earlier read writes", func() {
				c := newChecker()
				rm := readOf(status, bind(temp, valueKey))
				Expect(c.Message(rm, plant, []string{"reads", "0"})).To(Succeed())
				Expect(
					c.Message(rm, plant, []string{"reads", "1"}),
				).Error().
					To(MatchError(
						"reads.1.bindings.0.channel: channel 2 is also written by " +
							"reads.0.bindings.0.channel",
					))
			})
		})

		Describe("Claim", func() {
			It("Should take a channel that nothing writes", func() {
				Expect(newChecker().Claim(valueKey, []string{"tag"})).To(Succeed())
			})

			It("Should reject a channel that another binding or tag writes", func() {
				c := newChecker()
				Expect(c.Claim(valueKey, []string{"a", "channel"})).To(Succeed())
				err := c.Claim(valueKey, []string{"b", "channel"})
				Expect(err).To(MatchError(validate.ErrValidation))
				Expect(err).To(MatchError(
					"b.channel: channel 2 is also written by a.channel",
				))
			})
		})
	})

	Describe("Hash", func() {
		sum := func(ic icd.ICD, rms ...icd.ReadMessage) string {
			GinkgoHelper()
			h := content.NewHash()
			Expect(read.Hash(&h, ic, rms)).To(Succeed())
			return h.Sum()
		}

		It("Should give the same sum for the same payloads", func() {
			rm := readOf(status, bind(temp, valueKey))
			Expect(sum(plant, rm)).To(Equal(sum(plant, rm)))
		})

		It("Should change when the payload of a read message changes", func() {
			rm := readOf(status, bind(temp, valueKey))
			moved := icd.JSONField{Variant: icd.FloatJSONField{}}
			moved.SetBase(icd.JSONBaseField{
				Key: temp.Base().Key, Name: "temp", Scale: 1, Pointer: "/temperature",
			})
			edited := plant
			edited.Messages = []icd.Message{
				{
					Key:  status.Key,
					Name: status.Name,
					Payload: icd.Payload{
						Variant: icd.JSONPayload{Fields: []icd.JSONField{moved}},
					},
				},
			}
			Expect(sum(edited, rm)).ToNot(Equal(sum(plant, rm)))
		})

		It("Should mark a read of a message that the ICD lacks", func() {
			rm := icd.ReadMessage{Message: uuid.New()}
			Expect(sum(plant, rm)).ToNot(Equal(sum(plant)))
		})
	})
})
