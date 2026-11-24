// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package join pairs the inputs of a calculation by sample time. The output takes the
// stamps and alignments of a primary input, and each other input holds its last value
// at or before each stamp, up to a hold limit set from its own sample interval.
package join

import (
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Config configures a Join.
type Config struct {
	// DataTypes holds the data type of each input, in input order. Each must have a
	// fixed density.
	DataTypes []telem.DataType
	// Primary is the index of the input whose stamps time the output.
	Primary int
	// From is the first stamp that gets an output. The samples before it only fill the
	// hold limits, so that a read can look back before its range.
	From telem.TimeStamp
	// Live makes the join judge an input that is late by arrival, as a live read cannot
	// wait for it. Each Next joins one frame of primary stamps. An input whose newest
	// stamp lies before the frame's first stamp holds its value for the whole frame
	// when that gap is within its hold limit.
	Live bool
}

// Run is a stretch of joined samples whose alignments touch.
type Run struct {
	// Values holds one series per input, in input order, with one sample per stamp of
	// Time.
	Values []telem.Series
	// Time holds the stamps of the primary input, with its alignments.
	Time telem.Series
}

// Output holds the samples that calls to Next join. Reset keeps its buffers, so an
// Output that is reset and reused allocates only when a run outgrows them or a gap
// starts a run.
type Output struct {
	// Runs holds the joined samples, a new run after each gap.
	Runs []Run
	// Dropped is the number of primary stamps from From on that got no output, because
	// an input had no sample at or before the stamp, or was past its hold limit.
	Dropped int
}

// Reset empties o and keeps its buffers.
func (o *Output) Reset() {
	o.Runs = o.Runs[:0]
	o.Dropped = 0
}

// Join pairs the samples of its inputs by sample time. It is not safe for concurrent
// use.
type Join struct {
	inputs  []*input
	primary int
	from    telem.TimeStamp
	live    bool
	// last is the alignment of the last primary sample that got an output, and joined
	// is false when no sample did or a dropped stamp came after it.
	last   telem.Alignment
	joined bool
}

type input struct {
	dataType telem.DataType
	density  int
	// data, stamps, and alignments hold the pushed samples that Next has not passed,
	// from first on. Only the primary keeps alignments.
	data       []byte
	stamps     []telem.TimeStamp
	alignments []telem.Alignment
	first      int
	// last is the last stamp pushed to the input, and pushed is false before the first
	// push.
	last   telem.TimeStamp
	pushed bool
	// held is the last sample at or before the last stamp that Next passed.
	held      []byte
	heldStamp telem.TimeStamp
	holds     bool
	intervals intervals
}

// New returns a Join of cfg.
func New(cfg Config) (*Join, error) {
	if len(cfg.DataTypes) == 0 {
		return nil, errors.Wrap(validate.ErrValidation, "join needs at least one input")
	}
	if cfg.Primary < 0 || cfg.Primary >= len(cfg.DataTypes) {
		return nil, errors.Wrapf(
			validate.ErrValidation,
			"primary input %d is not one of the %d inputs",
			cfg.Primary,
			len(cfg.DataTypes),
		)
	}
	j := &Join{
		inputs:  make([]*input, len(cfg.DataTypes)),
		primary: cfg.Primary,
		from:    cfg.From,
		live:    cfg.Live,
	}
	for i, dt := range cfg.DataTypes {
		if dt.IsVariable() || dt.Density() == 0 {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"input %d has data type %s, which has no fixed density",
				i,
				dt,
			)
		}
		density := int(dt.Density())
		j.inputs[i] = &input{
			dataType: dt,
			density:  density,
			held:     make([]byte, density),
		}
	}
	return j, nil
}

// Push adds the samples of input i, with their stamps in t. The stamps must rise
// strictly from the last stamp pushed to the input. The samples take the alignment of
// t, and Push ignores the alignment of data.
func (j *Join) Push(i int, data, t telem.Series) error {
	if i < 0 || i >= len(j.inputs) {
		return errors.Wrapf(validate.ErrValidation, "input %d does not exist", i)
	}
	in := j.inputs[i]
	switch {
	case data.DataType != in.dataType:
		return errors.Wrapf(
			validate.ErrValidation,
			"input %d has data type %s, not %s",
			i,
			in.dataType,
			data.DataType,
		)
	case t.DataType != telem.TimestampT:
		return errors.Wrapf(
			validate.ErrValidation,
			"stamps of input %d have data type %s, not timestamp",
			i,
			t.DataType,
		)
	case data.Len() != t.Len():
		return errors.Wrapf(
			validate.ErrValidation,
			"input %d has %d samples and %d stamps",
			i,
			data.Len(),
			t.Len(),
		)
	case i == j.primary && t.Multiple() != 1:
		return errors.Wrapf(
			validate.ErrValidation,
			"primary input %d steps its alignment by %d, not 1",
			i,
			t.Multiple(),
		)
	}
	stamps, alignments := len(in.stamps), len(in.alignments)
	last, pushed := in.last, in.pushed
	for k := range t.Len() {
		stamp := telem.TimeStamp(telem.ByteOrder.Uint64(t.Data[k*8:]))
		if pushed && stamp <= last {
			in.stamps, in.alignments = in.stamps[:stamps], in.alignments[:alignments]
			return errors.Wrapf(
				validate.ErrValidation,
				"stamp %s of input %d does not follow %s",
				stamp,
				i,
				last,
			)
		}
		last, pushed = stamp, true
		in.stamps = append(in.stamps, stamp)
		if i == j.primary {
			in.alignments = append(in.alignments, t.Alignment+telem.Alignment(k))
		}
	}
	in.data = append(in.data, data.Data...)
	in.last, in.pushed = last, pushed
	return nil
}

// Next appends to out a joined sample for each pushed primary stamp up to until. A live
// read passes the newest primary stamp, so no input is waited for. A historical read
// passes the earliest of the newest stamps of the inputs, so each input's value at a
// stamp is known.
func (j *Join) Next(until telem.TimeStamp, out *Output) {
	p := j.inputs[j.primary]
	var frame telem.TimeStamp
	if p.first < len(p.stamps) {
		frame = p.stamps[p.first]
	}
	for ; p.first < len(p.stamps) && p.stamps[p.first] <= until; p.first++ {
		stamp := p.stamps[p.first]
		j.pass(stamp)
		offset := p.first * p.density
		copy(p.held, p.data[offset:offset+p.density])
		if stamp < j.from {
			continue
		}
		if !j.holdsAll(stamp, frame) {
			out.Dropped++
			j.joined = false
			continue
		}
		j.emit(p.first, out)
	}
	for i, in := range j.inputs {
		if i != j.primary {
			in.bound()
		}
		in.compact()
	}
}

// pass moves the held sample of each other input than the primary to its last sample at
// or before stamp.
func (j *Join) pass(stamp telem.TimeStamp) {
	for i, in := range j.inputs {
		if i == j.primary {
			continue
		}
		for in.first < len(in.stamps) && in.stamps[in.first] <= stamp {
			in.step()
		}
	}
}

// step moves the held sample of in to its next pushed sample.
func (in *input) step() {
	next := in.stamps[in.first]
	if in.holds {
		in.intervals.add(telem.TimeSpan(next - in.heldStamp))
	}
	offset := in.first * in.density
	copy(in.held, in.data[offset:offset+in.density])
	in.heldStamp, in.holds = next, true
	in.first++
}

// most is the number of samples past the held one that an input other than the
// primary keeps.
const most = 1 << 16

// bound steps in past all but its last most samples, so a primary that stays silent
// leaves memory bounded. A later primary stamp before the held sample gets no output,
// as past the limit.
func (in *input) bound() {
	for len(in.stamps)-in.first > most {
		in.step()
	}
}

// holdsAll reports whether each other input than the primary holds a sample within its
// hold limit of stamp, in the frame of primary stamps that starts at frame.
func (j *Join) holdsAll(stamp, frame telem.TimeStamp) bool {
	for i, in := range j.inputs {
		if i == j.primary {
			continue
		}
		// A sample that bound stepped to can lie after stamp.
		if !in.holds || stamp < in.heldStamp {
			return false
		}
		limit := in.intervals.limit()
		if limit == 0 {
			continue
		}
		// A live input with no sample past the frame's start is judged at that start.
		at := stamp
		if j.live && in.first == len(in.stamps) && in.last < frame {
			at = frame
		}
		if telem.TimeSpan(at-in.heldStamp) > limit {
			return false
		}
	}
	return true
}

// emit appends the held samples of the inputs at the primary sample k to out, in a new
// run when the sample does not follow the last one that got an output.
func (j *Join) emit(k int, out *Output) {
	p := j.inputs[j.primary]
	alignment, stamp := p.alignments[k], p.stamps[k]
	if !j.joined || alignment != j.last+1 || len(out.Runs) == 0 {
		j.startRun(out, alignment, stamp)
	}
	run := &out.Runs[len(out.Runs)-1]
	for i, in := range j.inputs {
		run.Values[i].Data = append(run.Values[i].Data, in.held...)
		run.Values[i].TimeRange.End = after(stamp)
	}
	run.Time.Data = telem.ByteOrder.AppendUint64(run.Time.Data, uint64(stamp))
	run.Time.TimeRange.End = after(stamp)
	j.last, j.joined = alignment, true
}

// startRun appends a run that starts at alignment and stamp to out, and reuses the
// buffers of a run that out held before a reset.
func (j *Join) startRun(out *Output, alignment telem.Alignment, stamp telem.TimeStamp) {
	if len(out.Runs) < cap(out.Runs) {
		out.Runs = out.Runs[:len(out.Runs)+1]
	} else {
		out.Runs = append(out.Runs, Run{})
	}
	run := &out.Runs[len(out.Runs)-1]
	if len(run.Values) != len(j.inputs) {
		run.Values = make([]telem.Series, len(j.inputs))
	}
	tr := telem.TimeRange{Start: stamp, End: after(stamp)}
	for i, in := range j.inputs {
		run.Values[i] = telem.Series{
			DataType:  in.dataType,
			Data:      run.Values[i].Data[:0],
			Alignment: alignment,
			TimeRange: tr,
		}
	}
	run.Time = telem.Series{
		DataType:  telem.TimestampT,
		Data:      run.Time.Data[:0],
		Alignment: alignment,
		TimeRange: tr,
	}
}

// after returns the end of a time range whose last stamp is stamp.
func after(stamp telem.TimeStamp) telem.TimeStamp {
	if stamp == telem.TimeStampMax {
		return stamp
	}
	return stamp + 1
}

// compact drops the samples that Next passed, once they are half of the buffer.
func (in *input) compact() {
	if in.first == 0 || in.first*2 < len(in.stamps) {
		return
	}
	n := copy(in.stamps, in.stamps[in.first:])
	in.stamps = in.stamps[:n]
	in.data = in.data[:copy(in.data, in.data[in.first*in.density:])]
	if in.alignments != nil {
		in.alignments = in.alignments[:copy(in.alignments, in.alignments[in.first:])]
	}
	in.first = 0
}
