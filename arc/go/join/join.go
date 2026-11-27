// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package join pairs the inputs of a calculation by sample time. The output takes the
// stamps and alignments of a primary input. Each other input holds its last sample at
// or before each stamp, for at most its hold limit: 3 times the median of its last 16
// intervals. An input with fewer than 2 samples has no limit, and so does one with no
// steady rate, where fewer than 3 in 4 of those intervals lie within a factor of 2 of
// their median.
//
// A live join waits for no input, so it judges an input with no sample past a stamp by
// arrival. The input's sample stays valid while its last frame arrived within a window
// before the primary's last frame: the larger of its hold limit and 3 times the median
// of its last 16 frame gaps, or of the primary's until the input has one. Pushes that
// arrive at one time are one frame. An input with no steady rate holds with no window.
package join

import (
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Input configures one input of a Join.
type Input struct {
	// DataType is the data type of the input's samples. It must have a fixed density.
	//
	// [REQUIRED]
	DataType telem.DataType
	// ArrivalTimed marks an input whose stamps do not time its samples, as for a
	// channel with no index. It holds its newest pushed sample within its window, and
	// Push ignores its stamps. Only a live join takes one, and never as its primary.
	//
	// [OPTIONAL] - Defaults to false.
	ArrivalTimed bool
}

// Config configures a Join.
type Config struct {
	// Inputs holds the inputs, in input order.
	//
	// [REQUIRED]
	Inputs []Input
	// Primary is the index of the input whose stamps time the output.
	//
	// [OPTIONAL] - Defaults to 0.
	Primary int
	// From is the first stamp that gets an output. The samples before it only fill the
	// hold limits and the held samples.
	//
	// [OPTIONAL] - Defaults to 0.
	From telem.TimeStamp
	// Live makes the join judge late inputs by arrival. A live join drops the oldest
	// samples of an input past 65,536, so a silent primary leaves memory bounded.
	//
	// [OPTIONAL] - Defaults to false.
	Live bool
	// Now returns the arrival of a push. It must never go back.
	//
	// [REQUIRED] for a live join.
	Now func() telem.TimeStamp
}

// Run is a stretch of joined samples whose alignments touch.
type Run struct {
	// Values holds one series per input, in input order, with one sample per stamp of
	// Time.
	Values []telem.Series
	// Time holds the stamps of the primary input, with the run's alignment and time
	// range.
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
	// Settled is true once the primary of a live join has a frame gap and 3 times the
	// median of its frame gaps has passed from its first frame to its last, as of the
	// last call to Next. Before then, drops come from inputs that are still starting. A
	// history join is always settled.
	Settled bool
}

// Reset empties o and keeps its buffers.
func (o *Output) Reset() {
	o.Runs = o.Runs[:0]
	o.Dropped = 0
	o.Settled = false
}

// Join pairs the samples of its inputs by sample time. It is not safe for concurrent
// use.
type Join struct {
	inputs  []*input
	primary int
	from    telem.TimeStamp
	live    bool
	now     func() telem.TimeStamp
	// frames holds the live frames of the primary, and started the arrival of its
	// first.
	frames  cadence
	started telem.TimeStamp
	// last is the alignment of the last primary sample that got an output, and joined
	// is false when no sample did or a dropped stamp came after it.
	last   telem.Alignment
	joined bool
}

type input struct {
	dataType     telem.DataType
	density      int
	arrivalTimed bool
	// data, stamps, and alignments hold the added samples that Next has not passed,
	// from first on. Only the primary keeps alignments.
	data       []byte
	stamps     []telem.TimeStamp
	alignments []telem.Alignment
	first      int
	// last is the last stamp added to the input, and added is false before the first.
	last  telem.TimeStamp
	added bool
	// held is the last sample at or before the last stamp that Next passed.
	held      []byte
	heldStamp telem.TimeStamp
	holds     bool
	intervals intervals
	frames    cadence
}

// cadence holds the arrivals of the live frames of an input.
type cadence struct {
	gaps    intervals
	arrived telem.TimeStamp
	framed  bool
}

// arrive counts a frame that arrived at at. A frame that arrives with the last one is
// part of it.
func (c *cadence) arrive(at telem.TimeStamp) {
	if c.framed && at != c.arrived {
		c.gaps.add(telem.TimeSpan(at - c.arrived))
	}
	c.arrived, c.framed = at, true
}

// New returns a Join of cfg.
func New(cfg Config) (*Join, error) {
	if len(cfg.Inputs) == 0 {
		return nil, errors.Wrap(validate.ErrValidation, "join needs at least one input")
	}
	if cfg.Primary < 0 || cfg.Primary >= len(cfg.Inputs) {
		return nil, errors.Wrapf(
			validate.ErrValidation,
			"primary input %d is not one of the %d inputs",
			cfg.Primary,
			len(cfg.Inputs),
		)
	}
	if cfg.Live && cfg.Now == nil {
		return nil, errors.Wrap(validate.ErrValidation, "a live join needs Now")
	}
	j := &Join{
		inputs:  make([]*input, len(cfg.Inputs)),
		primary: cfg.Primary,
		from:    cfg.From,
		live:    cfg.Live,
		now:     cfg.Now,
	}
	for i, in := range cfg.Inputs {
		if err := validateInput(cfg, i); err != nil {
			return nil, err
		}
		density := int(in.DataType.Density())
		j.inputs[i] = &input{
			dataType:     in.DataType,
			density:      density,
			arrivalTimed: in.ArrivalTimed,
			held:         make([]byte, density),
		}
	}
	return j, nil
}

// validateInput returns why cfg cannot take its input i, or nil.
func validateInput(cfg Config, i int) error {
	in := cfg.Inputs[i]
	switch {
	case in.DataType.IsVariable() || in.DataType.Density() == 0:
		return errors.Wrapf(
			validate.ErrValidation,
			"input %d has data type %s, which has no fixed density",
			i,
			in.DataType,
		)
	case in.ArrivalTimed && i == cfg.Primary:
		return errors.Wrapf(
			validate.ErrValidation,
			"primary input %d cannot pair by arrival",
			i,
		)
	case in.ArrivalTimed && !cfg.Live:
		return errors.Wrap(validate.ErrValidation, "only a live join pairs by arrival")
	}
	return nil
}

// Load adds stored samples of input i from before a live open, with their stamps in t,
// which must rise. They fill the input's hold limit and held sample, and count as no
// frame. Until the input's first live frame, its held sample is valid for each primary
// stamp within one hold limit of its own stamp. Only a live join loads samples, never
// for an arrival-timed input, and only before the input's first live frame.
func (j *Join) Load(i int, data, t telem.Series) error {
	if err := j.validate(i, data, t); err != nil {
		return err
	}
	in := j.inputs[i]
	switch {
	case !j.live:
		return errors.Wrap(validate.ErrValidation, "only a live join loads samples")
	case in.arrivalTimed:
		return errors.Wrapf(
			validate.ErrValidation,
			"input %d is arrival-timed, so it loads no samples",
			i,
		)
	case in.frames.framed:
		return errors.Wrapf(
			validate.ErrValidation,
			"input %d loads after its first live frame",
			i,
		)
	}
	return j.add(i, data, t, false)
}

// Push adds the samples of input i, with their stamps in t. The samples take the
// alignment of t, not of data. A live join skips each sample whose stamp does not
// follow the input's last, as a stream can repeat the samples that Load added, and it
// counts a push of samples as a frame. A history join refuses such a sample, and keeps
// the input as it was. It also refuses samples that would leave an input other than the
// primary with more than 65,536 samples that Next has not passed, since a history read
// paces its inputs. An arrival-timed input keeps only its newest sample, and Push
// ignores t.
func (j *Join) Push(i int, data, t telem.Series) error {
	if err := j.validate(i, data, t); err != nil {
		return err
	}
	in := j.inputs[i]
	if in.arrivalTimed {
		if data.Len() > 0 {
			copy(in.held, data.Data[len(data.Data)-in.density:])
			in.holds = true
			in.frames.arrive(j.now())
		}
		return nil
	}
	if err := j.add(i, data, t, j.live); err != nil {
		return err
	}
	if !j.live || t.Len() == 0 {
		return nil
	}
	at := j.now()
	if i != j.primary {
		in.frames.arrive(at)
		return nil
	}
	if !j.frames.framed {
		j.started = at
	}
	j.frames.arrive(at)
	return nil
}

// add appends the samples of input i whose stamps follow its last. A stamp that does
// not follow is skipped when skip is true, or refused with the input kept as it was.
func (j *Join) add(i int, data, t telem.Series, skip bool) error {
	in := j.inputs[i]
	stamps, alignments := len(in.stamps), len(in.alignments)
	last, added, skipped := in.last, in.added, false
	for k := range int(t.Len()) {
		stamp := telem.TimeStamp(telem.ByteOrder.Uint64(t.Data[k*8:]))
		if added && stamp <= last {
			if skip {
				skipped = true
				continue
			}
			in.stamps, in.alignments = in.stamps[:stamps], in.alignments[:alignments]
			return errors.Wrapf(
				validate.ErrValidation,
				"stamp %s of input %d does not follow %s",
				stamp,
				i,
				last,
			)
		}
		last, added = stamp, true
		in.stamps = append(in.stamps, stamp)
		if i == j.primary {
			in.alignments = append(in.alignments, t.Alignment+telem.Alignment(k))
		}
	}
	if skipped {
		in.appendFollowing(data, t)
	} else {
		in.data = append(in.data, data.Data...)
	}
	in.last, in.added = last, added
	return nil
}

// appendFollowing appends the samples of data whose stamps in t follow the input's last
// stamp before the call to add.
func (in *input) appendFollowing(data, t telem.Series) {
	last, added := in.last, in.added
	for k := range int(t.Len()) {
		stamp := telem.TimeStamp(telem.ByteOrder.Uint64(t.Data[k*8:]))
		if added && stamp <= last {
			continue
		}
		last, added = stamp, true
		in.data = append(in.data, data.Data[k*in.density:(k+1)*in.density]...)
	}
}

// validate returns why the join refuses the samples data of input i with stamps t, or
// nil.
func (j *Join) validate(i int, data, t telem.Series) error {
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
	case in.arrivalTimed:
		return nil
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
	case !j.live && i != j.primary && len(in.stamps)-in.first+int(t.Len()) > most:
		return errors.Wrapf(
			validate.ErrValidation,
			"input %d would hold more than %d samples that the join has not passed",
			i,
			most,
		)
	case i == j.primary && t.Multiple() != 1:
		return errors.Wrapf(
			validate.ErrValidation,
			"primary input %d steps its alignment by %d, not 1",
			i,
			t.Multiple(),
		)
	}
	return nil
}

// Next appends to out a joined sample for each added primary stamp up to until, and
// sets out.Settled. A live read passes the newest primary stamp, so no input is waited
// for. A history read passes the earliest of the newest stamps of the inputs, so each
// input's value at a stamp is known.
func (j *Join) Next(until telem.TimeStamp, out *Output) {
	p := j.inputs[j.primary]
	for ; p.first < len(p.stamps) && p.stamps[p.first] <= until; p.first++ {
		stamp := p.stamps[p.first]
		j.pass(stamp)
		offset := p.first * p.density
		copy(p.held, p.data[offset:offset+p.density])
		if stamp < j.from {
			continue
		}
		if !j.holdsAll(stamp) {
			out.Dropped++
			j.joined = false
			continue
		}
		j.emit(p.first, out)
	}
	// No later primary stamp comes at or before the primary's last, so the other inputs
	// pass it too, which bounds their buffers across a gap of the primary.
	if p.added {
		j.pass(min(until, p.last))
	}
	for i, in := range j.inputs {
		if j.live && i != j.primary {
			in.bound()
		}
		in.compact()
	}
	gaps := &j.frames.gaps
	out.Settled = !j.live ||
		(gaps.n > 0 && telem.TimeSpan(j.frames.arrived-j.started) >= gaps.within())
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

// step moves the held sample of in to its next added sample.
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

// most is the number of samples past the held one that an input keeps.
const most = 1 << 16

// bound steps in past all but its last most samples. A later primary stamp before the
// held sample gets no output, as past the limit.
func (in *input) bound() {
	for len(in.stamps)-in.first > most {
		in.step()
	}
}

// holdsAll reports whether each other input than the primary holds a valid sample for
// stamp.
func (j *Join) holdsAll(stamp telem.TimeStamp) bool {
	for i, in := range j.inputs {
		if i != j.primary && !j.holds(in, stamp) {
			return false
		}
	}
	return true
}

// holds reports whether in holds a valid sample for stamp.
func (j *Join) holds(in *input, stamp telem.TimeStamp) bool {
	if in.arrivalTimed {
		return in.holds && j.arrivedWithin(in, 0)
	}
	// A sample that bound stepped to can lie after stamp.
	if !in.holds || stamp < in.heldStamp {
		return false
	}
	limit := in.intervals.limit()
	switch {
	case in.intervals.unsteady():
		return true
	case j.live && in.frames.framed && in.first == len(in.stamps):
		return j.arrivedWithin(in, limit)
	case limit == 0:
		return true
	}
	return telem.TimeSpan(stamp-in.heldStamp) <= limit
}

// arrivedWithin reports whether the last frame of in arrived within its window before
// the primary's last frame: the larger of limit and 3 times the median of its frame
// gaps, or of the primary's until it has one. With no limit and no gap, it holds.
func (j *Join) arrivedWithin(in *input, limit telem.TimeSpan) bool {
	gaps := &in.frames.gaps
	if gaps.n == 0 {
		gaps = &j.frames.gaps
	}
	span := max(limit, gaps.within())
	return span == 0 || telem.TimeSpan(j.frames.arrived-in.frames.arrived) <= span
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
	for i, in := range j.inputs {
		run.Values[i] = telem.Series{
			DataType: in.dataType,
			Data:     run.Values[i].Data[:0],
		}
	}
	run.Time = telem.Series{
		DataType:  telem.TimestampT,
		Data:      run.Time.Data[:0],
		Alignment: alignment,
		TimeRange: telem.TimeRange{Start: stamp, End: after(stamp)},
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
