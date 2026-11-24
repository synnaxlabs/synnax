// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package write_test

import (
	"math"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/write"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

const xKey, yKey channel.Key = 10, 11

var _ = Describe("Documents", func() {
	channels := map[channel.Key]channel.Channel{
		xKey: {DataType: telem.Float64T},
		yKey: {DataType: telem.Float64T},
	}
	var skips *driver.Skips
	BeforeEach(func() { skips = driver.NewSkips("command", driver.SkipHold) })

	// point returns the Documents of a message that sends x and y, where y has the
	// variant yField.
	point := func(yField icd.JSONFieldVariant) *write.Documents {
		GinkgoHelper()
		x, y := jsonField("x", icd.FloatJSONField{}), jsonField("y", yField)
		m := jsonMessage("Point", x, y)
		ic := icd.ICD{Name: "Plant", Messages: []icd.Message{m}}
		return MustSucceed(write.NewDocuments(
			writeOf(
				m,
				channelOf(x.Base().Key, xKey, nil),
				channelOf(y.Base().Key, yKey, nil),
			),
			ic,
			channels,
			"target plant/point",
		))
	}

	// both returns a frame of commands to x and y.
	both := func(xs, ys telem.Series) frame.Frame {
		return frame.NewMulti([]channel.Key{xKey, yKey}, []telem.Series{xs, ys})
	}

	// from returns the documents that d builds from fr, as text.
	from := func(d *write.Documents, fr frame.Frame) []string {
		var docs []string
		for _, doc := range d.From(fr, skips) {
			docs = append(docs, string(doc))
		}
		return docs
	}

	It("Should build one document from the commands of one frame", func() {
		d := point(icd.FloatJSONField{})
		Expect(from(d, both(telem.NewSeriesV(1.0), telem.NewSeriesV(2.0)))).
			To(HaveExactElements(`{"x":1,"y":2}`))
	})

	It(
		"Should send the last command of a channel with fewer commands in a frame",
		func() {
			d := point(icd.FloatJSONField{})
			Expect(from(d, both(telem.NewSeriesV(1.0, 3.0), telem.NewSeriesV(2.0)))).
				To(HaveExactElements(`{"x":1,"y":2}`, `{"x":3,"y":2}`))
		},
	)

	It("Should send the last command of a channel with no new one", func() {
		d := point(icd.FloatJSONField{})
		from(d, both(telem.NewSeriesV(1.0), telem.NewSeriesV(2.0)))
		Expect(from(d, frame.NewUnary(xKey, telem.NewSeriesV(3.0)))).
			To(HaveExactElements(`{"x":3,"y":2}`))
	})

	It("Should send the default of a field with no command yet", func() {
		d := point(icd.FloatJSONField{Default: new(payload.Float(5))})
		Expect(from(d, frame.NewUnary(xKey, telem.NewSeriesV(1.0)))).
			To(HaveExactElements(`{"x":1,"y":5}`))
	})

	It("Should skip each document that would send a NaN default", func() {
		d := point(icd.FloatJSONField{Default: new(payload.Float(math.NaN()))})
		Expect(from(d, frame.NewUnary(xKey, telem.NewSeriesV(1.0)))).To(BeEmpty())
		w := skips.Warning(time.Now())
		Expect(w).ToNot(BeNil())
		Expect(w.Description).To(Equal("Command for target plant/point: message " +
			"Point: field y: default: NaN is not a JSON number: validation error"))
		Expect(from(d, frame.NewUnary(yKey, telem.NewSeriesV(2.0)))).
			To(HaveExactElements(`{"x":1,"y":2}`))
	})

	It("Should send an empty command instead of the default", func() {
		x, mode := jsonField("x", icd.FloatJSONField{}), jsonField(
			"mode", icd.StringJSONField{Default: new("auto")},
		)
		m := jsonMessage("Mode", x, mode)
		const modeKey channel.Key = 12
		d := MustSucceed(write.NewDocuments(
			writeOf(
				m,
				channelOf(x.Base().Key, xKey, nil),
				channelOf(mode.Base().Key, modeKey, nil),
			),
			icd.ICD{Name: "Plant", Messages: []icd.Message{m}},
			map[channel.Key]channel.Channel{
				xKey:    {DataType: telem.Float64T},
				modeKey: {DataType: telem.StringT},
			},
			"target plant/mode",
		))
		Expect(from(d, frame.NewMulti(
			[]channel.Key{xKey, modeKey},
			[]telem.Series{telem.NewSeriesV(1.0), telem.NewSeriesV("")},
		))).To(HaveExactElements(`{"x":1,"mode":""}`))
	})

	It("Should build nothing until each field with no default has a command", func() {
		d := point(icd.FloatJSONField{})
		Expect(from(d, frame.NewUnary(xKey, telem.NewSeriesV(1.0)))).To(BeEmpty())
		Expect(skips.Warning(time.Now())).To(BeNil())
		Expect(from(d, frame.NewUnary(yKey, telem.NewSeriesV(2.0)))).
			To(HaveExactElements(`{"x":1,"y":2}`))
	})

	It("Should forget every command on a reset", func() {
		d := point(icd.FloatJSONField{})
		from(d, both(telem.NewSeriesV(1.0), telem.NewSeriesV(2.0)))
		d.Reset()
		Expect(from(d, frame.NewUnary(xKey, telem.NewSeriesV(3.0)))).To(BeEmpty())
		Expect(from(d, frame.NewUnary(yKey, telem.NewSeriesV(4.0)))).
			To(HaveExactElements(`{"x":3,"y":4}`))
	})

	It("Should keep the last good command of a field after one it cannot hold", func() {
		d := point(icd.FloatJSONField{})
		from(d, both(telem.NewSeriesV(1.0), telem.NewSeriesV(2.0)))
		Expect(from(d, frame.NewUnary(xKey, telem.NewSeriesV(math.NaN())))).
			To(BeEmpty())
		Expect(from(d, frame.NewUnary(yKey, telem.NewSeriesV(4.0)))).
			To(HaveExactElements(`{"x":1,"y":4}`))
	})

	It("Should drop a command it cannot hold while another field waits", func() {
		d := point(icd.FloatJSONField{})
		Expect(from(d, frame.NewUnary(xKey, telem.NewSeriesV(1.0, math.NaN())))).
			To(BeEmpty())
		Expect(from(d, frame.NewUnary(yKey, telem.NewSeriesV(2.0)))).
			To(HaveExactElements(`{"x":1,"y":2}`))
	})

	It("Should give each document the row of the commands that it holds", func() {
		d := point(icd.FloatJSONField{})
		var rows []int
		for row := range d.From(both(
			telem.NewSeriesV(math.NaN(), 3.0, 5.0), telem.NewSeriesV(2.0),
		), skips) {
			rows = append(rows, row)
		}
		Expect(rows).To(HaveExactElements(1, 2))
	})

	It("Should send the good commands of a frame that holds a bad one", func() {
		d := point(icd.FloatJSONField{})
		from(d, both(telem.NewSeriesV(1.0), telem.NewSeriesV(2.0)))
		Expect(from(d, both(telem.NewSeriesV(math.NaN()), telem.NewSeriesV(4.0)))).
			To(HaveExactElements(`{"x":1,"y":4}`))
	})

	It("Should build no document for a frame that holds only bad commands", func() {
		d := point(icd.FloatJSONField{})
		from(d, both(telem.NewSeriesV(1.0), telem.NewSeriesV(2.0)))
		Expect(from(d, frame.NewUnary(xKey, telem.NewSeriesV(math.NaN())))).
			To(BeEmpty())
	})

	It("Should count each command as read and each dropped one as skipped", func() {
		d := point(icd.FloatJSONField{})
		from(d, both(telem.NewSeriesV(math.NaN()), telem.NewSeriesV(4.0)))
		w := skips.Warning(time.Now())
		Expect(w).ToNot(BeNil())
		Expect(w.Message).To(Equal("Skipped 1 of 2 commands"))
		Expect(w.Description).To(HavePrefix("Command for target plant/point: "))
	})

	It("Should return the command channel of each channel binding in order", func() {
		Expect(point(icd.FloatJSONField{}).Channels()).
			To(Equal(channel.Keys{xKey, yKey}))
	})

	It("Should return a validation error for a command channel that does not exist",
		func() {
			x := jsonField("x", icd.FloatJSONField{})
			m := jsonMessage("Point", x)
			ic := icd.ICD{Name: "Plant", Messages: []icd.Message{m}}
			Expect(write.NewDocuments(
				writeOf(m, channelOf(x.Base().Key, 99, nil)), ic, channels, "target",
			)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("channel 99 does not exist")),
			))
		},
	)
})

var _ = Describe("ChannelBindings", func() {
	It("Should return the channel bindings of a message in binding order", func() {
		Expect(write.ChannelBindings(writeOf(
			setpoint,
			channelOf(temp.Base().Key, xKey, nil),
			fixed(mode, icd.StringWriteValue{Value: "manual"}),
			channelOf(limit.Base().Key, yKey, nil),
		))).To(HaveExactElements(
			SatisfyAll(HaveField("Field", temp.Base().Key), HaveField("Channel", xKey)),
			SatisfyAll(
				HaveField("Field", limit.Base().Key),
				HaveField("Channel", yKey),
			),
		))
	})

	It("Should return no bindings for a message that binds only values", func() {
		Expect(write.ChannelBindings(
			writeOf(setpoint, fixed(mode, icd.StringWriteValue{Value: "manual"})),
		)).To(BeEmpty())
	})
})
