// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json_test

import (
	"encoding/json/jsontext"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

type sampleSlots struct {
	encoders []*xjson.Encoder
	samples  [][]byte
}

func (s *sampleSlots) AppendSlot(dst []byte, field int) ([]byte, error) {
	return s.encoders[field].Append(dst, s.samples[field])
}

var _ = Describe("Allocation", func() {
	It("Should read a document into samples without allocating", func() {
		doc := []byte(`{"ts":"2026-01-02T03:04:05Z","v":{"a":1.5,"b":"x\ny"},` +
			`"s":"ON","n":[1,{"m":-2}],"b":"AP8\/"}`)
		e := xjson.NewExtractor(
			pointers("/ts", "/v/a", "/v/b", "/s", "/n/1/m", "/b"),
		)
		decoders := []*xjson.Decoder{
			MustSucceed(xjson.NewDecoder(xjson.Field{
				DataType: telem.TimestampT, TimeFormat: xjson.ISO8601,
			})),
			MustSucceed(xjson.NewDecoder(xjson.Field{DataType: telem.Float64T})),
			MustSucceed(xjson.NewDecoder(xjson.Field{DataType: telem.StringT})),
			MustSucceed(xjson.NewDecoder(xjson.Field{
				DataType: telem.Uint8T, Enums: onOff,
			})),
			MustSucceed(xjson.NewDecoder(xjson.Field{DataType: telem.Int64T})),
			MustSucceed(xjson.NewDecoder(xjson.Field{DataType: telem.BytesT})),
		}
		values := make([]jsontext.Value, len(decoders))
		dst := make([]byte, 0, 1024)
		var err error
		read := func() {
			if err = e.Extract(doc, values); err != nil {
				return
			}
			dst = dst[:0]
			for i, d := range decoders {
				if dst, err = d.Append(dst, values[i]); err != nil {
					return
				}
			}
		}
		read()
		Expect(err).ToNot(HaveOccurred())
		Expect(testing.AllocsPerRun(100, read)).To(BeZero())
		Expect(err).ToNot(HaveOccurred())
	})

	It("Should compact a JSON value without allocating", func() {
		if raceEnabled {
			Skip("jsontext reuses its encoder through a sync.Pool")
		}
		d := MustSucceed(xjson.NewDecoder(xjson.Field{DataType: telem.JSONT}))
		value := jsontext.Value(`{ "a": 1.5, "b": "x\ny" }`)
		dst := make([]byte, 0, 1024)
		var err error
		compact := func() { dst, err = d.Append(dst[:0], value) }
		compact()
		Expect(err).ToNot(HaveOccurred())
		Expect(testing.AllocsPerRun(100, compact)).To(BeZero())
		Expect(err).ToNot(HaveOccurred())
	})

	It("Should render a document from samples without allocating", func() {
		fields := []xjson.Field{
			{DataType: telem.TimestampT, TimeFormat: xjson.UnixSecond},
			{DataType: telem.Float64T},
			{DataType: telem.StringT, Type: xjson.String},
			{DataType: telem.Uint8T, Type: xjson.String, Enums: onOff},
			{DataType: telem.BytesT, Type: xjson.String},
		}
		slots := sampleSlots{
			samples: [][]byte{
				sample(telem.TimeStamp(1767323045123456789)),
				sample(0.1),
				[]byte("a\"b"),
				sample[uint8](1),
				{0x00, 0xff, 0x3f},
			},
		}
		for _, f := range fields {
			slots.encoders = append(slots.encoders, MustSucceed(xjson.NewEncoder(f)))
		}
		t := MustSucceed(xjson.NewTemplate([]xjson.TemplateField{
			field("/ts", ""), field("/v/a", ""), field("/v/b", ""), field("/s", ""),
			field("/b", ""), field("/meta", `{"source":"synnax"}`),
		}))
		dst := make([]byte, 0, 1024)
		var err error
		render := func() { dst, err = t.Append(dst[:0], &slots) }
		render()
		Expect(err).ToNot(HaveOccurred())
		Expect(string(dst)).To(Equal(`{"ts":1767323045.123456789,"v":{"a":0.1,` +
			`"b":"a\"b"},"s":"ON","b":"AP8/","meta":{"source":"synnax"}}`))
		Expect(testing.AllocsPerRun(100, render)).To(BeZero())
	})
})
