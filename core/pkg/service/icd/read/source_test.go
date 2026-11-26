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
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/read"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Source", func() {
	const (
		clock   channel.Key = 1
		tempCh  channel.Key = 2
		tick    channel.Key = 3
		levelCh channel.Key = 4
		flowCh  channel.Key = 5
		dutyCh  channel.Key = 6
	)
	var (
		channels = map[channel.Key]icdv0.Channel{
			clock: {
				Name: "clock", DataType: telem.TimestampT, Index: true, IndexKey: clock,
			},
			tempCh: {Name: "temp", DataType: telem.Float64T, IndexKey: clock},
			tick: {
				Name: "tick", DataType: telem.TimestampT, Index: true, IndexKey: tick,
			},
			levelCh: {Name: "level", DataType: telem.Float64T, IndexKey: tick},
			flowCh:  {Name: "flow", DataType: telem.Float64T},
			dutyCh:  {Name: "duty", DataType: telem.Float64T},
		}
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
		ic = icd.ICD{Key: uuid.New(), Name: "Plant", Messages: []icd.Message{mixed}}
		// mixedRead reads a field on each of two index channels, one of them stamped,
		// and two fields on virtual channels.
		mixedRead = indexed(atField, readOf(
			mixed,
			bind(atField, clock),
			bind(tempField, tempCh),
			bind(levelField, levelCh),
			bind(flowField, flowCh),
			bind(dutyField, dutyCh),
		))
		f64 = func(v float64) []byte { return telem.NewSeriesV(v).Data }
	)
	var resolver read.Resolver
	BeforeEach(func() {
		resolver = read.NewResolver(icdSvc, func(
			context.Context,
			gorp.Snapshot,
			[]channel.Key,
		) (map[channel.Key]icdv0.Channel, error) {
			return channels, nil
		})
	})
	// bindRead returns the Source of a read of rms, messages of ic, after Read accepts
	// it.
	bindRead := func(
		ctx context.Context,
		ic icd.ICD,
		rms ...icd.ReadMessage,
	) *read.Source {
		GinkgoHelper()
		rd := readAt(0, ic.Key, rms...)
		c := MustSucceed(resolver.NewChecker(
			ctx, db, []read.Read{rd}, nil, map[icd.Key]icd.ICD{ic.Key: ic},
		))
		Expect(c.Read(rd)).To(Succeed())
		return MustSucceed(c.Bind(rd))
	}
	newSource := func(ctx context.Context, rms ...icd.ReadMessage) *read.Source {
		GinkgoHelper()
		return bindRead(ctx, ic, rms...)
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
	twoReads := func(ctx context.Context, alarm *icd.JSONMatch) *read.Source {
		GinkgoHelper()
		reading := jsonMessage("Reading", atField, tempField)
		reading.Payload.Variant = icd.JSONPayload{
			Match: typeMatch("reading"), Fields: []icd.JSONField{atField, tempField},
		}
		alarming := jsonMessage("Alarm", levelField)
		alarming.Payload.Variant = icd.JSONPayload{
			Match: alarm, Fields: []icd.JSONField{levelField},
		}
		ic := icd.ICD{
			Key: uuid.New(), Name: "Plant", Messages: []icd.Message{reading, alarming},
		}
		return bindRead(
			ctx,
			ic,
			indexed(atField, readOf(
				reading, bind(atField, clock), bind(tempField, tempCh),
			)),
			readOf(alarming, bind(levelField, levelCh)),
		)
	}

	It("Should group fields by index channel, and each virtual field alone", func(
		ctx SpecContext,
	) {
		Expect(newSource(ctx, mixedRead).Groups()).To(Equal([]read.Group{
			{
				Message:   "Mixed",
				Index:     clock,
				Stamped:   true,
				Channels:  channel.Keys{tempCh},
				DataTypes: []telem.DataType{telem.Float64T},
			},
			{
				Message:   "Mixed",
				Index:     tick,
				Channels:  channel.Keys{levelCh},
				DataTypes: []telem.DataType{telem.Float64T},
			},
			{
				Message:   "Mixed",
				Channels:  channel.Keys{flowCh},
				DataTypes: []telem.DataType{telem.Float64T},
			},
			{
				Message:   "Mixed",
				Channels:  channel.Keys{dutyCh},
				DataTypes: []telem.DataType{telem.Float64T},
			},
		}))
	})

	It("Should give each group a sample, and stamp only the stamped group", func(
		ctx SpecContext,
	) {
		samples, ok := MustSucceed2(newSource(ctx, mixedRead).Decode(
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
		func(ctx SpecContext, doc string, groups []int, msg string) {
			samples, ok, err := newSource(ctx, mixedRead).Decode([]byte(doc))
			Expect(ok).To(BeTrue())
			Expect(err).To(MatchError(msg))
			Expect(groupsOf(samples)).To(Equal(groups))
		},
		Entry("a data field",
			`{"at":5,"temp":"hot","level":2.5,"flow":3.5,"duty":4.5}`,
			[]int{1, 2, 3},
			`field temp: invalid number "hot": validation error`,
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
			`field flow: invalid number "fast": validation error`,
		),
	)

	It("Should drop each group with a failed field, and name the first", func(
		ctx SpecContext,
	) {
		samples, ok, err := newSource(ctx, mixedRead).Decode(
			[]byte(`{"at":5,"temp":"hot","flow":3.5,"duty":4.5}`),
		)
		Expect(ok).To(BeTrue())
		Expect(
			err,
		).To(MatchError(`field temp: invalid number "hot": validation error`))
		Expect(groupsOf(samples)).To(Equal([]int{2, 3}))
	})

	It("Should return no samples for a document that is not JSON", func(
		ctx SpecContext,
	) {
		Expect(newSource(ctx, mixedRead).Decode([]byte("on"))).
			Error().To(MatchError(validate.ErrValidation))
	})

	It("Should return no samples and no error for a document that meets no rule",
		func(ctx SpecContext) {
			samples, ok := MustSucceed2(twoReads(ctx, typeMatch("alarm")).Decode(
				[]byte(`{"type":"other","level":2.5}`),
			))
			Expect(ok).To(BeFalse())
			Expect(samples).To(BeEmpty())
		},
	)

	It("Should return no samples for a document that meets two match rules", func(
		ctx SpecContext,
	) {
		present := &icd.JSONMatch{Variant: icd.PresentJSONMatch{
			JSONBaseMatch: icd.JSONBaseMatch{Pointer: "/level"},
		}}
		Expect(twoReads(ctx, present).Decode(
			[]byte(`{"type":"reading","at":5,"temp":1.5,"level":2.5}`),
		)).Error().To(MatchError(
			"document matches messages Reading and Alarm: validation error",
		))
	})

	It("Should hold only the samples of the last document", func(ctx SpecContext) {
		src := newSource(ctx, mixedRead)
		MustSucceed2(src.Decode(
			[]byte(`{"at":5,"temp":1.5,"level":2.5,"flow":3.5,"duty":4.5}`),
		))
		samples, _ := MustSucceed2(src.Decode(
			[]byte(`{"at":6,"temp":7.5,"level":8.5,"flow":9.5,"duty":10.5}`),
		))
		Expect(samples[0].Data).To(Equal([][]byte{f64(7.5)}))
		Expect(samples[3].Data).To(Equal([][]byte{f64(10.5)}))
	})

	It("Should give the groups of each read their own positions", func(
		ctx SpecContext,
	) {
		src := twoReads(ctx, typeMatch("alarm"))
		samples, _ := MustSucceed2(src.Decode([]byte(`{"type":"alarm","level":2.5}`)))
		Expect(samples).To(Equal([]read.Sample{
			{Group: 1, Data: [][]byte{f64(2.5)}},
		}))
		Expect(src.Groups()[1]).To(Equal(read.Group{
			Read:      1,
			Message:   "Alarm",
			Index:     tick,
			Channels:  channel.Keys{levelCh},
			DataTypes: []telem.DataType{telem.Float64T},
		}))
	})

	It("Should name the message of the read of each group", func(ctx SpecContext) {
		Expect(lo.Map(
			twoReads(ctx, typeMatch("alarm")).Groups(),
			func(g read.Group, _ int) string { return g.Message },
		)).To(Equal([]string{"Reading", "Alarm"}))
	})

	DescribeTable("Should give the keys of a group with its index channel first",
		func(g read.Group, keys channel.Keys) { Expect(g.Keys()).To(Equal(keys)) },
		Entry("a group on an index channel",
			read.Group{Index: clock, Channels: channel.Keys{tempCh}},
			channel.Keys{clock, tempCh},
		),
		Entry("a group that binds only the index field",
			read.Group{Index: clock, Stamped: true},
			channel.Keys{clock},
		),
		Entry("a group on a virtual channel",
			read.Group{Channels: channel.Keys{flowCh}},
			channel.Keys{flowCh},
		),
	)
})
