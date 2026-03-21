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
	"encoding/json"
	"fmt"
	"math"

	"github.com/synnaxlabs/x/binary"
)

var (
	_ json.Unmarshaler = (*Alignment)(nil)
	_ json.Marshaler   = (*Alignment)(nil)
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

// UnmarshalJSON implements json.Unmarshaler.
func (a *Alignment) UnmarshalJSON(b []byte) error {
	n, err := binary.UnmarshalJSONStringUint64(b)
	if err != nil {
		return err
	}
	*a = Alignment(n)
	return nil
}

// MarshalJSON implements json.Marshaler.
func (a Alignment) MarshalJSON() ([]byte, error) {
	return binary.MarshalStringUint64(uint64(a))
}

// AddSamples increments the sample index of the alignment.
func (a Alignment) AddSamples(samples uint32) Alignment {
	return NewAlignment(a.DomainIndex(), a.SampleIndex()+samples)
}
