// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"math"
	"sync/atomic"

	xjson "github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/errors"
)

var (
	_ json.UnmarshalerFrom = (*Alignment)(nil)
	_ json.MarshalerTo     = (*Alignment)(nil)
)

// NewAlignment takes the given array index and sample index within that array and
// returns a new Alignment (see Alignment for more information).
func NewAlignment(domainIdx, sampleIdx uint32) Alignment {
	return Alignment(domainIdx)<<32 | Alignment(sampleIdx)
}

// MaxAlignment is the maximum possible value for an alignment.
const MaxAlignment = Alignment(math.MaxUint64)

// DomainIndex returns the domain index of the Alignment. This is the index
// in the array of arrays.
func (a Alignment) DomainIndex() uint32 { return uint32(a >> 32) }

// SampleIndex returns the sample index of the Alignment. This is the index within
// a particular array.
func (a Alignment) SampleIndex() uint32 { return uint32(a) }

// String implements fmt.Stringer to return a nicely formatted string representing the
// alignment.
func (a Alignment) String() string {
	return fmt.Sprintf("%v-%v", a.DomainIndex(), a.SampleIndex())
}

// UnmarshalJSONFrom implements json.UnmarshalerFrom.
func (a *Alignment) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	n, err := xjson.UnmarshalStringUint64From(dec)
	if err != nil {
		return err
	}
	*a = Alignment(n)
	return nil
}

// MarshalJSONTo implements json.MarshalerTo.
func (a Alignment) MarshalJSONTo(enc *jsontext.Encoder) error {
	return xjson.MarshalStringUint64To(enc, uint64(a))
}

// AddSamples increments the sample index of the alignment.
func (a Alignment) AddSamples(samples uint32) Alignment {
	return NewAlignment(a.DomainIndex(), a.SampleIndex()+samples)
}

// LeadingDomain is the first domain index of alignments for written data that is not
// yet stored. Stored data takes the domain indexes below it, so new data orders after
// stored data.
const LeadingDomain uint32 = 1 << 31

// LeadingAlignment returns the alignment of sample sampleIdx in the domain domainIdx
// domains above LeadingDomain.
func LeadingAlignment(domainIdx, sampleIdx uint32) Alignment {
	return NewAlignment(LeadingDomain+domainIdx, sampleIdx)
}

// ErrAlignmentsExhausted is returned by AlignmentCounter.Next when no domain index is
// left above the counter's start.
var ErrAlignmentsExhausted = errors.New("alignment domains exhausted, restart to reset")

// AlignmentCounter hands out alignment domains in increasing order. It is safe for
// concurrent use.
type AlignmentCounter struct{ last atomic.Uint64 }

// NewAlignmentCounter returns an AlignmentCounter whose first domain index is start+1.
func NewAlignmentCounter(start uint32) *AlignmentCounter {
	c := &AlignmentCounter{}
	c.last.Store(uint64(start))
	return c
}

// Next returns the alignment of the first sample in the next domain. It returns
// ErrAlignmentsExhausted once the domain index would pass math.MaxUint32.
func (c *AlignmentCounter) Next() (Alignment, error) {
	domainIdx := c.last.Add(1)
	if domainIdx > math.MaxUint32 {
		return 0, ErrAlignmentsExhausted
	}
	return NewAlignment(uint32(domainIdx), 0), nil
}
