// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package segment

import (
	"encoding/binary"
	"math"

	"github.com/synnaxlabs/x/errors"
)

const (
	// minRun is the fewest stamps of a progression entry.
	minRun = 5
	// maxRaw is the most stamps of a raw entry.
	maxRaw = 8192
	// maxStep is the largest step of a progression entry, in nanoseconds.
	maxStep = math.MaxInt32
	// rawEntry marks the word of a raw entry. The other bits of the word hold the index
	// of its first stamp in the raw area.
	rawEntry   uint32 = 1 << 31
	stampSize         = 8
	entrySize         = 16
	fenceSize         = 12
	footerSize        = 20
)

// entry is an entry of the table of a version 2 segment: the stamp and the position of
// its first sample, and the step of a progression or the raw area index of a raw entry.
type entry struct {
	stamp int64
	pos   uint32
	word  uint32
}

// tableAt returns the stream offset of the first entry of a version 2 segment with nRaw
// raw stamps.
func tableAt(nRaw uint32) int64 { return (stampSize*int64(nRaw) + 15) &^ 15 }

// fenceCount returns the number of blocks in which one of the n entries from stream
// offset at starts.
func fenceCount(at int64, n uint32) int64 {
	if n == 0 {
		return 0
	}
	return (at+entrySize*int64(n-1))/BlockSize - at/BlockSize + 1
}

// encodedSize returns the stream length of a version 2 segment with nRaw raw stamps and
// n entries.
func encodedSize(nRaw, n uint32) int64 {
	end := tableAt(nRaw) + entrySize*int64(n)
	end += fenceSize*fenceCount(tableAt(nRaw), n) + footerSize
	return (end + BlockSize - 1) / BlockSize * BlockSize
}

// splitter splits rising stamps into entries. The longest run from a stamp of minRun
// stamps or more with one step from 1 to maxStep is a progression. Every other stamp is
// raw, in raw entries of at most maxRaw stamps. splitter calls entry for each entry as
// it starts, and raw for each raw stamp, in order.
type splitter struct {
	entry func(entry) error
	raw   func(int64) error
	// pending holds the stamps that may still start a progression, from position at.
	pending [minRun]int64
	n       int
	at      uint32
	// run is the open progression, with its last stamp, while open is true.
	run struct {
		last, step int64
		open       bool
	}
	// rawLeft is the room in the open raw entry, and nRaw counts the raw stamps.
	rawLeft int
	nRaw    uint32
}

func (s *splitter) add(x int64) error {
	if s.run.open {
		if x-s.run.last == s.run.step {
			s.run.last = x
			s.at++
			return nil
		}
		s.run.open = false
	}
	s.pending[s.n] = x
	s.n++
	return s.settle(false)
}

// settle opens a progression when the pending stamps hold one, and makes raw each
// pending stamp that no progression can start at. At the end, each pending stamp is
// raw.
func (s *splitter) settle(end bool) error {
	for s.n > 0 {
		k, step := s.runLength()
		if k == minRun {
			err := s.entry(entry{stamp: s.pending[0], pos: s.at, word: uint32(step)})
			if err != nil {
				return err
			}
			s.run.last, s.run.step, s.run.open = s.pending[minRun-1], step, true
			s.rawLeft = 0
			s.at += minRun
			s.n = 0
			return nil
		}
		if k == s.n && !end {
			return nil
		}
		if err := s.rawStamp(s.pending[0]); err != nil {
			return err
		}
		copy(s.pending[:], s.pending[1:s.n])
		s.n--
		s.at++
	}
	return nil
}

// runLength returns how many pending stamps from the first share one step from 1 to
// maxStep, and the step.
func (s *splitter) runLength() (int, int64) {
	if s.n < 2 {
		return s.n, 0
	}
	step := s.pending[1] - s.pending[0]
	if step < 1 || step > maxStep {
		return 1, 0
	}
	k := 2
	for k < s.n && s.pending[k]-s.pending[k-1] == step {
		k++
	}
	return k, step
}

func (s *splitter) rawStamp(x int64) error {
	if s.rawLeft == 0 {
		if err := s.entry(
			entry{stamp: x, pos: s.at, word: rawEntry | s.nRaw},
		); err != nil {
			return err
		}
		s.rawLeft = maxRaw
	}
	s.rawLeft--
	s.nRaw++
	return s.raw(x)
}

// Encoder writes a sealed clock segment as version 2, through the writer of a new
// segment, in two passes over the same rising stamps. The first pass writes the raw
// area. The second writes the entries, the fences, and the footer, which ends the last
// block. An Encoder is not safe for concurrent use.
type Encoder struct {
	w      *Writer
	limit  int64
	split  splitter
	second bool
	fits   bool
	// last is the last stamp of the pass, and samples counts its stamps.
	last    int64
	samples uint32
	// first holds the counts of the first pass, which the second must match.
	first struct{ samples, entries, raw uint32 }
	// entries counts the entries of the pass.
	entries uint32
	fences  []byte
	buf     []byte
}

// NewEncoder returns an encoder that writes through w, which must hold no bytes yet.
// The encoder fits only while the result takes at most limit stream bytes.
func NewEncoder(w *Writer, limit int64) *Encoder {
	e := &Encoder{w: w, limit: limit, fits: true}
	e.split.entry, e.split.raw = e.countEntry, e.rawStamp
	return e
}

func (e *Encoder) countEntry(entry) error {
	e.entries++
	return nil
}

func (e *Encoder) rawStamp(x int64) error {
	e.buf = binary.LittleEndian.AppendUint64(e.buf, uint64(x))
	return e.flush(BlockSize)
}

func (e *Encoder) writeEntry(en entry) error {
	at := tableAt(e.first.raw) + entrySize*int64(e.entries)
	if at%BlockSize == 0 || e.entries == 0 {
		e.fences = binary.LittleEndian.AppendUint64(e.fences, uint64(en.stamp))
		e.fences = binary.LittleEndian.AppendUint32(e.fences, en.pos)
	}
	e.entries++
	e.buf = binary.LittleEndian.AppendUint64(e.buf, uint64(en.stamp))
	e.buf = binary.LittleEndian.AppendUint32(e.buf, en.pos)
	e.buf = binary.LittleEndian.AppendUint32(e.buf, en.word)
	return e.flush(BlockSize)
}

// flush writes the staged bytes once at least n are staged.
func (e *Encoder) flush(n int) error {
	if len(e.buf) < n {
		return nil
	}
	err := e.w.Write(e.buf)
	e.buf = e.buf[:0]
	return err
}

// Fits reports whether the result fits: every stamp of the first pass rose, and the
// result takes at most the limit of the encoder.
func (e *Encoder) Fits() bool { return e.fits }

// Add adds the stamps in p, each an int64, little-endian, to the pass in progress. In
// the first pass, the encoder stops fitting when a stamp does not rise past the one
// before it, or when the result would pass the limit, and later calls do nothing. Add
// returns an error when a write fails, or when the second pass gets other stamps.
func (e *Encoder) Add(p []byte) error {
	if !e.fits {
		return nil
	}
	if len(p)%stampSize != 0 {
		return errors.Newf(
			"add of %d bytes, which hold no whole number of stamps",
			len(p),
		)
	}
	for i := 0; i < len(p); i += stampSize {
		x := int64(binary.LittleEndian.Uint64(p[i:]))
		if e.samples > 0 && x <= e.last {
			if e.second {
				return errors.Newf(
					"second pass stamp %d does not rise past %d",
					x,
					e.last,
				)
			}
			e.fits = false
			return nil
		}
		e.last = x
		e.samples++
		if err := e.split.add(x); err != nil {
			return err
		}
	}
	if !e.second && encodedSize(e.split.nRaw, e.entries) > e.limit {
		e.fits = false
	}
	return nil
}

// Rewind ends the first pass. When the result fits, it writes the rest of the raw area
// with its padding, and readies the encoder for the second pass. It returns whether the
// result fits.
func (e *Encoder) Rewind() (bool, error) {
	if !e.fits || e.second {
		return false, nil
	}
	if err := e.split.settle(true); err != nil {
		return false, err
	}
	if e.samples == 0 || encodedSize(e.split.nRaw, e.entries) > e.limit {
		e.fits = false
		return false, nil
	}
	if e.split.nRaw%2 != 0 {
		e.buf = append(e.buf, zeroBlock[:stampSize]...)
	}
	if err := e.flush(1); err != nil {
		return false, err
	}
	e.first.samples, e.first.entries, e.first.raw = e.samples, e.entries, e.split.nRaw
	e.split = splitter{entry: e.writeEntry, raw: func(int64) error { return nil }}
	e.second, e.samples, e.entries = true, 0, 0
	return true, nil
}

// Close ends the second pass: it writes the fences, zeros, and the footer, which ends
// the last block. It fails when the second pass did not add the stamps of the first.
func (e *Encoder) Close() error {
	if !e.second {
		return errors.New("close of an encoder before its second pass")
	}
	if err := e.split.settle(true); err != nil {
		return err
	}
	if e.samples != e.first.samples || e.entries != e.first.entries ||
		e.split.nRaw != e.first.raw {
		return errors.Newf("second pass of %d stamps and %d entries, not %d and %d",
			e.samples, e.entries, e.first.samples, e.first.entries)
	}
	e.buf = append(e.buf, e.fences...)
	end := encodedSize(e.first.raw, e.entries) - footerSize
	e.buf = append(e.buf, zeroBlock[:end-e.w.Len()-int64(len(e.buf))]...)
	e.buf = binary.LittleEndian.AppendUint64(e.buf, uint64(e.last))
	e.buf = binary.LittleEndian.AppendUint32(e.buf, e.entries)
	e.buf = binary.LittleEndian.AppendUint32(e.buf, e.samples)
	e.buf = binary.LittleEndian.AppendUint32(e.buf, e.first.raw)
	return e.flush(1)
}
