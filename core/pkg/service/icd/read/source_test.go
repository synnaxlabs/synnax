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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/read"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Source", func() {
	var (
		clock = channel.Channel{
			Leaseholder: 1, LocalKey: 1, LocalIndex: 1, IsIndex: true,
			DataType: telem.TimestampT,
		}
		tempCh = channel.Channel{
			Leaseholder: 1, LocalKey: 2, LocalIndex: 1, DataType: telem.Float64T,
		}
		tick = channel.Channel{
			Leaseholder: 1, LocalKey: 3, LocalIndex: 3, IsIndex: true,
			DataType: telem.TimestampT,
		}
		levelCh = channel.Channel{
			Leaseholder: 1, LocalKey: 4, LocalIndex: 3, DataType: telem.Float64T,
		}
		flowCh = channel.Channel{
			Leaseholder: 1, LocalKey: 5, Virtual: true, DataType: telem.Float64T,
		}
		dutyCh = channel.Channel{
			Leaseholder: 1, LocalKey: 6, Virtual: true, DataType: telem.Float64T,
		}
		channels = lo.KeyBy(
			[]channel.Channel{clock, tempCh, tick, levelCh, flowCh, dutyCh},
			func(c channel.Channel) channel.Key { return c.Key() },
		)
		atField = jsonField(
			"at", "/at", icd.TimestampJSONField{TimeFormat: payload.TimeFormatUnixSec},
		)
		tempField  = jsonField("temp", "/temp", icd.FloatJSONField{})
		levelField = jsonField("level", "/level", icd.FloatJSONField{})
		flowField  = jsonField("flow", "/flow", icd.FloatJSONField{})
		dutyField  = jsonField("duty", "/duty", icd.FloatJSONField{})
		mixed      = jsonMessage(
			"Mixed", atField, tempField, levelField, flowField, dutyField,
		)
		ic = icd.ICD{Name: "Plant", Messages: []icd.Message{mixed}}
		// mixedRead reads a field on each of two index channels, one of them stamped,
		// and two fields on virtual channels.
		mixedRead = indexed(atField, readOf(
			mixed,
			bind(atField, clock.Key()),
			bind(tempField, tempCh.Key()),
			bind(levelField, levelCh.Key()),
			bind(flowField, flowCh.Key()),
			bind(dutyField, dutyCh.Key()),
		))
		f64 = func(v float64) []byte { return telem.NewSeriesV(v).Data }
	)
	newSource := func(reads ...icd.ReadMessage) *read.Source {
		GinkgoHelper()
		return MustSucceed(read.NewSource(reads, ic, channels))
	}
	groupsOf := func(samples []read.Sample) []int {
		return lo.Map(samples, func(s read.Sample, _ int) int { return s.Group })
	}
	typeMatch := func(value string) *icd.JSONMatch {
		return &icd.JSONMatch{Variant: icd.StringJSONMatch{
			JSONBaseMatch: icd.JSONBaseMatch{Pointer: "/type"}, Value: value,
		}}
	}
	// twoReads returns a source of a Reading read whose /type is "reading", and an
	// Alarm read that meets alarm.
	twoReads := func(alarm *icd.JSONMatch) *read.Source {
		GinkgoHelper()
		reading := jsonMessage("Reading", atField, tempField)
		reading.Payload.Variant = icd.JSONPayload{
			Match: typeMatch("reading"), Fields: []icd.JSONField{atField, tempField},
		}
		alarming := jsonMessage("Alarm", levelField)
		alarming.Payload.Variant = icd.JSONPayload{
			Match: alarm, Fields: []icd.JSONField{levelField},
		}
		ic := icd.ICD{Name: "Plant", Messages: []icd.Message{reading, alarming}}
		return MustSucceed(read.NewSource([]icd.ReadMessage{
			indexed(atField, readOf(
				reading, bind(atField, clock.Key()), bind(tempField, tempCh.Key()),
			)),
			readOf(alarming, bind(levelField, levelCh.Key())),
		}, ic, channels))
	}

	It("Should group fields by index channel, and each virtual field alone", func() {
		Expect(newSource(mixedRead).Groups()).To(Equal([]read.Group{
			{
				Index:     clock.Key(),
				Stamped:   true,
				Channels:  channel.Keys{tempCh.Key()},
				DataTypes: []telem.DataType{telem.Float64T},
			},
			{
				Index:     tick.Key(),
				Channels:  channel.Keys{levelCh.Key()},
				DataTypes: []telem.DataType{telem.Float64T},
			},
			{
				Channels:  channel.Keys{flowCh.Key()},
				DataTypes: []telem.DataType{telem.Float64T},
			},
			{
				Channels:  channel.Keys{dutyCh.Key()},
				DataTypes: []telem.DataType{telem.Float64T},
			},
		}))
	})

	It("Should give each group a sample, and stamp only the stamped group", func() {
		samples, ok := MustSucceed2(newSource(mixedRead).Decode(
			[]byte(`{"at":5,"temp":1.5,"level":2.5,"flow":3.5,"duty":4.5}`),
		))
		Expect(ok).To(BeTrue())
		Expect(samples).To(Equal([]read.Sample{
			{Group: 0, Stamp: 5 * telem.SecondTS, Data: [][]byte{f64(1.5)}},
			{Group: 1, Data: [][]byte{f64(2.5)}},
			{Group: 2, Data: [][]byte{f64(3.5)}},
			{Group: 3, Data: [][]byte{f64(4.5)}},
		}))
	})

	DescribeTable("Should drop only the group of a field that does not decode",
		func(doc string, groups []int, msg string) {
			samples, ok, err := newSource(mixedRead).Decode([]byte(doc))
			Expect(ok).To(BeTrue())
			Expect(err).To(MatchError(msg))
			Expect(groupsOf(samples)).To(Equal(groups))
		},
		Entry("a data field",
			`{"at":5,"temp":"hot","level":2.5,"flow":3.5,"duty":4.5}`,
			[]int{1, 2, 3},
			`field temp: unknown enum label "hot": validation error`,
		),
		Entry("the index field",
			`{"at":"noon","temp":1.5,"level":2.5,"flow":3.5,"duty":4.5}`,
			[]int{1, 2, 3},
			`field at: invalid number "noon": validation error`,
		),
		Entry("a field with no value and no default",
			`{"at":5,"temp":1.5,"flow":3.5,"duty":4.5}`,
			[]int{0, 2, 3},
			"field level: no value at /level: validation error",
		),
		Entry("a field on a virtual channel",
			`{"at":5,"temp":1.5,"level":2.5,"flow":"fast","duty":4.5}`,
			[]int{0, 1, 3},
			`field flow: unknown enum label "fast": validation error`,
		),
	)

	It("Should drop each group with a failed field, and name the first", func() {
		samples, ok, err := newSource(mixedRead).Decode(
			[]byte(`{"at":5,"temp":"hot","flow":3.5,"duty":4.5}`),
		)
		Expect(ok).To(BeTrue())
		Expect(
			err,
		).To(MatchError(`field temp: unknown enum label "hot": validation error`))
		Expect(groupsOf(samples)).To(Equal([]int{2, 3}))
	})

	It("Should return no samples for a document that is not JSON", func() {
		Expect(newSource(mixedRead).Decode([]byte("on"))).
			Error().To(MatchError(validate.ErrValidation))
	})

	It("Should return no samples and no error for a document that meets no rule",
		func() {
			samples, ok := MustSucceed2(twoReads(typeMatch("alarm")).Decode(
				[]byte(`{"type":"other","level":2.5}`),
			))
			Expect(ok).To(BeFalse())
			Expect(samples).To(BeEmpty())
		},
	)

	It("Should return no samples for a document that meets two match rules", func() {
		present := &icd.JSONMatch{Variant: icd.PresentJSONMatch{
			JSONBaseMatch: icd.JSONBaseMatch{Pointer: "/level"},
		}}
		Expect(twoReads(present).Decode(
			[]byte(`{"type":"reading","at":5,"temp":1.5,"level":2.5}`),
		)).Error().To(MatchError(
			"document matches messages Reading and Alarm: validation error",
		))
	})

	It("Should hold only the samples of the last document", func() {
		src := newSource(mixedRead)
		MustSucceed2(src.Decode(
			[]byte(`{"at":5,"temp":1.5,"level":2.5,"flow":3.5,"duty":4.5}`),
		))
		samples, _ := MustSucceed2(src.Decode(
			[]byte(`{"at":6,"temp":7.5,"level":8.5,"flow":9.5,"duty":10.5}`),
		))
		Expect(samples[0].Data).To(Equal([][]byte{f64(7.5)}))
		Expect(samples[3].Data).To(Equal([][]byte{f64(10.5)}))
	})

	It("Should give the groups of each read their own positions", func() {
		src := twoReads(typeMatch("alarm"))
		samples, _ := MustSucceed2(src.Decode([]byte(`{"type":"alarm","level":2.5}`)))
		Expect(samples).To(Equal([]read.Sample{
			{Group: 1, Data: [][]byte{f64(2.5)}},
		}))
		Expect(src.Groups()[1]).To(Equal(read.Group{
			Read:      1,
			Index:     tick.Key(),
			Channels:  channel.Keys{levelCh.Key()},
			DataTypes: []telem.DataType{telem.Float64T},
		}))
	})

	DescribeTable("Should refuse reads that it cannot decode",
		func(msg string, reads ...icd.ReadMessage) {
			Expect(read.NewSource(reads, ic, channels)).Error().To(MatchError(msg))
		},
		Entry("a read that binds no fields",
			"message Mixed binds no fields: validation error",
			readOf(mixed),
		),
		Entry("a channel that does not exist",
			"message Mixed: channel 99 does not exist: validation error",
			readOf(mixed, bind(tempField, 99)),
		),
		Entry("two reads that the decoder cannot tell apart",
			"messages Mixed and Mixed both have no match rule: validation error",
			readOf(mixed, bind(tempField, tempCh.Key())),
			readOf(mixed, bind(levelField, levelCh.Key())),
		),
	)
})
